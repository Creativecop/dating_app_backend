package discovery

import (
	"context"
	"errors"
	"log"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
	ActionLike      = "LIKE"
	ActionPass      = "PASS"
	ActionSuperLike = "SUPER_LIKE"

	defaultActionSource = "DISCOVERY_FEED"
)

type normalizedActionRequest struct {
	TargetUserUUID string
	ActionType     string
	ClientActionID uuid.UUID
}

type actionRecord struct {
	ID                   uint64
	UUID                 string
	ActorUserID          uint64
	TargetUserID         uint64
	TargetUserUUID       string
	ActionType           string
	ClientActionID       string
	TargetDistanceMeters *int
	CreatedAt            time.Time
}

type matchRecord struct {
	ID        uint64
	UUID      string
	MatchedAt time.Time
}

func (s *Service) CreateAction(ctx context.Context, actorUserID uint64, req CreateActionRequest) (*ActionResponse, bool, error) {
	normalized, err := normalizeActionRequest(req)
	if err != nil {
		return nil, false, err
	}

	if existing, err := s.findActionByClientID(ctx, actorUserID, normalized.ClientActionID); err != nil {
		return nil, false, err
	} else if existing != nil {
		if existing.TargetUserUUID != normalized.TargetUserUUID || existing.ActionType != normalized.ActionType {
			return nil, false, idempotencyKeyConflictError()
		}
		response, err := s.actionResponseForPair(ctx, existing.ActionType, actorUserID, existing.TargetUserID)
		return response, response != nil && response.Matched, err
	}

	targetUserID, err := s.userIDByUUIDAny(ctx, normalized.TargetUserUUID)
	if err != nil {
		return nil, false, err
	}
	if actorUserID == targetUserID {
		return nil, false, validationError("Cannot act on yourself", map[string]any{"field": "targetUserUuid"})
	}

	if existing, err := s.findActionByPair(ctx, actorUserID, targetUserID); err != nil {
		return nil, false, err
	} else if existing != nil {
		if existing.ActionType != normalized.ActionType {
			return nil, false, actionAlreadyExistsError()
		}
		response, err := s.actionResponseForPair(ctx, existing.ActionType, actorUserID, targetUserID)
		return response, response != nil && response.Matched, err
	}

	if blocked, err := s.isBlockedEitherDirection(ctx, actorUserID, targetUserID); err != nil {
		return nil, false, err
	} else if blocked {
		return nil, false, actionNotAllowedError()
	}

	target, err := s.CheckTargetDiscoverable(ctx, actorUserID, normalized.TargetUserUUID, DiscoverabilityModeAction)
	if err != nil {
		return nil, false, err
	}

	var result *ActionResponse
	var newMatchUUID string
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		low, high := orderedPair(actorUserID, target.TargetUserID)
		if err := lockPair(ctx, tx, low, high); err != nil {
			return err
		}

		if existing, err := s.findActionByClientIDTx(ctx, tx, actorUserID, normalized.ClientActionID); err != nil {
			return err
		} else if existing != nil {
			if existing.TargetUserID != target.TargetUserID || existing.ActionType != normalized.ActionType {
				return idempotencyKeyConflictError()
			}
			response, err := s.actionResponseForPairTx(ctx, tx, existing.ActionType, actorUserID, existing.TargetUserID)
			if err != nil {
				return err
			}
			result = response
			return nil
		}

		if existing, err := s.findActionByPairTx(ctx, tx, actorUserID, target.TargetUserID); err != nil {
			return err
		} else if existing != nil {
			if existing.ActionType != normalized.ActionType {
				return actionAlreadyExistsError()
			}
			response, err := s.actionResponseForPairTx(ctx, tx, existing.ActionType, actorUserID, target.TargetUserID)
			if err != nil {
				return err
			}
			result = response
			return nil
		}

		actionDate := utcDate(time.Now().UTC())
		if s.usageLimiter != nil && isPositiveAction(normalized.ActionType) {
			if err := s.usageLimiter.ConsumeActionUsageTx(ctx, tx, actorUserID, normalized.ActionType, actionDate); err != nil {
				return err
			}
		}

		if err := tx.Exec(`
			INSERT INTO discovery_actions (
			  actor_user_id,
			  target_user_id,
			  action_type,
			  client_action_id,
			  action_date,
			  source,
			  target_distance_meters
			)
			VALUES (?, ?, ?, ?, ?, ?, ?)
		`,
			actorUserID,
			target.TargetUserID,
			normalized.ActionType,
			normalized.ClientActionID,
			actionDate,
			defaultActionSource,
			target.DistanceMeters,
		).Error; err != nil {
			return err
		}

		if isPositiveAction(normalized.ActionType) {
			hasReciprocal, err := s.hasReciprocalPositiveActionTx(ctx, tx, actorUserID, target.TargetUserID)
			if err != nil {
				return err
			}
			if hasReciprocal {
				matchUUID, created, err := s.ensureMatchTx(ctx, tx, low, high, actorUserID)
				if err != nil {
					return err
				}
				if created {
					newMatchUUID = matchUUID
				}
			}
		}

		response, err := s.actionResponseForPairTx(ctx, tx, normalized.ActionType, actorUserID, target.TargetUserID)
		if err != nil {
			return err
		}
		result = response
		return nil
	})
	if err != nil {
		return nil, false, err
	}
	if newMatchUUID != "" && s.notifications != nil {
		if err := s.notifications.NotifyNewMatch(context.Background(), newMatchUUID); err != nil {
			log.Printf("notification_event=new_match match_uuid=%s error=%v", newMatchUUID, err)
		}
	}
	return result, result != nil && result.Matched, nil
}

func normalizeActionRequest(req CreateActionRequest) (normalizedActionRequest, error) {
	targetUUID, err := uuid.Parse(strings.TrimSpace(req.TargetUserUUID))
	if err != nil {
		return normalizedActionRequest{}, validationError("Target user UUID is invalid", map[string]any{"field": "targetUserUuid"})
	}
	clientActionID, err := uuid.Parse(strings.TrimSpace(req.ClientActionID))
	if err != nil {
		return normalizedActionRequest{}, validationError("clientActionId must be a valid UUID", map[string]any{"field": "clientActionId"})
	}
	actionType := strings.ToUpper(strings.TrimSpace(req.ActionType))
	if !validActionType(actionType) {
		return normalizedActionRequest{}, validationError("Action type is invalid", map[string]any{"field": "actionType"})
	}
	return normalizedActionRequest{
		TargetUserUUID: targetUUID.String(),
		ActionType:     actionType,
		ClientActionID: clientActionID,
	}, nil
}

func validActionType(value string) bool {
	return value == ActionLike || value == ActionPass || value == ActionSuperLike
}

func (s *Service) findActionByClientID(ctx context.Context, actorUserID uint64, clientActionID uuid.UUID) (*actionRecord, error) {
	return s.findActionByClientIDTx(ctx, s.db.WithContext(ctx), actorUserID, clientActionID)
}

func (s *Service) findActionByClientIDTx(ctx context.Context, tx *gorm.DB, actorUserID uint64, clientActionID uuid.UUID) (*actionRecord, error) {
	var row actionRecord
	err := tx.WithContext(ctx).Raw(`
		SELECT
		  da.id,
		  da.uuid::text AS uuid,
		  da.actor_user_id,
		  da.target_user_id,
		  u.uuid::text AS target_user_uuid,
		  da.action_type,
		  da.client_action_id::text AS client_action_id,
		  da.target_distance_meters,
		  da.created_at
		FROM discovery_actions da
		JOIN users u ON u.id = da.target_user_id
		WHERE da.actor_user_id = ?
		  AND da.client_action_id = ?
	`, actorUserID, clientActionID).Scan(&row).Error
	if err != nil {
		return nil, err
	}
	if row.ID == 0 {
		return nil, nil
	}
	return &row, nil
}

func (s *Service) findActionByPair(ctx context.Context, actorUserID uint64, targetUserID uint64) (*actionRecord, error) {
	return s.findActionByPairTx(ctx, s.db.WithContext(ctx), actorUserID, targetUserID)
}

func (s *Service) findActionByPairTx(ctx context.Context, tx *gorm.DB, actorUserID uint64, targetUserID uint64) (*actionRecord, error) {
	var row actionRecord
	err := tx.WithContext(ctx).Raw(`
		SELECT
		  da.id,
		  da.uuid::text AS uuid,
		  da.actor_user_id,
		  da.target_user_id,
		  u.uuid::text AS target_user_uuid,
		  da.action_type,
		  da.client_action_id::text AS client_action_id,
		  da.target_distance_meters,
		  da.created_at
		FROM discovery_actions da
		JOIN users u ON u.id = da.target_user_id
		WHERE da.actor_user_id = ?
		  AND da.target_user_id = ?
	`, actorUserID, targetUserID).Scan(&row).Error
	if err != nil {
		return nil, err
	}
	if row.ID == 0 {
		return nil, nil
	}
	return &row, nil
}

func (s *Service) actionResponseForPair(ctx context.Context, actionType string, actorUserID uint64, targetUserID uint64) (*ActionResponse, error) {
	return s.actionResponseForPairTx(ctx, s.db.WithContext(ctx), actionType, actorUserID, targetUserID)
}

func (s *Service) actionResponseForPairTx(ctx context.Context, tx *gorm.DB, actionType string, actorUserID uint64, targetUserID uint64) (*ActionResponse, error) {
	match, err := s.activeMatchForPairTx(ctx, tx, actorUserID, targetUserID)
	if err != nil {
		return nil, err
	}
	response := &ActionResponse{ActionType: actionType}
	if match != nil {
		response.Matched = true
		response.Match = &MatchResponse{MatchUUID: match.UUID, MatchedAt: match.MatchedAt}
	}
	return response, nil
}

func (s *Service) hasReciprocalPositiveActionTx(ctx context.Context, tx *gorm.DB, actorUserID uint64, targetUserID uint64) (bool, error) {
	var exists bool
	err := tx.WithContext(ctx).Raw(`
		SELECT EXISTS (
		  SELECT 1
		  FROM discovery_actions
		  WHERE actor_user_id = ?
		    AND target_user_id = ?
		    AND action_type IN ('LIKE', 'SUPER_LIKE')
		)
	`, targetUserID, actorUserID).Scan(&exists).Error
	return exists, err
}

func (s *Service) ensureMatchTx(ctx context.Context, tx *gorm.DB, userLowID uint64, userHighID uint64, initiatedByUserID uint64) (string, bool, error) {
	now := time.Now().UTC()
	var inserted matchRecord
	if err := tx.WithContext(ctx).Raw(`
		INSERT INTO matches (
		  user_low_id,
		  user_high_id,
		  initiated_by_user_id,
		  status,
		  matched_at,
		  created_at,
		  updated_at
		)
		VALUES (?, ?, ?, 'ACTIVE', ?, ?, ?)
		ON CONFLICT (user_low_id, user_high_id) DO NOTHING
		RETURNING id, uuid::text AS uuid, matched_at
	`, userLowID, userHighID, initiatedByUserID, now, now, now).Scan(&inserted).Error; err != nil {
		return "", false, err
	}

	var matchID uint64
	matchUUID := inserted.UUID
	created := inserted.ID != 0
	if created {
		matchID = inserted.ID
	} else if err := tx.WithContext(ctx).Raw(`
		SELECT id, uuid::text AS uuid
		FROM matches
		WHERE user_low_id = ?
		  AND user_high_id = ?
	`, userLowID, userHighID).Scan(&inserted).Error; err != nil {
		return "", false, err
	} else {
		matchID = inserted.ID
		matchUUID = inserted.UUID
	}
	if matchID == 0 {
		return "", false, nil
	}
	if err := tx.WithContext(ctx).Exec(`
		INSERT INTO match_participants (match_id, user_id, created_at, updated_at)
		VALUES (?, ?, ?, ?), (?, ?, ?, ?)
		ON CONFLICT (match_id, user_id) DO NOTHING
	`, matchID, userLowID, now, now, matchID, userHighID, now, now).Error; err != nil {
		return "", false, err
	}

	var conversationID uint64
	if err := tx.WithContext(ctx).Raw(`
		INSERT INTO conversations (match_id, status, created_at, updated_at)
		VALUES (?, 'ACTIVE', ?, ?)
		ON CONFLICT (match_id) DO UPDATE
		SET status = CASE
		      WHEN conversations.status = 'CLOSED' THEN conversations.status
		      ELSE 'ACTIVE'
		    END,
		    updated_at = EXCLUDED.updated_at
		RETURNING id
	`, matchID, now, now).Scan(&conversationID).Error; err != nil {
		return "", false, err
	}
	if conversationID == 0 {
		if err := tx.WithContext(ctx).Raw("SELECT id FROM conversations WHERE match_id = ?", matchID).Scan(&conversationID).Error; err != nil {
			return "", false, err
		}
	}
	if conversationID == 0 {
		return matchUUID, created, nil
	}
	if err := tx.WithContext(ctx).Exec(`
		INSERT INTO conversation_participants (conversation_id, user_id, created_at, updated_at)
		VALUES (?, ?, ?, ?), (?, ?, ?, ?)
		ON CONFLICT (conversation_id, user_id) DO NOTHING
	`, conversationID, userLowID, now, now, conversationID, userHighID, now, now).Error; err != nil {
		return "", false, err
	}
	return matchUUID, created, nil
}

func (s *Service) activeMatchForPairTx(ctx context.Context, tx *gorm.DB, userAID uint64, userBID uint64) (*matchRecord, error) {
	low, high := orderedPair(userAID, userBID)
	var row matchRecord
	err := tx.WithContext(ctx).Raw(`
		SELECT id, uuid::text AS uuid, matched_at
		FROM matches
		WHERE user_low_id = ?
		  AND user_high_id = ?
		  AND status = 'ACTIVE'
	`, low, high).Scan(&row).Error
	if err != nil {
		return nil, err
	}
	if row.ID == 0 {
		return nil, nil
	}
	return &row, nil
}

func (s *Service) userIDByUUIDAny(ctx context.Context, rawUUID string) (uint64, error) {
	parsed, err := uuid.Parse(strings.TrimSpace(rawUUID))
	if err != nil {
		return 0, validationError("User UUID is invalid", map[string]any{"field": "userUuid"})
	}
	var id uint64
	err = s.db.WithContext(ctx).Raw("SELECT id FROM users WHERE uuid = ?", parsed).Scan(&id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return 0, notFoundError("User not found")
	}
	if err != nil {
		return 0, err
	}
	if id == 0 {
		return 0, notFoundError("User not found")
	}
	return id, nil
}

func (s *Service) isBlockedEitherDirection(ctx context.Context, actorUserID uint64, targetUserID uint64) (bool, error) {
	if s.blockChecker != nil {
		return s.blockChecker.IsBlockedEitherDirection(ctx, actorUserID, targetUserID)
	}
	var exists bool
	err := s.db.WithContext(ctx).Raw(`
		SELECT EXISTS (
		  SELECT 1
		  FROM user_blocks
		  WHERE (blocker_user_id = ? AND blocked_user_id = ?)
		     OR (blocker_user_id = ? AND blocked_user_id = ?)
		)
	`, actorUserID, targetUserID, targetUserID, actorUserID).Scan(&exists).Error
	return exists, err
}

func orderedPair(a uint64, b uint64) (uint64, uint64) {
	if a < b {
		return a, b
	}
	return b, a
}

func lockPair(ctx context.Context, tx *gorm.DB, low uint64, high uint64) error {
	return tx.WithContext(ctx).Exec("SELECT pg_advisory_xact_lock(?::int, ?::int)", int64(low), int64(high)).Error
}

func isPositiveAction(actionType string) bool {
	return actionType == ActionLike || actionType == ActionSuperLike
}

func utcDate(now time.Time) time.Time {
	return time.Date(now.UTC().Year(), now.UTC().Month(), now.UTC().Day(), 0, 0, 0, 0, time.UTC)
}

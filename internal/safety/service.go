package safety

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Service struct {
	db       *gorm.DB
	adminOps AdminOperations
}

type reasonRow struct {
	ID            uint64
	ReasonCode    string
	Title         string
	Description   *string
	AppliesToJSON string
	SortOrder     int
	IsActive      bool
}

type reportRow struct {
	ID               uint64
	UUID             string
	CaseUUID         string
	Status           string
	TargetType       string
	ReasonCode       string
	CreatedAt        time.Time
	ModerationCaseID *uint64
}

type blockRow struct {
	UserUUID    string
	DisplayName *string
	BlockedAt   time.Time
	ReasonCode  *string
	Source      string
}

type targetSnapshot struct {
	TargetType       string
	TargetUUID       uuid.UUID
	ReportedUserID   uint64
	ReportedUserUUID string
	Evidence         map[string]any
}

func NewService(db *gorm.DB) *Service {
	return &Service{db: db}
}

func (s *Service) SetAdminOperations(ops AdminOperations) {
	s.adminOps = ops
}

func (s *Service) EnsureSafetySettings(ctx context.Context, userID uint64) error {
	now := time.Now().UTC()
	settings := SafetySettings{
		UUID:                 uuid.New(),
		UserID:               userID,
		AllowMessageRequests: true,
		AutoHideBlockedUsers: true,
		CreatedAt:            now,
		UpdatedAt:            now,
	}
	return s.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_id"}},
		DoNothing: true,
	}).Create(&settings).Error
}

func (s *Service) ListReasons(ctx context.Context) (*ReportReasonsResponse, error) {
	var rows []reasonRow
	if err := s.db.WithContext(ctx).Raw(`
		SELECT
		  reason_code,
		  title,
		  description,
		  array_to_json(applies_to)::text AS applies_to_json
		FROM report_reasons
		WHERE is_active = TRUE
		ORDER BY sort_order ASC, title ASC
	`).Scan(&rows).Error; err != nil {
		return nil, err
	}
	response := &ReportReasonsResponse{Items: make([]ReportReasonResponse, 0, len(rows))}
	for _, row := range rows {
		response.Items = append(response.Items, ReportReasonResponse{
			ReasonCode:  row.ReasonCode,
			Title:       row.Title,
			Description: row.Description,
			AppliesTo:   decodeStringArray(row.AppliesToJSON),
		})
	}
	return response, nil
}

func (s *Service) CreateReport(ctx context.Context, reporterID uint64, req CreateReportRequest) (*CreateReportResponse, error) {
	normalized, err := normalizeReportRequest(req)
	if err != nil {
		return nil, err
	}
	reason, err := s.reasonByCode(ctx, normalized.ReasonCode)
	if err != nil {
		return nil, err
	}
	if !reason.IsActive || !reasonApplies(reason.AppliesToJSON, normalized.TargetType) {
		return nil, validationError("Report reason is not valid for this target", map[string]any{"field": "reasonCode"})
	}
	target, err := s.validateReportTargetAccess(ctx, reporterID, normalized.TargetType, normalized.TargetUUID)
	if err != nil {
		return nil, err
	}
	if target.ReportedUserID == reporterID {
		return nil, validationError("Cannot report yourself", map[string]any{"field": "targetUuid"})
	}

	evidence, err := json.Marshal(target.Evidence)
	if err != nil {
		return nil, err
	}
	severity := SeverityForReason(normalized.ReasonCode)

	var result CreateReportResponse
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		existing, err := s.openReportTx(ctx, tx, reporterID, target.TargetType, target.TargetUUID, reason.ID)
		if err != nil {
			return err
		}
		if existing != nil {
			result = CreateReportResponse{ReportUUID: existing.UUID, CaseUUID: existing.CaseUUID, Status: existing.Status}
			if normalized.BlockUser {
				blocked, err := s.blockUserTx(ctx, tx, reporterID, target.ReportedUserID, normalized.ReasonCode, normalized.Note, BlockSourceReportFlow, &existing.ID)
				if err != nil {
					return err
				}
				result.Blocked = blocked
			}
			return nil
		}

		caseID, caseUUID, err := s.findOrCreateCaseTx(ctx, tx, &target.ReportedUserID, target.TargetType, target.TargetUUID)
		if err != nil {
			return err
		}
		reportID, reportUUID, err := s.insertReportTx(ctx, tx, reporterID, target, reason.ID, normalized.Note, evidence, severity, caseID)
		if err != nil {
			return err
		}
		if err := s.incrementCaseReportCountTx(ctx, tx, caseID); err != nil {
			return err
		}
		if err := s.createModerationActionTx(ctx, tx, caseID, &reporterID, &target.ReportedUserID, ActionReportCreated, normalized.ReasonCode, map[string]any{
			"reportUuid": reportUUID,
			"targetType": target.TargetType,
			"targetUuid": target.TargetUUID.String(),
			"severity":   severity,
		}); err != nil {
			return err
		}
		blocked := false
		if normalized.BlockUser {
			blocked, err = s.blockUserTx(ctx, tx, reporterID, target.ReportedUserID, normalized.ReasonCode, normalized.Note, BlockSourceReportFlow, &reportID)
			if err != nil {
				return err
			}
			if blocked {
				if err := s.createModerationActionTx(ctx, tx, caseID, &reporterID, &target.ReportedUserID, ActionUserBlocked, normalized.ReasonCode, map[string]any{
					"reportUuid": reportUUID,
					"source":     BlockSourceReportFlow,
				}); err != nil {
					return err
				}
			}
		}
		result = CreateReportResponse{ReportUUID: reportUUID, CaseUUID: caseUUID, Status: ReportPending, Blocked: blocked}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func (s *Service) MyReports(ctx context.Context, reporterID uint64) (*MyReportsResponse, error) {
	var rows []reportRow
	if err := s.db.WithContext(ctx).Raw(`
		SELECT
		  r.uuid::text AS uuid,
		  r.target_type,
		  rr.reason_code,
		  r.status,
		  r.created_at
		FROM reports r
		JOIN report_reasons rr ON rr.id = r.reason_id
		WHERE r.reporter_user_id = ?
		ORDER BY r.created_at DESC, r.id DESC
	`, reporterID).Scan(&rows).Error; err != nil {
		return nil, err
	}
	response := &MyReportsResponse{Items: make([]MyReportResponse, 0, len(rows))}
	for _, row := range rows {
		response.Items = append(response.Items, MyReportResponse{
			ReportUUID: row.UUID,
			TargetType: row.TargetType,
			ReasonCode: row.ReasonCode,
			Status:     row.Status,
			CreatedAt:  row.CreatedAt,
		})
	}
	return response, nil
}

func (s *Service) BlockUser(ctx context.Context, blockerID uint64, targetUserUUID string, req BlockUserRequest, source string, reportID *uint64) error {
	targetID, err := s.userIDByUUID(ctx, targetUserUUID)
	if err != nil {
		return err
	}
	if blockerID == targetID {
		return validationError("Cannot block yourself", map[string]any{"field": "userUuid"})
	}
	normalized, err := normalizeBlockRequest(req, source)
	if err != nil {
		return err
	}
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		_, err := s.blockUserTx(ctx, tx, blockerID, targetID, normalized.ReasonCode, normalized.Note, normalized.Source, reportID)
		return err
	})
}

func (s *Service) UnblockUser(ctx context.Context, blockerID uint64, targetUserUUID string) error {
	targetID, err := s.userIDByUUID(ctx, targetUserUUID)
	if err != nil {
		return err
	}
	return s.db.WithContext(ctx).Exec(`
		DELETE FROM user_blocks
		WHERE blocker_user_id = ?
		  AND blocked_user_id = ?
	`, blockerID, targetID).Error
}

func (s *Service) ListBlocks(ctx context.Context, blockerID uint64) (*BlockListResponse, error) {
	var rows []blockRow
	if err := s.db.WithContext(ctx).Raw(`
		SELECT
		  u.uuid::text AS user_uuid,
		  p.display_name,
		  ub.created_at AS blocked_at,
		  ub.reason_code,
		  ub.source
		FROM user_blocks ub
		JOIN users u ON u.id = ub.blocked_user_id
		LEFT JOIN profiles p ON p.user_id = u.id
		WHERE ub.blocker_user_id = ?
		ORDER BY ub.created_at DESC
	`, blockerID).Scan(&rows).Error; err != nil {
		return nil, err
	}
	response := &BlockListResponse{Items: make([]BlockedUserResponse, 0, len(rows))}
	for _, row := range rows {
		response.Items = append(response.Items, BlockedUserResponse{
			UserUUID:    row.UserUUID,
			DisplayName: row.DisplayName,
			BlockedAt:   row.BlockedAt,
			ReasonCode:  row.ReasonCode,
			Source:      row.Source,
		})
	}
	return response, nil
}

func (s *Service) IsBlockedEitherDirection(ctx context.Context, userAID uint64, userBID uint64) (bool, error) {
	var exists bool
	err := s.db.WithContext(ctx).Raw(`
		SELECT EXISTS (
		  SELECT 1
		  FROM user_blocks
		  WHERE (blocker_user_id = ? AND blocked_user_id = ?)
		     OR (blocker_user_id = ? AND blocked_user_id = ?)
		)
	`, userAID, userBID, userBID, userAID).Scan(&exists).Error
	return exists, err
}

func (s *Service) ValidateReportTargetAccess(ctx context.Context, reporterID uint64, targetType string, rawTargetUUID string) (*ReportTargetSnapshot, error) {
	normalizedType := strings.ToUpper(strings.TrimSpace(targetType))
	if !validTargetType(normalizedType) {
		return nil, validationError("targetType is invalid", map[string]any{"field": "targetType"})
	}
	targetUUID, err := uuid.Parse(strings.TrimSpace(rawTargetUUID))
	if err != nil {
		return nil, validationError("targetUuid is invalid", map[string]any{"field": "targetUuid"})
	}
	snapshot, err := s.validateReportTargetAccess(ctx, reporterID, normalizedType, targetUUID)
	if err != nil {
		return nil, err
	}
	return &ReportTargetSnapshot{
		TargetType:       snapshot.TargetType,
		TargetUUID:       snapshot.TargetUUID.String(),
		ReportedUserID:   snapshot.ReportedUserID,
		ReportedUserUUID: snapshot.ReportedUserUUID,
		Evidence:         snapshot.Evidence,
	}, nil
}

func (s *Service) GetSettings(ctx context.Context, userID uint64) (*SafetySettingsResponse, error) {
	if err := s.EnsureSafetySettings(ctx, userID); err != nil {
		return nil, err
	}
	var row SafetySettings
	if err := s.db.WithContext(ctx).Where("user_id = ?", userID).First(&row).Error; err != nil {
		return nil, err
	}
	response := settingsResponse(row)
	return &response, nil
}

func (s *Service) UpdateSettings(ctx context.Context, userID uint64, req UpdateSafetySettingsRequest) (*SafetySettingsResponse, error) {
	if req.AllowMessageRequests == nil {
		return nil, validationError("allowMessageRequests is required", map[string]any{"field": "allowMessageRequests"})
	}
	if req.AutoHideBlockedUsers == nil {
		return nil, validationError("autoHideBlockedUsers is required", map[string]any{"field": "autoHideBlockedUsers"})
	}
	if err := s.EnsureSafetySettings(ctx, userID); err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	if err := s.db.WithContext(ctx).Model(&SafetySettings{}).
		Where("user_id = ?", userID).
		Updates(map[string]any{
			"allow_message_requests":  *req.AllowMessageRequests,
			"auto_hide_blocked_users": *req.AutoHideBlockedUsers,
			"updated_at":              now,
		}).Error; err != nil {
		return nil, err
	}
	return s.GetSettings(ctx, userID)
}

func (s *Service) reasonByCode(ctx context.Context, code string) (*reasonRow, error) {
	var row reasonRow
	if err := s.db.WithContext(ctx).Raw(`
		SELECT
		  id,
		  reason_code,
		  title,
		  description,
		  array_to_json(applies_to)::text AS applies_to_json,
		  sort_order,
		  is_active
		FROM report_reasons
		WHERE reason_code = ?
	`, code).Scan(&row).Error; err != nil {
		return nil, err
	}
	if row.ID == 0 {
		return nil, validationError("Report reason is invalid", map[string]any{"field": "reasonCode"})
	}
	return &row, nil
}

func (s *Service) validateReportTargetAccess(ctx context.Context, reporterID uint64, targetType string, targetUUID uuid.UUID) (targetSnapshot, error) {
	switch targetType {
	case TargetUser:
		return s.userTarget(ctx, reporterID, targetUUID)
	case TargetProfile:
		return s.profileTarget(ctx, reporterID, targetUUID)
	case TargetMessage:
		return s.messageTarget(ctx, reporterID, targetUUID)
	case TargetMedia:
		return s.mediaTarget(ctx, reporterID, targetUUID)
	case TargetMatch:
		return s.matchTarget(ctx, reporterID, targetUUID)
	default:
		return targetSnapshot{}, validationError("targetType is invalid", map[string]any{"field": "targetType"})
	}
}

func (s *Service) userTarget(ctx context.Context, reporterID uint64, targetUUID uuid.UUID) (targetSnapshot, error) {
	type row struct {
		UserID      uint64
		UserUUID    string
		DisplayName *string
		Bio         *string
	}
	var item row
	if err := s.db.WithContext(ctx).Raw(`
		SELECT u.id AS user_id, u.uuid::text AS user_uuid, p.display_name, p.bio
		FROM users u
		LEFT JOIN profiles p ON p.user_id = u.id
		WHERE u.uuid = ?
		  AND u.deleted_at IS NULL
	`, targetUUID).Scan(&item).Error; err != nil {
		return targetSnapshot{}, err
	}
	if item.UserID == 0 {
		return targetSnapshot{}, notFoundError("Report target not found")
	}
	if reporterID == item.UserID {
		return targetSnapshot{}, validationError("Cannot report yourself", map[string]any{"field": "targetUuid"})
	}
	if ok, err := s.hasUserContext(ctx, reporterID, item.UserID); err != nil {
		return targetSnapshot{}, err
	} else if !ok {
		return targetSnapshot{}, forbiddenError("You cannot report this target")
	}
	return targetSnapshot{
		TargetType:       TargetUser,
		TargetUUID:       targetUUID,
		ReportedUserID:   item.UserID,
		ReportedUserUUID: item.UserUUID,
		Evidence: map[string]any{
			"reportedUserUuid": item.UserUUID,
			"displayName":      item.DisplayName,
			"bio":              item.Bio,
		},
	}, nil
}

func (s *Service) profileTarget(ctx context.Context, reporterID uint64, targetUUID uuid.UUID) (targetSnapshot, error) {
	type row struct {
		ProfileUUID   string
		UserID        uint64
		UserUUID      string
		DisplayName   *string
		Bio           *string
		ProfileStatus string
	}
	var item row
	if err := s.db.WithContext(ctx).Raw(`
		SELECT p.uuid::text AS profile_uuid, u.id AS user_id, u.uuid::text AS user_uuid, p.display_name, p.bio, p.profile_status
		FROM profiles p
		JOIN users u ON u.id = p.user_id
		WHERE p.uuid = ?
		  AND u.deleted_at IS NULL
	`, targetUUID).Scan(&item).Error; err != nil {
		return targetSnapshot{}, err
	}
	if item.UserID == 0 {
		return targetSnapshot{}, notFoundError("Report target not found")
	}
	if reporterID == item.UserID {
		return targetSnapshot{}, validationError("Cannot report yourself", map[string]any{"field": "targetUuid"})
	}
	if ok, err := s.hasUserContext(ctx, reporterID, item.UserID); err != nil {
		return targetSnapshot{}, err
	} else if !ok {
		return targetSnapshot{}, forbiddenError("You cannot report this target")
	}
	return targetSnapshot{
		TargetType:       TargetProfile,
		TargetUUID:       targetUUID,
		ReportedUserID:   item.UserID,
		ReportedUserUUID: item.UserUUID,
		Evidence: map[string]any{
			"profileUuid":      item.ProfileUUID,
			"reportedUserUuid": item.UserUUID,
			"displayName":      item.DisplayName,
			"bio":              item.Bio,
			"profileStatus":    item.ProfileStatus,
		},
	}, nil
}

func (s *Service) messageTarget(ctx context.Context, reporterID uint64, targetUUID uuid.UUID) (targetSnapshot, error) {
	type row struct {
		MessageUUID      string
		MessageBody      *string
		MessageType      string
		MessageCreatedAt time.Time
		SenderUserID     uint64
		SenderUserUUID   string
		ConversationUUID string
	}
	var item row
	if err := s.db.WithContext(ctx).Raw(`
		SELECT
		  m.uuid::text AS message_uuid,
		  m.body AS message_body,
		  m.message_type,
		  m.created_at AS message_created_at,
		  m.sender_user_id,
		  sender.uuid::text AS sender_user_uuid,
		  c.uuid::text AS conversation_uuid
		FROM messages m
		JOIN conversations c ON c.id = m.conversation_id
		JOIN conversation_participants cp ON cp.conversation_id = c.id AND cp.user_id = ?
		JOIN users sender ON sender.id = m.sender_user_id
		WHERE m.uuid = ?
	`, reporterID, targetUUID).Scan(&item).Error; err != nil {
		return targetSnapshot{}, err
	}
	if item.SenderUserID == 0 {
		return targetSnapshot{}, notFoundError("Report target not found")
	}
	if item.SenderUserID == reporterID {
		return targetSnapshot{}, validationError("Cannot report your own message", map[string]any{"field": "targetUuid"})
	}
	return targetSnapshot{
		TargetType:       TargetMessage,
		TargetUUID:       targetUUID,
		ReportedUserID:   item.SenderUserID,
		ReportedUserUUID: item.SenderUserUUID,
		Evidence: map[string]any{
			"messageUuid":      item.MessageUUID,
			"senderUserUuid":   item.SenderUserUUID,
			"conversationUuid": item.ConversationUUID,
			"messageType":      item.MessageType,
			"body":             item.MessageBody,
			"createdAt":        item.MessageCreatedAt,
		},
	}, nil
}

func (s *Service) mediaTarget(ctx context.Context, reporterID uint64, targetUUID uuid.UUID) (targetSnapshot, error) {
	type row struct {
		MediaUUID          string
		OwnerUserID        uint64
		OwnerUserUUID      string
		MediaType          string
		MediaPurpose       string
		ProcessingStatus   string
		ModerationStatus   string
		IsPrimary          bool
		UploadedAt         time.Time
		DisplayObjectKey   *string
		ThumbnailObjectKey *string
	}
	var item row
	if err := s.db.WithContext(ctx).Raw(`
		SELECT
		  m.uuid::text AS media_uuid,
		  m.user_id AS owner_user_id,
		  u.uuid::text AS owner_user_uuid,
		  m.media_type,
		  m.media_purpose,
		  m.processing_status,
		  m.moderation_status,
		  m.is_primary,
		  m.uploaded_at,
		  display.object_key AS display_object_key,
		  thumb.object_key AS thumbnail_object_key
		FROM user_media m
		JOIN users u ON u.id = m.user_id
		LEFT JOIN user_media_variants display ON display.media_id = m.id AND display.variant_type = 'DISPLAY'
		LEFT JOIN user_media_variants thumb ON thumb.media_id = m.id AND thumb.variant_type = 'THUMBNAIL'
		WHERE m.uuid = ?
		  AND m.deleted_at IS NULL
	`, targetUUID).Scan(&item).Error; err != nil {
		return targetSnapshot{}, err
	}
	if item.OwnerUserID == 0 {
		return targetSnapshot{}, notFoundError("Report target not found")
	}
	if item.OwnerUserID == reporterID {
		return targetSnapshot{}, validationError("Cannot report your own media", map[string]any{"field": "targetUuid"})
	}
	if ok, err := s.hasUserContext(ctx, reporterID, item.OwnerUserID); err != nil {
		return targetSnapshot{}, err
	} else if !ok {
		return targetSnapshot{}, forbiddenError("You cannot report this media")
	}
	return targetSnapshot{
		TargetType:       TargetMedia,
		TargetUUID:       targetUUID,
		ReportedUserID:   item.OwnerUserID,
		ReportedUserUUID: item.OwnerUserUUID,
		Evidence: map[string]any{
			"mediaUuid":          item.MediaUUID,
			"ownerUserUuid":      item.OwnerUserUUID,
			"mediaType":          item.MediaType,
			"mediaPurpose":       item.MediaPurpose,
			"processingStatus":   item.ProcessingStatus,
			"moderationStatus":   item.ModerationStatus,
			"isPrimary":          item.IsPrimary,
			"uploadedAt":         item.UploadedAt,
			"displayObjectKey":   item.DisplayObjectKey,
			"thumbnailObjectKey": item.ThumbnailObjectKey,
		},
	}, nil
}

func (s *Service) matchTarget(ctx context.Context, reporterID uint64, targetUUID uuid.UUID) (targetSnapshot, error) {
	type row struct {
		MatchID      uint64
		MatchUUID    string
		UserLowID    uint64
		UserHighID   uint64
		UserLowUUID  string
		UserHighUUID string
		MatchedAt    time.Time
		Status       string
	}
	var item row
	if err := s.db.WithContext(ctx).Raw(`
		SELECT
		  m.id AS match_id,
		  m.uuid::text AS match_uuid,
		  m.user_low_id,
		  m.user_high_id,
		  low_u.uuid::text AS user_low_uuid,
		  high_u.uuid::text AS user_high_uuid,
		  m.matched_at,
		  m.status
		FROM matches m
		JOIN match_participants mp ON mp.match_id = m.id AND mp.user_id = ?
		JOIN users low_u ON low_u.id = m.user_low_id
		JOIN users high_u ON high_u.id = m.user_high_id
		WHERE m.uuid = ?
	`, reporterID, targetUUID).Scan(&item).Error; err != nil {
		return targetSnapshot{}, err
	}
	if item.MatchID == 0 {
		return targetSnapshot{}, notFoundError("Report target not found")
	}
	reportedID := item.UserHighID
	reportedUUID := item.UserHighUUID
	if reporterID == item.UserHighID {
		reportedID = item.UserLowID
		reportedUUID = item.UserLowUUID
	}
	return targetSnapshot{
		TargetType:       TargetMatch,
		TargetUUID:       targetUUID,
		ReportedUserID:   reportedID,
		ReportedUserUUID: reportedUUID,
		Evidence: map[string]any{
			"matchUuid":    item.MatchUUID,
			"userLowUuid":  item.UserLowUUID,
			"userHighUuid": item.UserHighUUID,
			"matchedAt":    item.MatchedAt,
			"status":       item.Status,
		},
	}, nil
}

func (s *Service) hasUserContext(ctx context.Context, reporterID uint64, targetID uint64) (bool, error) {
	var exists bool
	err := s.db.WithContext(ctx).Raw(`
		SELECT EXISTS (
		  SELECT 1
		  FROM discovery_impressions di
		  WHERE di.viewer_user_id = ?
		    AND di.candidate_user_id = ?
		)
		OR EXISTS (
		  SELECT 1
		  FROM discovery_actions da
		  WHERE (da.actor_user_id = ? AND da.target_user_id = ?)
		     OR (da.actor_user_id = ? AND da.target_user_id = ?)
		)
		OR EXISTS (
		  SELECT 1
		  FROM matches m
		  WHERE (m.user_low_id = LEAST(?::bigint, ?::bigint)
		     AND m.user_high_id = GREATEST(?::bigint, ?::bigint))
		)
	`, reporterID, targetID, reporterID, targetID, targetID, reporterID, reporterID, targetID, reporterID, targetID).Scan(&exists).Error
	return exists, err
}

func (s *Service) openReportTx(ctx context.Context, tx *gorm.DB, reporterID uint64, targetType string, targetUUID uuid.UUID, reasonID uint64) (*reportRow, error) {
	var row reportRow
	err := tx.WithContext(ctx).Raw(`
		SELECT
		  r.id,
		  r.uuid::text AS uuid,
		  mc.uuid::text AS case_uuid,
		  r.status,
		  r.moderation_case_id
		FROM reports r
		LEFT JOIN moderation_cases mc ON mc.id = r.moderation_case_id
		WHERE r.reporter_user_id = ?
		  AND r.target_type = ?
		  AND r.target_uuid = ?
		  AND r.reason_id = ?
		  AND r.status = 'PENDING'
		LIMIT 1
	`, reporterID, targetType, targetUUID, reasonID).Scan(&row).Error
	if err != nil {
		return nil, err
	}
	if row.ID == 0 {
		return nil, nil
	}
	return &row, nil
}

func (s *Service) findOrCreateCaseTx(ctx context.Context, tx *gorm.DB, subjectUserID *uint64, targetType string, targetUUID uuid.UUID) (uint64, string, error) {
	var existing struct {
		ID   uint64
		UUID string
	}
	if err := tx.WithContext(ctx).Raw(`
		SELECT id, uuid::text AS uuid
		FROM moderation_cases
		WHERE subject_user_id = ?
		  AND target_type = ?
		  AND target_uuid = ?
		  AND status IN ('OPEN', 'IN_REVIEW')
		FOR UPDATE
	`, subjectUserID, targetType, targetUUID).Scan(&existing).Error; err != nil {
		return 0, "", err
	}
	if existing.ID != 0 {
		return existing.ID, existing.UUID, nil
	}
	now := time.Now().UTC()
	var inserted struct {
		ID   uint64
		UUID string
	}
	if err := tx.WithContext(ctx).Raw(`
		INSERT INTO moderation_cases (
		  uuid,
		  subject_user_id,
		  target_type,
		  target_uuid,
		  status,
		  priority,
		  report_count,
		  opened_at,
		  created_at,
		  updated_at
		)
		VALUES (?, ?, ?, ?, 'OPEN', 'NORMAL', 0, ?, ?, ?)
		RETURNING id, uuid::text AS uuid
	`, uuid.New(), subjectUserID, targetType, targetUUID, now, now, now).Scan(&inserted).Error; err != nil {
		return 0, "", err
	}
	return inserted.ID, inserted.UUID, nil
}

func (s *Service) insertReportTx(ctx context.Context, tx *gorm.DB, reporterID uint64, target targetSnapshot, reasonID uint64, note *string, evidence []byte, severity string, caseID uint64) (uint64, string, error) {
	now := time.Now().UTC()
	var inserted struct {
		ID   uint64
		UUID string
	}
	if err := tx.WithContext(ctx).Raw(`
		INSERT INTO reports (
		  uuid,
		  reporter_user_id,
		  reported_user_id,
		  target_type,
		  target_uuid,
		  reason_id,
		  note,
		  evidence_snapshot,
		  status,
		  severity,
		  moderation_case_id,
		  created_at,
		  updated_at
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?::jsonb, 'PENDING', ?, ?, ?, ?)
		RETURNING id, uuid::text AS uuid
	`, uuid.New(), reporterID, target.ReportedUserID, target.TargetType, target.TargetUUID, reasonID, note, string(evidence), severity, caseID, now, now).Scan(&inserted).Error; err != nil {
		return 0, "", err
	}
	return inserted.ID, inserted.UUID, nil
}

func (s *Service) incrementCaseReportCountTx(ctx context.Context, tx *gorm.DB, caseID uint64) error {
	return tx.WithContext(ctx).Exec(`
		UPDATE moderation_cases
		SET report_count = report_count + 1,
		    updated_at = ?
		WHERE id = ?
	`, time.Now().UTC(), caseID).Error
}

func (s *Service) blockUserTx(ctx context.Context, tx *gorm.DB, blockerID uint64, blockedID uint64, reasonCode string, note *string, source string, reportID *uint64) (bool, error) {
	if blockerID == blockedID {
		return false, validationError("Cannot block yourself", map[string]any{"field": "targetUuid"})
	}
	var insertedID uint64
	err := tx.WithContext(ctx).Raw(`
		INSERT INTO user_blocks (
		  blocker_user_id,
		  blocked_user_id,
		  reason_code,
		  note,
		  source,
		  report_id
		)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT (blocker_user_id, blocked_user_id)
		DO UPDATE SET
		  reason_code = EXCLUDED.reason_code,
		  note = EXCLUDED.note,
		  source = EXCLUDED.source,
		  report_id = COALESCE(EXCLUDED.report_id, user_blocks.report_id)
		RETURNING id
	`, blockerID, blockedID, optionalString(reasonCode), note, source, reportID).Scan(&insertedID).Error
	return insertedID != 0, err
}

func (s *Service) createModerationActionTx(ctx context.Context, tx *gorm.DB, caseID uint64, actorID *uint64, targetID *uint64, actionType string, reason string, metadata map[string]any) error {
	payload, err := json.Marshal(metadata)
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	return tx.WithContext(ctx).Create(&ModerationAction{
		UUID:             uuid.New(),
		ModerationCaseID: &caseID,
		ActorUserID:      actorID,
		TargetUserID:     targetID,
		ActionType:       actionType,
		Reason:           optionalString(reason),
		Metadata:         datatypes.JSON(payload),
		CreatedAt:        now,
	}).Error
}

func (s *Service) userIDByUUID(ctx context.Context, rawUUID string) (uint64, error) {
	parsed, err := uuid.Parse(strings.TrimSpace(rawUUID))
	if err != nil {
		return 0, validationError("User UUID is invalid", map[string]any{"field": "userUuid"})
	}
	var id uint64
	err = s.db.WithContext(ctx).Raw("SELECT id FROM users WHERE uuid = ? AND deleted_at IS NULL", parsed).Scan(&id).Error
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

func settingsResponse(settings SafetySettings) SafetySettingsResponse {
	return SafetySettingsResponse{
		UUID:                 settings.UUID.String(),
		AllowMessageRequests: settings.AllowMessageRequests,
		AutoHideBlockedUsers: settings.AutoHideBlockedUsers,
	}
}

type normalizedReportRequest struct {
	TargetType string
	TargetUUID uuid.UUID
	ReasonCode string
	Note       *string
	BlockUser  bool
}

func normalizeReportRequest(req CreateReportRequest) (normalizedReportRequest, error) {
	targetType := strings.ToUpper(strings.TrimSpace(req.TargetType))
	if !validTargetType(targetType) {
		return normalizedReportRequest{}, validationError("targetType is invalid", map[string]any{"field": "targetType"})
	}
	targetUUID, err := uuid.Parse(strings.TrimSpace(req.TargetUUID))
	if err != nil {
		return normalizedReportRequest{}, validationError("targetUuid is invalid", map[string]any{"field": "targetUuid"})
	}
	reasonCode := strings.ToUpper(strings.TrimSpace(req.ReasonCode))
	if reasonCode == "" {
		return normalizedReportRequest{}, validationError("reasonCode is required", map[string]any{"field": "reasonCode"})
	}
	note, err := normalizeNote(req.Note)
	if err != nil {
		return normalizedReportRequest{}, err
	}
	return normalizedReportRequest{
		TargetType: targetType,
		TargetUUID: targetUUID,
		ReasonCode: reasonCode,
		Note:       note,
		BlockUser:  req.BlockUser,
	}, nil
}

type normalizedBlockRequest struct {
	ReasonCode string
	Note       *string
	Source     string
}

func normalizeBlockRequest(req BlockUserRequest, source string) (normalizedBlockRequest, error) {
	reasonCode := strings.ToUpper(strings.TrimSpace(derefString(req.ReasonCode)))
	note, err := normalizeNote(req.Note)
	if err != nil {
		return normalizedBlockRequest{}, err
	}
	source = strings.ToUpper(strings.TrimSpace(source))
	if source == "" {
		source = BlockSourceManual
	}
	if source != BlockSourceManual && source != BlockSourceReportFlow && source != BlockSourceModeration {
		return normalizedBlockRequest{}, validationError("Block source is invalid", map[string]any{"field": "source"})
	}
	return normalizedBlockRequest{ReasonCode: reasonCode, Note: note, Source: source}, nil
}

func normalizeNote(note *string) (*string, error) {
	if note == nil {
		return nil, nil
	}
	trimmed := strings.TrimSpace(*note)
	if trimmed == "" {
		return nil, nil
	}
	if len(trimmed) > 1000 {
		return nil, validationError("note must be 1000 characters or less", map[string]any{"field": "note"})
	}
	return &trimmed, nil
}

func validTargetType(value string) bool {
	switch value {
	case TargetUser, TargetProfile, TargetMessage, TargetMedia, TargetMatch:
		return true
	default:
		return false
	}
}

func reasonApplies(appliesJSON string, targetType string) bool {
	for _, value := range decodeStringArray(appliesJSON) {
		if value == targetType {
			return true
		}
	}
	return false
}

func decodeStringArray(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return []string{}
	}
	var values []string
	if err := json.Unmarshal([]byte(raw), &values); err != nil {
		return []string{}
	}
	return values
}

func optionalString(value string) *string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return &value
}

func derefString(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func SeverityForReason(reasonCode string) string {
	switch strings.ToUpper(strings.TrimSpace(reasonCode)) {
	case ReasonUnderage, ReasonViolenceThreat:
		return SeverityCritical
	case ReasonHateSpeech, ReasonHarassment, ReasonSexualContent, ReasonImpersonation:
		return SeverityHigh
	case ReasonScamSpam, ReasonFakeProfile, ReasonOther:
		return SeverityMedium
	default:
		return SeverityLow
	}
}

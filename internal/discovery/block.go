package discovery

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type blockedUserRow struct {
	UserUUID    string
	DisplayName *string
	BlockedAt   time.Time
}

func (s *Service) BlockUser(ctx context.Context, blockerUserID uint64, targetUserUUID string, req BlockUserRequest) error {
	targetID, err := s.userIDByUUID(ctx, targetUserUUID)
	if err != nil {
		return err
	}
	if blockerUserID == targetID {
		return validationError("Cannot block yourself", map[string]any{"field": "userUuid"})
	}
	reason := normalizeReason(req.Reason)
	return s.db.WithContext(ctx).Exec(`
		INSERT INTO user_blocks (blocker_user_id, blocked_user_id, reason)
		VALUES (?, ?, ?)
		ON CONFLICT (blocker_user_id, blocked_user_id)
		DO UPDATE SET reason = EXCLUDED.reason
	`, blockerUserID, targetID, reason).Error
}

func (s *Service) UnblockUser(ctx context.Context, blockerUserID uint64, targetUserUUID string) error {
	targetID, err := s.userIDByUUID(ctx, targetUserUUID)
	if err != nil {
		return err
	}
	return s.db.WithContext(ctx).
		Exec("DELETE FROM user_blocks WHERE blocker_user_id = ? AND blocked_user_id = ?", blockerUserID, targetID).
		Error
}

func (s *Service) ListBlocks(ctx context.Context, blockerUserID uint64) (*BlockListResponse, error) {
	var rows []blockedUserRow
	err := s.db.WithContext(ctx).Raw(`
		SELECT
		  u.uuid::text AS user_uuid,
		  p.display_name,
		  ub.created_at AS blocked_at
		FROM user_blocks ub
		JOIN users u ON u.id = ub.blocked_user_id
		LEFT JOIN profiles p ON p.user_id = u.id
		WHERE ub.blocker_user_id = ?
		ORDER BY ub.created_at DESC
	`, blockerUserID).Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	response := &BlockListResponse{Items: make([]BlockedUserResponse, 0, len(rows))}
	for _, row := range rows {
		response.Items = append(response.Items, BlockedUserResponse{
			UserUUID:    row.UserUUID,
			DisplayName: row.DisplayName,
			BlockedAt:   row.BlockedAt,
		})
	}
	return response, nil
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

func normalizeReason(reason *string) *string {
	if reason == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*reason)
	if trimmed == "" {
		return nil
	}
	if len(trimmed) > 1000 {
		trimmed = trimmed[:1000]
	}
	return &trimmed
}

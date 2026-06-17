package restriction

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"
)

var ErrActionRestricted = errors.New("user action restricted")

type ActionRestrictedError struct {
	Action          string
	RestrictionType string
}

func (e *ActionRestrictedError) Error() string {
	return fmt.Sprintf("%s: %s blocked by %s", ErrActionRestricted.Error(), e.Action, e.RestrictionType)
}

func (e *ActionRestrictedError) Unwrap() error {
	return ErrActionRestricted
}

type Service struct {
	db *gorm.DB
}

func NewService(db *gorm.DB) *Service {
	return &Service{db: db}
}

func (s *Service) CanPerform(ctx context.Context, userID uint64, action string) error {
	rows, err := s.activeRestrictions(ctx, userID, time.Now().UTC())
	if err != nil {
		return err
	}
	for _, row := range rows {
		if RestrictionBlocksAction(row.RestrictionType, action) {
			return &ActionRestrictedError{Action: action, RestrictionType: row.RestrictionType}
		}
	}
	return nil
}

func RestrictionBlocksAction(restrictionType string, action string) bool {
	if restrictionType == TypeFullPlatformBan {
		return true
	}
	switch action {
	case ActionCreateLive:
		return restrictionType == TypeLiveCreateBan
	case ActionSendComment:
		return restrictionType == TypeCommentBan
	case ActionSendGift:
		return restrictionType == TypeGiftSendBan
	case ActionResellerTopup:
		return restrictionType == TypeResellerOperationBan
	case ActionAgencyManage:
		return restrictionType == TypeAgencyOperationBan
	default:
		return false
	}
}

func IsValidRestrictionType(restrictionType string) bool {
	switch restrictionType {
	case TypeFullPlatformBan,
		TypeLiveCreateBan,
		TypeCommentBan,
		TypeGiftSendBan,
		TypeResellerOperationBan,
		TypeAgencyOperationBan:
		return true
	default:
		return false
	}
}

func IsValidStatus(status string) bool {
	return status == StatusActive || status == StatusRevoked || status == StatusExpired
}

func (s *Service) activeRestrictions(ctx context.Context, userID uint64, now time.Time) ([]UserRestriction, error) {
	var rows []UserRestriction
	err := s.db.WithContext(ctx).
		Where("user_id = ? AND status = ? AND (expires_at IS NULL OR expires_at > ?)", userID, StatusActive, now).
		Find(&rows).Error
	return rows, err
}

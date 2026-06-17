package restriction

import (
	"time"

	"github.com/google/uuid"
)

const (
	StatusActive  = "ACTIVE"
	StatusRevoked = "REVOKED"
	StatusExpired = "EXPIRED"

	TypeFullPlatformBan      = "FULL_PLATFORM_BAN"
	TypeLiveCreateBan        = "LIVE_CREATE_BAN"
	TypeCommentBan           = "COMMENT_BAN"
	TypeGiftSendBan          = "GIFT_SEND_BAN"
	TypeResellerOperationBan = "RESELLER_OPERATION_BAN"
	TypeAgencyOperationBan   = "AGENCY_OPERATION_BAN"

	ActionLogin         = "LOGIN"
	ActionRefreshToken  = "REFRESH_TOKEN"
	ActionSocketConnect = "SOCKET_CONNECT"
	ActionCreateLive    = "CREATE_LIVE"
	ActionSendComment   = "SEND_COMMENT"
	ActionSendGift      = "SEND_GIFT"
	ActionResellerTopup = "RESELLER_TOPUP"
	ActionAgencyManage  = "AGENCY_MANAGE"
	ActionAuthenticated = "AUTHENTICATED_REQUEST"
)

type UserRestriction struct {
	ID                   uint64    `gorm:"primaryKey"`
	UUID                 uuid.UUID `gorm:"type:uuid;uniqueIndex"`
	UserID               uint64
	RestrictionType      string
	Status               string
	Reason               string
	CreatedByAdminUserID uint64
	RevokedByAdminUserID *uint64
	RevokedAt            *time.Time
	RevocationReason     *string
	ExpiresAt            *time.Time
	CreatedAt            time.Time
	UpdatedAt            time.Time
}

func (UserRestriction) TableName() string {
	return "user_restrictions"
}

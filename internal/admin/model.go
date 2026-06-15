package admin

import (
	"time"

	"github.com/google/uuid"
)

const (
	StatusActive    = "ACTIVE"
	StatusSuspended = "SUSPENDED"

	PermissionPaymentRequestsRead    = "subscriptions.payment_requests.read"
	PermissionPaymentRequestsApprove = "subscriptions.payment_requests.approve"
	PermissionPaymentRequestsReject  = "subscriptions.payment_requests.reject"
	PermissionAuditRead              = "audit.read"

	RevokedReasonPasswordChanged = "PASSWORD_CHANGED"
)

type AdminUser struct {
	ID                uint64    `gorm:"primaryKey"`
	UUID              uuid.UUID `gorm:"type:uuid;uniqueIndex"`
	Email             string
	Name              *string
	PasswordHash      string
	Status            string
	LastLoginAt       *time.Time
	PasswordChangedAt *time.Time
	TokenVersion      int
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

func (AdminUser) TableName() string {
	return "admin_users"
}

type AdminSession struct {
	ID               uint64    `gorm:"primaryKey"`
	UUID             uuid.UUID `gorm:"type:uuid;uniqueIndex"`
	AdminUserID      uint64
	AdminUser        AdminUser
	RefreshTokenHash string
	TokenFamilyID    uuid.UUID `gorm:"type:uuid"`
	IPAddress        string
	UserAgent        string
	ExpiresAt        time.Time
	LastUsedAt       *time.Time
	RevokedAt        *time.Time
	RevokedReason    *string
	ReplacedAt       *time.Time
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

func (AdminSession) TableName() string {
	return "admin_sessions"
}

type AdminUserPermission struct {
	ID             uint64    `gorm:"primaryKey"`
	UUID           uuid.UUID `gorm:"type:uuid;uniqueIndex"`
	AdminUserID    uint64
	PermissionCode string
	CreatedAt      time.Time
}

func (AdminUserPermission) TableName() string {
	return "admin_user_permissions"
}

type AdminAuditLog struct {
	ID             uint64    `gorm:"primaryKey"`
	UUID           uuid.UUID `gorm:"type:uuid;uniqueIndex"`
	AdminUserID    *uint64
	Action         string
	ResourceType   string
	ResourceUUID   *uuid.UUID `gorm:"type:uuid"`
	BeforeSnapshot []byte
	AfterSnapshot  []byte
	IPAddress      *string
	UserAgent      *string
	CreatedAt      time.Time
}

func (AdminAuditLog) TableName() string {
	return "admin_audit_logs"
}

package auth

import (
	"time"

	"github.com/google/uuid"
)

const (
	UserStatusActive    = "ACTIVE"
	UserStatusSuspended = "SUSPENDED"
	UserStatusBanned    = "BANNED"
	UserStatusDeleted   = "DELETED"

	OnboardingPending         = "PENDING"
	OnboardingProfileRequired = "PROFILE_REQUIRED"
	OnboardingCompleted       = "COMPLETED"
)

type User struct {
	ID               uint64    `gorm:"primaryKey"`
	UUID             uuid.UUID `gorm:"type:uuid;uniqueIndex"`
	Phone            *string
	Email            *string `gorm:"type:citext"`
	PhoneVerifiedAt  *time.Time
	EmailVerifiedAt  *time.Time
	Status           string
	OnboardingStatus string
	LastLoginAt      *time.Time
	DeletedAt        *time.Time
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

func (User) TableName() string {
	return "users"
}

type UserSession struct {
	ID               uint64    `gorm:"primaryKey"`
	UUID             uuid.UUID `gorm:"type:uuid;uniqueIndex"`
	UserID           uint64
	User             User
	RefreshTokenHash string
	TokenFamilyID    uuid.UUID `gorm:"type:uuid"`
	DeviceID         *string
	DeviceName       *string
	Platform         *string
	IPAddress        string
	UserAgent        string
	ExpiresAt        time.Time
	LastUsedAt       *time.Time
	RevokedAt        *time.Time
	ReplacedAt       *time.Time
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

func (UserSession) TableName() string {
	return "user_sessions"
}

type Device struct {
	ID                uint64    `gorm:"primaryKey"`
	UUID              uuid.UUID `gorm:"type:uuid;uniqueIndex"`
	UserID            uint64
	DeviceID          string
	DeviceName        *string
	Platform          *string
	FCMToken          *string
	PushEnabled       bool
	FCMTokenUpdatedAt *time.Time
	LastPushSuccessAt *time.Time
	LastPushFailureAt *time.Time
	PushFailureCount  int
	AppVersion        *string
	OSVersion         *string
	LastActiveAt      *time.Time
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

func (Device) TableName() string {
	return "devices"
}

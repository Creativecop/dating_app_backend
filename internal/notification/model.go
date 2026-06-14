package notification

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

const (
	TypeNewMatch    = "NEW_MATCH"
	TypeChatMessage = "CHAT_MESSAGE"
	TypeSuperLike   = "SUPER_LIKE"
	TypeSystem      = "SYSTEM"

	DeliveryStatusSent    = "SENT"
	DeliveryStatusFailed  = "FAILED"
	DeliveryStatusSkipped = "SKIPPED"

	ProviderFCM  = "FCM"
	ProviderNoop = "NOOP"
)

type NotificationSettings struct {
	ID                 uint64    `gorm:"primaryKey"`
	UUID               uuid.UUID `gorm:"type:uuid;uniqueIndex"`
	UserID             uint64
	PushEnabled        bool
	NewMatchEnabled    bool
	ChatMessageEnabled bool
	SuperLikeEnabled   bool
	QuietHoursEnabled  bool
	QuietHoursStart    *string
	QuietHoursEnd      *string
	Timezone           string
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

func (NotificationSettings) TableName() string {
	return "notification_settings"
}

type Notification struct {
	ID               uint64    `gorm:"primaryKey"`
	UUID             uuid.UUID `gorm:"type:uuid;uniqueIndex"`
	UserID           uint64
	NotificationType string
	Title            string
	Body             string
	Data             datatypes.JSON
	ReadAt           *time.Time
	ClickedAt        *time.Time
	DedupeKey        *string
	CreatedAt        time.Time
}

func (Notification) TableName() string {
	return "notifications"
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

type PushDeliveryLog struct {
	ID                uint64    `gorm:"primaryKey"`
	UUID              uuid.UUID `gorm:"type:uuid;uniqueIndex"`
	NotificationID    *uint64
	UserID            uint64
	DeviceID          *uint64
	Provider          string
	Status            string
	ProviderMessageID *string
	ErrorCode         *string
	ErrorMessage      *string
	AttemptedAt       time.Time
	CreatedAt         time.Time
}

func (PushDeliveryLog) TableName() string {
	return "push_delivery_logs"
}

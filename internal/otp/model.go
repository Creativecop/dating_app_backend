package otp

import (
	"time"

	"github.com/google/uuid"
)

const (
	ChannelWhatsApp = "WHATSAPP"
	ChannelEmail    = "EMAIL"

	PurposeLogin       = "LOGIN"
	PurposeRegister    = "REGISTER"
	PurposeVerifyPhone = "VERIFY_PHONE"
	PurposeVerifyEmail = "VERIFY_EMAIL"
	PurposeReset       = "RESET"

	DeliveryPending = "PENDING"
	DeliveryQueued  = "QUEUED"
	DeliverySending = "SENDING"
	DeliverySent    = "SENT"
	DeliveryFailed  = "FAILED"
	DeliveryExpired = "EXPIRED"
)

type OTPCode struct {
	ID      uint64    `gorm:"primaryKey"`
	UUID    uuid.UUID `gorm:"type:uuid;uniqueIndex"`
	UserID  *uint64
	Channel string
	Purpose string

	IdentifierHash string
	Phone          *string
	Email          *string `gorm:"type:citext"`
	OTPHash        string

	ExpiresAt    time.Time
	ConsumedAt   *time.Time
	AttemptCount int
	MaxAttempts  int
	ResendCount  int

	IPAddress string
	UserAgent string
	DeviceID  *string

	DeliveryStatus        string
	DeliveryProvider      *string
	ProviderMessageID     *string
	DeliveryAttempts      int
	LastDeliveryAttemptAt *time.Time
	SentAt                *time.Time
	FailedAt              *time.Time
	DeliveryError         *string

	CreatedAt time.Time
	UpdatedAt time.Time
}

func (OTPCode) TableName() string {
	return "otp_codes"
}

type OTPDeliveryLog struct {
	ID                uint64    `gorm:"primaryKey"`
	UUID              uuid.UUID `gorm:"type:uuid;uniqueIndex"`
	OTPCodeID         uint64
	Channel           string
	Provider          *string
	Identifier        string
	Status            string
	ProviderMessageID *string
	ErrorMessage      *string
	AttemptedAt       time.Time
	CreatedAt         time.Time
}

func (OTPDeliveryLog) TableName() string {
	return "otp_delivery_logs"
}

package subscription

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

const (
	PaymentStatusPending     = "PENDING"
	PaymentStatusUnderReview = "UNDER_REVIEW"
	PaymentStatusApproved    = "APPROVED"
	PaymentStatusRejected    = "REJECTED"
	PaymentStatusCanceled    = "CANCELED"

	SubscriptionStatusActive   = "ACTIVE"
	SubscriptionStatusExpired  = "EXPIRED"
	SubscriptionStatusCanceled = "CANCELED"

	ReviewActionApproved = "APPROVED"
	ReviewActionRejected = "REJECTED"

	SubscriptionSourceManualPayment = "MANUAL_PAYMENT"
)

type SubscriptionPlan struct {
	ID           uint64    `gorm:"primaryKey"`
	UUID         uuid.UUID `gorm:"type:uuid;uniqueIndex"`
	PlanCode     string
	Name         string
	Description  *string
	PriceAmount  int
	Currency     string
	DurationDays int
	Entitlements datatypes.JSON
	SortOrder    int
	IsActive     bool
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func (SubscriptionPlan) TableName() string {
	return "subscription_plans"
}

type ManualPaymentRequest struct {
	ID                   uint64    `gorm:"primaryKey"`
	UUID                 uuid.UUID `gorm:"type:uuid;uniqueIndex"`
	UserID               uint64
	PlanID               *uint64
	PlanCodeSnapshot     string
	PlanNameSnapshot     string
	PriceAmountSnapshot  int
	CurrencySnapshot     string
	DurationDaysSnapshot int
	EntitlementsSnapshot datatypes.JSON
	PaymentProvider      string
	PaymentReference     *string
	PayerPhone           *string
	Note                 *string
	Status               string
	SubmittedAt          time.Time
	ReviewedAt           *time.Time
	ReviewedByAdminID    *uint64
	RejectionReason      *string
	SubscriptionID       *uint64
	CreatedAt            time.Time
	UpdatedAt            time.Time
}

func (ManualPaymentRequest) TableName() string {
	return "manual_payment_requests"
}

type UserSubscription struct {
	ID                   uint64    `gorm:"primaryKey"`
	UUID                 uuid.UUID `gorm:"type:uuid;uniqueIndex"`
	UserID               uint64
	PlanID               *uint64
	PaymentRequestID     *uint64
	PlanCode             string
	PlanName             string
	Source               string
	Status               string
	StartsAt             time.Time
	ExpiresAt            time.Time
	CanceledAt           *time.Time
	EntitlementsSnapshot datatypes.JSON
	CreatedAt            time.Time
	UpdatedAt            time.Time
}

func (UserSubscription) TableName() string {
	return "user_subscriptions"
}

type UserFeatureUsage struct {
	ID          uint64    `gorm:"primaryKey"`
	UUID        uuid.UUID `gorm:"type:uuid;uniqueIndex"`
	UserID      uint64
	FeatureKey  string
	UsageDate   time.Time
	UsedCount   int
	UsedSeconds int
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func (UserFeatureUsage) TableName() string {
	return "user_feature_usage"
}

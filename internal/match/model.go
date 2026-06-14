package match

import (
	"time"

	"github.com/google/uuid"
)

const (
	StatusActive    = "ACTIVE"
	StatusUnmatched = "UNMATCHED"

	UserStatusActive = "ACTIVE"

	VariantDisplay   = "DISPLAY"
	VariantThumbnail = "THUMBNAIL"
)

type Match struct {
	ID                uint64    `gorm:"primaryKey"`
	UUID              uuid.UUID `gorm:"type:uuid;uniqueIndex"`
	UserLowID         uint64
	UserHighID        uint64
	InitiatedByUserID *uint64
	Status            string
	MatchedAt         time.Time
	UnmatchedAt       *time.Time
	UnmatchedByUserID *uint64
	UnmatchReasonCode *string
	UnmatchReasonNote *string
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

func (Match) TableName() string {
	return "matches"
}

type MatchParticipant struct {
	ID           uint64    `gorm:"primaryKey"`
	UUID         uuid.UUID `gorm:"type:uuid;uniqueIndex"`
	MatchID      uint64
	UserID       uint64
	SeenAt       *time.Time
	LastOpenedAt *time.Time
	HiddenAt     *time.Time
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func (MatchParticipant) TableName() string {
	return "match_participants"
}

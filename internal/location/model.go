package location

import (
	"time"

	"github.com/google/uuid"
)

const (
	SourceGPS    = "GPS"
	SourceManual = "MANUAL"
	SourceIP     = "IP"
)

type UserLocation struct {
	ID                uint64    `gorm:"primaryKey"`
	UUID              uuid.UUID `gorm:"type:uuid;uniqueIndex"`
	UserID            uint64    `gorm:"uniqueIndex"`
	Latitude          float64
	Longitude         float64
	AccuracyMeters    *float64
	IsPrecise         bool
	LocationConsentAt *time.Time
	City              *string
	Country           *string
	Source            string
	LastUpdatedAt     time.Time
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

func (UserLocation) TableName() string {
	return "user_locations"
}

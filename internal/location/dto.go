package location

import "time"

type UpdateLocationRequest struct {
	Latitude       *float64 `json:"latitude"`
	Longitude      *float64 `json:"longitude"`
	AccuracyMeters *float64 `json:"accuracyMeters"`
	City           *string  `json:"city"`
	Country        *string  `json:"country"`
	Source         *string  `json:"source"`
}

type LocationResponse struct {
	UUID              string     `json:"uuid"`
	Latitude          float64    `json:"latitude"`
	Longitude         float64    `json:"longitude"`
	AccuracyMeters    *float64   `json:"accuracyMeters"`
	IsPrecise         bool       `json:"isPrecise"`
	LocationConsentAt *time.Time `json:"locationConsentAt"`
	City              *string    `json:"city"`
	Country           *string    `json:"country"`
	Source            string     `json:"source"`
	LastUpdatedAt     time.Time  `json:"lastUpdatedAt"`
}

package location

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type DiscoveryEligibilityRefresher interface {
	RefreshDiscoveryEligibility(ctx context.Context, userID uint64) error
}

type Service struct {
	db        *gorm.DB
	refresher DiscoveryEligibilityRefresher
}

type normalizedLocation struct {
	Latitude          float64
	Longitude         float64
	AccuracyMeters    *float64
	City              *string
	Country           *string
	Source            string
	IsPrecise         bool
	LocationConsentAt *time.Time
}

func NewService(db *gorm.DB) *Service {
	return &Service{db: db}
}

func (s *Service) SetDiscoveryEligibilityRefresher(refresher DiscoveryEligibilityRefresher) {
	s.refresher = refresher
}

func (s *Service) GetMine(ctx context.Context, userID uint64) (*LocationResponse, error) {
	var row UserLocation
	err := s.db.WithContext(ctx).Where("user_id = ?", userID).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, notFoundError("Location not found")
	}
	if err != nil {
		return nil, err
	}
	response := toResponse(row)
	return &response, nil
}

func (s *Service) Update(ctx context.Context, userID uint64, req UpdateLocationRequest) (*LocationResponse, error) {
	normalized, err := normalizeUpdateRequest(req)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	row := UserLocation{
		UUID:              uuid.New(),
		UserID:            userID,
		Latitude:          normalized.Latitude,
		Longitude:         normalized.Longitude,
		AccuracyMeters:    normalized.AccuracyMeters,
		IsPrecise:         normalized.IsPrecise,
		LocationConsentAt: normalized.LocationConsentAt,
		City:              normalized.City,
		Country:           normalized.Country,
		Source:            normalized.Source,
		LastUpdatedAt:     now,
		CreatedAt:         now,
		UpdatedAt:         now,
	}

	err = s.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "user_id"}},
		DoUpdates: clause.Assignments(map[string]any{
			"latitude":            row.Latitude,
			"longitude":           row.Longitude,
			"accuracy_meters":     row.AccuracyMeters,
			"is_precise":          row.IsPrecise,
			"location_consent_at": row.LocationConsentAt,
			"city":                row.City,
			"country":             row.Country,
			"source":              row.Source,
			"last_updated_at":     row.LastUpdatedAt,
			"updated_at":          row.UpdatedAt,
		}),
	}).Create(&row).Error
	if err != nil {
		return nil, err
	}

	if err := s.refreshEligibility(ctx, userID); err != nil {
		return nil, err
	}

	var fresh UserLocation
	if err := s.db.WithContext(ctx).Where("user_id = ?", userID).First(&fresh).Error; err != nil {
		return nil, err
	}
	response := toResponse(fresh)
	return &response, nil
}

func (s *Service) refreshEligibility(ctx context.Context, userID uint64) error {
	if s.refresher == nil {
		return nil
	}
	return s.refresher.RefreshDiscoveryEligibility(ctx, userID)
}

func normalizeUpdateRequest(req UpdateLocationRequest) (normalizedLocation, error) {
	if req.Latitude == nil {
		return normalizedLocation{}, validationError("Latitude is required", map[string]any{"field": "latitude"})
	}
	if req.Longitude == nil {
		return normalizedLocation{}, validationError("Longitude is required", map[string]any{"field": "longitude"})
	}
	if *req.Latitude < -90 || *req.Latitude > 90 {
		return normalizedLocation{}, validationError("Latitude must be between -90 and 90", map[string]any{"field": "latitude"})
	}
	if *req.Longitude < -180 || *req.Longitude > 180 {
		return normalizedLocation{}, validationError("Longitude must be between -180 and 180", map[string]any{"field": "longitude"})
	}
	if req.AccuracyMeters != nil && (*req.AccuracyMeters <= 0 || *req.AccuracyMeters > 10000) {
		return normalizedLocation{}, validationError("Accuracy must be greater than 0 and at most 10000 meters", map[string]any{"field": "accuracyMeters"})
	}
	if req.Source == nil || strings.TrimSpace(*req.Source) == "" {
		return normalizedLocation{}, validationError("Location source is required", map[string]any{"field": "source"})
	}

	source := strings.ToUpper(strings.TrimSpace(*req.Source))
	if !validSource(source) {
		return normalizedLocation{}, validationError("Location source is invalid", map[string]any{"field": "source"})
	}

	city, err := normalizeOptionalText(req.City, 120, "city")
	if err != nil {
		return normalizedLocation{}, err
	}
	country, err := normalizeOptionalText(req.Country, 120, "country")
	if err != nil {
		return normalizedLocation{}, err
	}

	var consentAt *time.Time
	isPrecise := source == SourceGPS || source == SourceManual
	if isPrecise {
		now := time.Now().UTC()
		consentAt = &now
	}

	return normalizedLocation{
		Latitude:          *req.Latitude,
		Longitude:         *req.Longitude,
		AccuracyMeters:    req.AccuracyMeters,
		City:              city,
		Country:           country,
		Source:            source,
		IsPrecise:         isPrecise,
		LocationConsentAt: consentAt,
	}, nil
}

func validSource(source string) bool {
	return source == SourceGPS || source == SourceManual || source == SourceIP
}

func normalizeOptionalText(value *string, maxLength int, field string) (*string, error) {
	if value == nil {
		return nil, nil
	}
	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil, nil
	}
	if len(trimmed) > maxLength {
		return nil, validationError(field+" must be 120 characters or less", map[string]any{"field": field})
	}
	return &trimmed, nil
}

func toResponse(row UserLocation) LocationResponse {
	return LocationResponse{
		UUID:              row.UUID.String(),
		Latitude:          row.Latitude,
		Longitude:         row.Longitude,
		AccuracyMeters:    row.AccuracyMeters,
		IsPrecise:         row.IsPrecise,
		LocationConsentAt: row.LocationConsentAt,
		City:              row.City,
		Country:           row.Country,
		Source:            row.Source,
		LastUpdatedAt:     row.LastUpdatedAt,
	}
}

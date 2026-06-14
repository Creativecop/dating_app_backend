package discovery

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/neoscoder/aura-backend/internal/config"
)

type DiscoveryEligibilityRefresher interface {
	RefreshDiscoveryEligibility(ctx context.Context, userID uint64) error
}

type MatchNotificationDispatcher interface {
	NotifyNewMatch(ctx context.Context, matchUUID string) error
}

type ActionUsageLimiter interface {
	ConsumeActionUsageTx(ctx context.Context, tx *gorm.DB, userID uint64, actionType string, usageDate time.Time) error
}

type BlockChecker interface {
	IsBlockedEitherDirection(ctx context.Context, userAID uint64, userBID uint64) (bool, error)
}

type Service struct {
	db            *gorm.DB
	cfg           config.DiscoveryConfig
	refresher     DiscoveryEligibilityRefresher
	notifications MatchNotificationDispatcher
	blockChecker  BlockChecker
	usageLimiter  ActionUsageLimiter
}

type normalizedPreferences struct {
	MinAge            int
	MaxAge            int
	PreferredGenders  []string
	MaxDistanceKM     int
	VerifiedOnly      bool
	ShowMeInDiscovery bool
	HideDistance      bool
}

type preferenceRow struct {
	UUID                 string
	MinAge               int
	MaxAge               int
	PreferredGendersJSON string
	MaxDistanceKM        int
	VerifiedOnly         bool
	ShowMeInDiscovery    bool
	HideDistance         bool
	IsDefault            bool
	CustomizedAt         *time.Time
	ActivatedAt          *time.Time
}

type readinessSnapshot struct {
	UserStatus                  string
	UserDeleted                 bool
	HasProfile                  bool
	ProfileStatus               string
	HasApprovedPrimaryPhoto     bool
	HasLocation                 bool
	HasPreciseConsentedLocation bool
	HasFreshLocation            bool
	HasPreferences              bool
	ShowMeInDiscovery           bool
}

func NewService(db *gorm.DB, cfg config.DiscoveryConfig) *Service {
	return &Service{db: db, cfg: cfg}
}

func (s *Service) SetDiscoveryEligibilityRefresher(refresher DiscoveryEligibilityRefresher) {
	s.refresher = refresher
}

func (s *Service) SetMatchNotificationDispatcher(dispatcher MatchNotificationDispatcher) {
	s.notifications = dispatcher
}

func (s *Service) SetBlockChecker(checker BlockChecker) {
	s.blockChecker = checker
}

func (s *Service) SetActionUsageLimiter(limiter ActionUsageLimiter) {
	s.usageLimiter = limiter
}

func (s *Service) EnsureDiscoveryPreferences(ctx context.Context, userID uint64) error {
	return s.db.WithContext(ctx).Exec(`
		INSERT INTO discovery_preferences (user_id)
		VALUES (?)
		ON CONFLICT (user_id) DO NOTHING
	`, userID).Error
}

func (s *Service) GetPreferences(ctx context.Context, userID uint64) (*PreferencesResponse, error) {
	if err := s.EnsureDiscoveryPreferences(ctx, userID); err != nil {
		return nil, err
	}
	row, err := s.preferenceByUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	response, err := row.toResponse()
	if err != nil {
		return nil, err
	}
	return &response, nil
}

func (s *Service) UpdatePreferences(ctx context.Context, userID uint64, req UpdatePreferencesRequest) (*PreferencesResponse, error) {
	normalized, err := normalizePreferencesRequest(req)
	if err != nil {
		return nil, err
	}
	if err := s.EnsureDiscoveryPreferences(ctx, userID); err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	arrayLiteral := textArrayLiteral(normalized.PreferredGenders)
	err = s.db.WithContext(ctx).Exec(`
		UPDATE discovery_preferences
		SET min_age = ?,
		    max_age = ?,
		    preferred_genders = ?::text[],
		    max_distance_km = ?,
		    verified_only = ?,
		    show_me_in_discovery = ?,
		    hide_distance = ?,
		    is_default = FALSE,
		    customized_at = ?,
		    activated_at = CASE
		      WHEN ? = TRUE AND activated_at IS NULL THEN ?
		      ELSE activated_at
		    END,
		    updated_at = ?
		WHERE user_id = ?
	`,
		normalized.MinAge,
		normalized.MaxAge,
		arrayLiteral,
		normalized.MaxDistanceKM,
		normalized.VerifiedOnly,
		normalized.ShowMeInDiscovery,
		normalized.HideDistance,
		now,
		normalized.ShowMeInDiscovery,
		now,
		now,
		userID,
	).Error
	if err != nil {
		return nil, err
	}

	if err := s.refreshEligibility(ctx, userID); err != nil {
		return nil, err
	}

	return s.GetPreferences(ctx, userID)
}

func (s *Service) Readiness(ctx context.Context, userID uint64) (*ReadinessResponse, error) {
	if err := s.EnsureDiscoveryPreferences(ctx, userID); err != nil {
		return nil, err
	}
	snapshot, err := s.readinessSnapshot(ctx, userID)
	if err != nil {
		return nil, err
	}
	response := buildReadiness(snapshot)
	return &response, nil
}

func (s *Service) preferenceByUser(ctx context.Context, userID uint64) (preferenceRow, error) {
	var row preferenceRow
	err := s.db.WithContext(ctx).Raw(`
		SELECT
		  uuid::text AS uuid,
		  min_age,
		  max_age,
		  COALESCE(array_to_json(preferred_genders)::text, '[]') AS preferred_genders_json,
		  max_distance_km,
		  verified_only,
		  show_me_in_discovery,
		  hide_distance,
		  is_default,
		  customized_at,
		  activated_at
		FROM discovery_preferences
		WHERE user_id = ?
	`, userID).Scan(&row).Error
	if err != nil {
		return preferenceRow{}, err
	}
	if row.UUID == "" {
		return preferenceRow{}, notFoundError("Discovery preferences not found")
	}
	return row, nil
}

func (s *Service) readinessSnapshot(ctx context.Context, userID uint64) (readinessSnapshot, error) {
	var snapshot readinessSnapshot
	freshSince := time.Now().UTC().AddDate(0, 0, -s.cfg.LocationMaxAgeDays)
	err := s.db.WithContext(ctx).Raw(`
		SELECT
		  COALESCE(u.status, '') AS user_status,
		  (u.deleted_at IS NOT NULL) AS user_deleted,
		  (p.id IS NOT NULL) AS has_profile,
		  COALESCE(p.profile_status, '') AS profile_status,
		  EXISTS (
		    SELECT 1
		    FROM user_media m
		    WHERE m.user_id = u.id
		      AND m.media_purpose = 'PROFILE_PHOTO'
		      AND m.processing_status = 'READY'
		      AND m.moderation_status = 'APPROVED'
		      AND m.is_primary = TRUE
		      AND m.deleted_at IS NULL
		  ) AS has_approved_primary_photo,
		  (ul.id IS NOT NULL) AS has_location,
		  COALESCE(
		    ul.source IN ('GPS', 'MANUAL')
		    AND ul.is_precise = TRUE
		    AND ul.location_consent_at IS NOT NULL,
		    FALSE
		  ) AS has_precise_consented_location,
		  COALESCE(ul.last_updated_at >= ?, FALSE) AS has_fresh_location,
		  (dp.id IS NOT NULL) AS has_preferences,
		  COALESCE(dp.show_me_in_discovery, FALSE) AS show_me_in_discovery
		FROM users u
		LEFT JOIN profiles p ON p.user_id = u.id
		LEFT JOIN user_locations ul ON ul.user_id = u.id
		LEFT JOIN discovery_preferences dp ON dp.user_id = u.id
		WHERE u.id = ?
	`, freshSince, userID).Scan(&snapshot).Error
	if err != nil {
		return readinessSnapshot{}, err
	}
	if snapshot.UserStatus == "" {
		return readinessSnapshot{}, notFoundError("User not found")
	}
	return snapshot, nil
}

func (s *Service) refreshEligibility(ctx context.Context, userID uint64) error {
	if s.refresher == nil {
		return nil
	}
	return s.refresher.RefreshDiscoveryEligibility(ctx, userID)
}

func (r preferenceRow) toResponse() (PreferencesResponse, error) {
	genders := make([]string, 0)
	if err := json.Unmarshal([]byte(r.PreferredGendersJSON), &genders); err != nil {
		return PreferencesResponse{}, err
	}
	return PreferencesResponse{
		UUID:              r.UUID,
		MinAge:            r.MinAge,
		MaxAge:            r.MaxAge,
		PreferredGenders:  genders,
		MaxDistanceKM:     r.MaxDistanceKM,
		VerifiedOnly:      r.VerifiedOnly,
		ShowMeInDiscovery: r.ShowMeInDiscovery,
		HideDistance:      r.HideDistance,
		IsDefault:         r.IsDefault,
		CustomizedAt:      r.CustomizedAt,
		ActivatedAt:       r.ActivatedAt,
	}, nil
}

func normalizePreferencesRequest(req UpdatePreferencesRequest) (normalizedPreferences, error) {
	required := []string{
		"minAge",
		"maxAge",
		"preferredGenders",
		"maxDistanceKm",
		"verifiedOnly",
		"showMeInDiscovery",
		"hideDistance",
	}
	for _, field := range required {
		if !req.Has(field) || req.IsNull(field) {
			return normalizedPreferences{}, validationError("All discovery preference fields are required", map[string]any{"field": field})
		}
	}
	if req.MinAge == nil || req.MaxAge == nil || req.MaxDistanceKM == nil || req.VerifiedOnly == nil || req.ShowMeInDiscovery == nil || req.HideDistance == nil {
		return normalizedPreferences{}, validationError("All discovery preference fields are required", nil)
	}
	if *req.MinAge < 18 {
		return normalizedPreferences{}, validationError("Minimum age must be at least 18", map[string]any{"field": "minAge"})
	}
	if *req.MaxAge > 80 {
		return normalizedPreferences{}, validationError("Maximum age must be at most 80", map[string]any{"field": "maxAge"})
	}
	if *req.MinAge > *req.MaxAge {
		return normalizedPreferences{}, validationError("Minimum age cannot be greater than maximum age", map[string]any{"field": "minAge"})
	}
	if *req.MaxDistanceKM < 1 || *req.MaxDistanceKM > 200 {
		return normalizedPreferences{}, validationError("Maximum distance must be between 1 and 200 km", map[string]any{"field": "maxDistanceKm"})
	}

	genders, err := normalizeGenders(req.PreferredGenders)
	if err != nil {
		return normalizedPreferences{}, err
	}

	return normalizedPreferences{
		MinAge:            *req.MinAge,
		MaxAge:            *req.MaxAge,
		PreferredGenders:  genders,
		MaxDistanceKM:     *req.MaxDistanceKM,
		VerifiedOnly:      *req.VerifiedOnly,
		ShowMeInDiscovery: *req.ShowMeInDiscovery,
		HideDistance:      *req.HideDistance,
	}, nil
}

func normalizeGenders(values []string) ([]string, error) {
	if len(values) == 0 {
		return nil, validationError("Preferred genders cannot be empty", map[string]any{"field": "preferredGenders"})
	}
	seen := make(map[string]bool, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		normalized := strings.ToUpper(strings.TrimSpace(value))
		if !validGender(normalized) {
			return nil, validationError("Preferred gender is invalid", map[string]any{"field": "preferredGenders"})
		}
		if seen[normalized] {
			return nil, validationError("Preferred genders must be unique", map[string]any{"field": "preferredGenders"})
		}
		seen[normalized] = true
		if normalized == GenderEveryone {
			return []string{GenderEveryone}, nil
		}
		result = append(result, normalized)
	}
	return result, nil
}

func validGender(value string) bool {
	return value == GenderMale ||
		value == GenderFemale ||
		value == GenderNonBinary ||
		value == GenderOther ||
		value == GenderEveryone
}

func buildReadiness(snapshot readinessSnapshot) ReadinessResponse {
	missing := make([]string, 0)
	blocked := make([]string, 0)

	if snapshot.UserDeleted || snapshot.UserStatus == UserStatusDeleted ||
		snapshot.UserStatus == UserStatusBanned || snapshot.UserStatus == UserStatusSuspended ||
		snapshot.UserStatus != UserStatusActive {
		blocked = append(blocked, "userBanned")
	}

	if !snapshot.HasProfile || snapshot.ProfileStatus == "" || snapshot.ProfileStatus == ProfileStatusDraft {
		missing = append(missing, "profile")
	} else if snapshot.ProfileStatus == ProfileStatusPaused {
		blocked = append(blocked, "profilePaused")
	} else if snapshot.ProfileStatus == ProfileStatusUnderReview || snapshot.ProfileStatus == ProfileStatusRejected {
		blocked = append(blocked, "profileUnderReview")
	} else if snapshot.ProfileStatus != ProfileStatusActive {
		missing = append(missing, "profile")
	}

	if !snapshot.HasApprovedPrimaryPhoto {
		missing = append(missing, "approvedPrimaryPhoto")
	}
	if !snapshot.HasLocation || !snapshot.HasPreciseConsentedLocation {
		missing = append(missing, "location")
	} else if !snapshot.HasFreshLocation {
		missing = append(missing, "freshLocation")
	}
	if !snapshot.HasPreferences {
		missing = append(missing, "preferences")
	} else if !snapshot.ShowMeInDiscovery {
		blocked = append(blocked, "showMeInDiscoveryDisabled")
	}

	return ReadinessResponse{
		DiscoveryEligible: len(missing) == 0 && len(blocked) == 0,
		Missing:           missing,
		Blocked:           blocked,
	}
}

func textArrayLiteral(values []string) string {
	escaped := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.ReplaceAll(value, `\`, `\\`)
		value = strings.ReplaceAll(value, `"`, `\"`)
		escaped = append(escaped, `"`+value+`"`)
	}
	return "{" + strings.Join(escaped, ",") + "}"
}

func IsServiceError(err error) bool {
	var serviceErr *ServiceError
	return errors.As(err, &serviceErr)
}

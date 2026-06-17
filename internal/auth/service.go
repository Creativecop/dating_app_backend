package auth

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/neoscoder/aura-backend/internal/config"
	"github.com/neoscoder/aura-backend/internal/otp"
	"github.com/neoscoder/aura-backend/internal/restriction"
)

var (
	ErrUnauthorized   = errors.New("unauthorized")
	ErrInactiveUser   = errors.New("inactive user")
	ErrInvalidRefresh = errors.New("invalid refresh token")
	ErrDeviceMismatch = errors.New("device mismatch")
	ErrUserNotFound   = errors.New("user not found")
	ErrAccountDeleted = errors.New("account deleted")
)

type Service struct {
	db                          *gorm.DB
	cfg                         *config.Config
	otp                         *otp.Service
	tokens                      *TokenService
	profileEnsurer              ProfileEnsurer
	discoveryPreferencesEnsurer DiscoveryPreferencesEnsurer
	notificationSettingsEnsurer NotificationSettingsEnsurer
	safetySettingsEnsurer       SafetySettingsEnsurer
	restrictionChecker          RestrictionChecker
}

type RequestMeta struct {
	IPAddress string
	UserAgent string
}

type ProfileEnsurer interface {
	EnsureProfile(ctx context.Context, userID uint64) error
}

type DiscoveryPreferencesEnsurer interface {
	EnsureDiscoveryPreferences(ctx context.Context, userID uint64) error
}

type NotificationSettingsEnsurer interface {
	EnsureNotificationSettings(ctx context.Context, userID uint64) error
}

type SafetySettingsEnsurer interface {
	EnsureSafetySettings(ctx context.Context, userID uint64) error
}

type RestrictionChecker interface {
	CanPerform(ctx context.Context, userID uint64, action string) error
}

func NewService(db *gorm.DB, cfg *config.Config, otpService *otp.Service) *Service {
	return &Service{
		db:     db,
		cfg:    cfg,
		otp:    otpService,
		tokens: NewTokenService(cfg.JWT),
	}
}

func (s *Service) SetProfileEnsurer(ensurer ProfileEnsurer) {
	s.profileEnsurer = ensurer
}

func (s *Service) SetDiscoveryPreferencesEnsurer(ensurer DiscoveryPreferencesEnsurer) {
	s.discoveryPreferencesEnsurer = ensurer
}

func (s *Service) SetNotificationSettingsEnsurer(ensurer NotificationSettingsEnsurer) {
	s.notificationSettingsEnsurer = ensurer
}

func (s *Service) SetSafetySettingsEnsurer(ensurer SafetySettingsEnsurer) {
	s.safetySettingsEnsurer = ensurer
}

func (s *Service) SetRestrictionChecker(checker RestrictionChecker) {
	s.restrictionChecker = checker
}

func (s *Service) RequestOTP(ctx context.Context, req RequestOTPRequest, meta RequestMeta) error {
	return s.requestOTP(ctx, req, meta)
}

func (s *Service) requestOTP(ctx context.Context, req RequestOTPRequest, meta RequestMeta) error {
	phone, email := deref(req.Phone), deref(req.Email)
	_, err := s.otp.Request(ctx, otp.RequestInput{
		Channel:   req.Channel,
		Phone:     phone,
		Email:     email,
		Purpose:   req.Purpose,
		IPAddress: meta.IPAddress,
		UserAgent: meta.UserAgent,
		DeviceID:  req.DeviceID,
	})
	return err
}

func (s *Service) VerifyOTP(ctx context.Context, req VerifyOTPRequest, meta RequestMeta) (*AuthResponse, error) {
	code, err := s.otp.Verify(ctx, otp.VerifyInput{
		Channel: req.Channel,
		Phone:   deref(req.Phone),
		Email:   deref(req.Email),
		Purpose: req.Purpose,
		Code:    req.Code,
	})
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	var user User
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var findErr error
		if code.Channel == otp.ChannelWhatsApp && code.Phone != nil {
			findErr = tx.Where("phone = ? AND deleted_at IS NULL", *code.Phone).First(&user).Error
		} else if code.Channel == otp.ChannelEmail && code.Email != nil {
			findErr = tx.Where("email = ? AND deleted_at IS NULL", *code.Email).First(&user).Error
		} else {
			return otp.ErrInvalidIdentifier
		}

		if errors.Is(findErr, gorm.ErrRecordNotFound) {
			user = User{
				UUID:             uuid.New(),
				Status:           UserStatusActive,
				OnboardingStatus: OnboardingProfileRequired,
				CreatedAt:        now,
				UpdatedAt:        now,
			}
			if code.Phone != nil {
				user.Phone = code.Phone
			}
			if code.Email != nil {
				user.Email = code.Email
			}
			if err := tx.Create(&user).Error; err != nil {
				return err
			}
		} else if findErr != nil {
			return findErr
		}

		if user.Status != UserStatusActive || user.DeletedAt != nil {
			return ErrInactiveUser
		}
		if err := s.canPerform(ctx, user.ID, restriction.ActionLogin); err != nil {
			return err
		}

		updates := map[string]any{"last_login_at": now, "updated_at": now}
		if code.Channel == otp.ChannelWhatsApp {
			updates["phone_verified_at"] = now
		}
		if code.Channel == otp.ChannelEmail {
			updates["email_verified_at"] = now
		}
		if err := tx.Model(&User{}).Where("id = ?", user.ID).Updates(updates).Error; err != nil {
			return err
		}
		if err := tx.Where("id = ?", user.ID).First(&user).Error; err != nil {
			return err
		}

		return s.upsertDevice(ctx, tx, user.ID, req, now)
	})
	if err != nil {
		return nil, err
	}

	if s.profileEnsurer != nil {
		if err := s.profileEnsurer.EnsureProfile(ctx, user.ID); err != nil {
			return nil, err
		}
	}
	if s.discoveryPreferencesEnsurer != nil {
		if err := s.discoveryPreferencesEnsurer.EnsureDiscoveryPreferences(ctx, user.ID); err != nil {
			return nil, err
		}
	}
	if s.notificationSettingsEnsurer != nil {
		if err := s.notificationSettingsEnsurer.EnsureNotificationSettings(ctx, user.ID); err != nil {
			return nil, err
		}
	}
	if s.safetySettingsEnsurer != nil {
		if err := s.safetySettingsEnsurer.EnsureSafetySettings(ctx, user.ID); err != nil {
			return nil, err
		}
	}

	session, refreshToken, err := s.createSession(ctx, user, req, meta, now)
	if err != nil {
		return nil, err
	}

	accessToken, err := s.tokens.GenerateAccessToken(user, *session)
	if err != nil {
		return nil, err
	}

	return &AuthResponse{
		AccessToken:      accessToken,
		RefreshToken:     refreshToken,
		ExpiresIn:        int64(s.cfg.JWT.AccessTTL().Seconds()),
		RefreshExpiresIn: int64(s.cfg.JWT.RefreshTTL().Seconds()),
		User:             ToUserResponse(user),
	}, nil
}

func (s *Service) RefreshToken(ctx context.Context, req RefreshTokenRequest, meta RequestMeta) (*TokenResponse, error) {
	incomingHash := HashRefreshToken(req.RefreshToken)
	now := time.Now().UTC()

	var session UserSession
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Preload("User").
			Where("refresh_token_hash = ?", incomingHash).
			First(&session).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrInvalidRefresh
			}
			return err
		}

		if err := validateSession(session, now); err != nil {
			return err
		}
		if err := validateUser(session.User); err != nil {
			return err
		}
		if err := s.canPerform(ctx, session.UserID, restriction.ActionRefreshToken); err != nil {
			return err
		}
		if req.DeviceID != "" && session.DeviceID != nil && *session.DeviceID != req.DeviceID {
			return ErrDeviceMismatch
		}

		newRefreshToken, err := GenerateRefreshToken()
		if err != nil {
			return err
		}
		session.RefreshTokenHash = HashRefreshToken(newRefreshToken)
		session.LastUsedAt = &now
		session.ReplacedAt = &now
		session.IPAddress = meta.IPAddress
		session.UserAgent = meta.UserAgent
		session.UpdatedAt = now

		if err := tx.Save(&session).Error; err != nil {
			return err
		}

		req.RefreshToken = newRefreshToken
		return nil
	})
	if err != nil {
		return nil, err
	}

	accessToken, err := s.tokens.GenerateAccessToken(session.User, session)
	if err != nil {
		return nil, err
	}

	return &TokenResponse{
		AccessToken:      accessToken,
		RefreshToken:     req.RefreshToken,
		ExpiresIn:        int64(s.cfg.JWT.AccessTTL().Seconds()),
		RefreshExpiresIn: int64(time.Until(session.ExpiresAt).Seconds()),
	}, nil
}

func (s *Service) Logout(ctx context.Context, user AuthenticatedUser, req LogoutRequest) error {
	now := time.Now().UTC()
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		deviceID := strings.TrimSpace(req.DeviceID)
		if deviceID == "" {
			var session UserSession
			query := tx.WithContext(ctx).Where("user_id = ?", user.UserID)
			if req.RefreshToken != "" {
				query = query.Where("refresh_token_hash = ?", HashRefreshToken(req.RefreshToken))
			} else {
				query = query.Where("id = ?", user.SessionID)
			}
			if err := query.First(&session).Error; err == nil && session.DeviceID != nil {
				deviceID = *session.DeviceID
			}
		}

		query := tx.Model(&UserSession{}).
			Where("user_id = ? AND revoked_at IS NULL", user.UserID)

		if req.RefreshToken != "" {
			query = query.Where("refresh_token_hash = ?", HashRefreshToken(req.RefreshToken))
		} else {
			query = query.Where("id = ?", user.SessionID)
		}
		if err := query.Updates(map[string]any{"revoked_at": now, "updated_at": now}).Error; err != nil {
			return err
		}
		if deviceID == "" {
			return nil
		}
		return tx.Model(&Device{}).
			Where("user_id = ? AND device_id = ?", user.UserID, deviceID).
			Updates(map[string]any{
				"fcm_token":    nil,
				"push_enabled": false,
				"updated_at":   now,
			}).Error
	})
}

func (s *Service) LogoutAll(ctx context.Context, user AuthenticatedUser) error {
	now := time.Now().UTC()
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&UserSession{}).
			Where("user_id = ? AND revoked_at IS NULL", user.UserID).
			Updates(map[string]any{"revoked_at": now, "updated_at": now}).Error; err != nil {
			return err
		}
		return tx.Model(&Device{}).
			Where("user_id = ?", user.UserID).
			Updates(map[string]any{
				"fcm_token":    nil,
				"push_enabled": false,
				"updated_at":   now,
			}).Error
	})
}

func (s *Service) Me(ctx context.Context, user AuthenticatedUser) (*UserResponse, error) {
	var dbUser User
	if err := s.db.WithContext(ctx).Where("id = ?", user.UserID).First(&dbUser).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	if err := validateUser(dbUser); err != nil {
		return nil, err
	}
	response := ToUserResponse(dbUser)
	return &response, nil
}

func (s *Service) DeleteAccount(ctx context.Context, user AuthenticatedUser) error {
	now := time.Now().UTC()
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&User{}).
			Where("id = ? AND deleted_at IS NULL", user.UserID).
			Updates(map[string]any{
				"status":     UserStatusDeleted,
				"deleted_at": now,
				"updated_at": now,
			}).Error; err != nil {
			return err
		}

		if err := tx.Model(&UserSession{}).
			Where("user_id = ? AND revoked_at IS NULL", user.UserID).
			Updates(map[string]any{"revoked_at": now, "updated_at": now}).Error; err != nil {
			return err
		}

		if err := tx.Table("profiles").
			Where("user_id = ?", user.UserID).
			Updates(map[string]any{
				"discovery_eligible":               false,
				"discovery_eligibility_updated_at": now,
				"updated_at":                       now,
			}).Error; err != nil {
			return err
		}

		return tx.Model(&Device{}).
			Where("user_id = ?", user.UserID).
			Updates(map[string]any{
				"fcm_token":    nil,
				"push_enabled": false,
				"updated_at":   now,
			}).Error
	})
}

func (s *Service) Authenticate(ctx context.Context, rawToken string) (*AuthenticatedUser, error) {
	claims, err := s.tokens.ParseAccessToken(rawToken)
	if err != nil {
		return nil, ErrUnauthorized
	}
	sessionUUID, err := uuid.Parse(claims.SessionID)
	if err != nil {
		return nil, ErrUnauthorized
	}

	now := time.Now().UTC()
	var session UserSession
	if err := s.db.WithContext(ctx).
		Preload("User").
		Where("uuid = ?", sessionUUID).
		First(&session).Error; err != nil {
		return nil, ErrUnauthorized
	}
	if err := validateSession(session, now); err != nil {
		return nil, err
	}
	if err := validateUser(session.User); err != nil {
		return nil, err
	}
	if err := s.canPerform(ctx, session.UserID, restriction.ActionAuthenticated); err != nil {
		return nil, err
	}
	if session.User.UUID.String() != claims.Subject {
		return nil, ErrUnauthorized
	}

	_ = s.db.WithContext(ctx).Model(&UserSession{}).
		Where("id = ?", session.ID).
		Updates(map[string]any{"last_used_at": now, "updated_at": now}).Error

	return &AuthenticatedUser{
		UserID:      session.UserID,
		UserUUID:    session.User.UUID,
		SessionID:   session.ID,
		SessionUUID: session.UUID,
	}, nil
}

func (s *Service) canPerform(ctx context.Context, userID uint64, action string) error {
	if s.restrictionChecker == nil {
		return nil
	}
	return s.restrictionChecker.CanPerform(ctx, userID, action)
}

func (s *Service) createSession(ctx context.Context, user User, req VerifyOTPRequest, meta RequestMeta, now time.Time) (*UserSession, string, error) {
	refreshToken, err := GenerateRefreshToken()
	if err != nil {
		return nil, "", err
	}

	session := &UserSession{
		UUID:             uuid.New(),
		UserID:           user.ID,
		User:             user,
		RefreshTokenHash: HashRefreshToken(refreshToken),
		TokenFamilyID:    uuid.New(),
		DeviceID:         optionalString(req.DeviceID),
		DeviceName:       optionalString(req.DeviceName),
		Platform:         optionalString(req.Platform),
		IPAddress:        meta.IPAddress,
		UserAgent:        meta.UserAgent,
		ExpiresAt:        now.Add(s.cfg.JWT.RefreshTTL()),
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	if err := s.db.WithContext(ctx).Create(session).Error; err != nil {
		return nil, "", err
	}

	return session, refreshToken, nil
}

func (s *Service) upsertDevice(ctx context.Context, tx *gorm.DB, userID uint64, req VerifyOTPRequest, now time.Time) error {
	if strings.TrimSpace(req.DeviceID) == "" {
		return nil
	}
	fcmToken := optionalString(req.FCMToken)
	if fcmToken != nil {
		if err := tx.WithContext(ctx).Model(&Device{}).
			Where("fcm_token = ? AND NOT (user_id = ? AND device_id = ?)", *fcmToken, userID, req.DeviceID).
			Updates(map[string]any{
				"fcm_token":    nil,
				"push_enabled": false,
				"updated_at":   now,
			}).Error; err != nil {
			return err
		}
	}

	device := Device{
		UUID:              uuid.New(),
		UserID:            userID,
		DeviceID:          req.DeviceID,
		DeviceName:        optionalString(req.DeviceName),
		Platform:          optionalString(req.Platform),
		FCMToken:          fcmToken,
		PushEnabled:       fcmToken != nil,
		FCMTokenUpdatedAt: nil,
		AppVersion:        optionalString(req.AppVersion),
		OSVersion:         optionalString(req.OSVersion),
		LastActiveAt:      &now,
		CreatedAt:         now,
		UpdatedAt:         now,
	}
	if fcmToken != nil {
		device.FCMTokenUpdatedAt = &now
	}

	updates := map[string]any{
		"device_name":    device.DeviceName,
		"platform":       device.Platform,
		"fcm_token":      device.FCMToken,
		"app_version":    device.AppVersion,
		"os_version":     device.OSVersion,
		"last_active_at": now,
		"updated_at":     now,
	}
	if fcmToken != nil {
		updates["push_enabled"] = true
		updates["fcm_token_updated_at"] = now
		updates["push_failure_count"] = 0
	} else {
		updates["push_enabled"] = false
	}

	return tx.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_id"}, {Name: "device_id"}},
		DoUpdates: clause.Assignments(updates),
	}).Create(&device).Error
}

func validateSession(session UserSession, now time.Time) error {
	if session.RevokedAt != nil {
		return ErrUnauthorized
	}
	if now.After(session.ExpiresAt) {
		return ErrUnauthorized
	}
	return nil
}

func validateUser(user User) error {
	if user.DeletedAt != nil || user.Status == UserStatusDeleted {
		return ErrAccountDeleted
	}
	if user.Status != UserStatusActive {
		return ErrInactiveUser
	}
	return nil
}

func deref(value *string) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(*value)
}

func optionalString(value string) *string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return &value
}

func IsClientError(err error) bool {
	return errors.Is(err, ErrUnauthorized) ||
		errors.Is(err, ErrInvalidRefresh) ||
		errors.Is(err, ErrDeviceMismatch) ||
		errors.Is(err, ErrInactiveUser) ||
		errors.Is(err, ErrAccountDeleted) ||
		errors.Is(err, restriction.ErrActionRestricted) ||
		errors.Is(err, otp.ErrInvalidChannel) ||
		errors.Is(err, otp.ErrInvalidPurpose) ||
		errors.Is(err, otp.ErrInvalidIdentifier) ||
		errors.Is(err, otp.ErrInvalidCode) ||
		errors.Is(err, otp.ErrOTPExpired) ||
		errors.Is(err, otp.ErrOTPMaxAttempts) ||
		errors.Is(err, otp.ErrRateLimited)
}

func PublicErrorMessage(err error) string {
	switch {
	case errors.Is(err, otp.ErrRateLimited):
		return "Too many OTP requests. Please try again later."
	case errors.Is(err, otp.ErrInvalidChannel), errors.Is(err, otp.ErrInvalidPurpose), errors.Is(err, otp.ErrInvalidIdentifier):
		return "Invalid OTP request."
	case errors.Is(err, otp.ErrInvalidCode), errors.Is(err, otp.ErrOTPExpired), errors.Is(err, otp.ErrOTPMaxAttempts):
		return "Invalid or expired OTP."
	case errors.Is(err, ErrInvalidRefresh):
		return "Invalid refresh token."
	case errors.Is(err, ErrDeviceMismatch):
		return "Device mismatch."
	case errors.Is(err, ErrInactiveUser), errors.Is(err, ErrAccountDeleted):
		return "Account is not active."
	case errors.Is(err, restriction.ErrActionRestricted):
		return "User action is restricted."
	case errors.Is(err, ErrUnauthorized):
		return "Unauthorized."
	default:
		return "Request failed."
	}
}

func PublicErrorCode(err error) string {
	switch {
	case errors.Is(err, otp.ErrRateLimited):
		return "RATE_LIMITED"
	case errors.Is(err, otp.ErrInvalidChannel), errors.Is(err, otp.ErrInvalidPurpose), errors.Is(err, otp.ErrInvalidIdentifier):
		return "VALIDATION_ERROR"
	case errors.Is(err, otp.ErrInvalidCode), errors.Is(err, otp.ErrOTPExpired), errors.Is(err, otp.ErrOTPMaxAttempts):
		return "INVALID_OTP"
	case errors.Is(err, ErrInvalidRefresh):
		return "INVALID_REFRESH_TOKEN"
	case errors.Is(err, ErrDeviceMismatch):
		return "DEVICE_MISMATCH"
	case errors.Is(err, ErrInactiveUser), errors.Is(err, ErrAccountDeleted):
		return "ACCOUNT_INACTIVE"
	case errors.Is(err, restriction.ErrActionRestricted):
		return "USER_ACTION_RESTRICTED"
	case errors.Is(err, ErrUnauthorized):
		return "UNAUTHORIZED"
	default:
		return "ERROR"
	}
}

func PublicStatusCode(err error) int {
	switch {
	case errors.Is(err, otp.ErrRateLimited):
		return 429
	case errors.Is(err, ErrUnauthorized), errors.Is(err, ErrInvalidRefresh), errors.Is(err, ErrInactiveUser), errors.Is(err, ErrAccountDeleted):
		return 401
	case errors.Is(err, restriction.ErrActionRestricted):
		return 403
	default:
		if IsClientError(err) {
			return 400
		}
		return 500
	}
}

func DebugError(err error) string {
	if err == nil {
		return ""
	}
	return fmt.Sprintf("%T: %v", err, err)
}

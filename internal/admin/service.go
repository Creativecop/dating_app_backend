package admin

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/neoscoder/aura-backend/internal/config"
)

type Service struct {
	db     *gorm.DB
	cfg    config.JWTConfig
	tokens *TokenService
}

func NewService(db *gorm.DB, cfg config.JWTConfig) *Service {
	return &Service{db: db, cfg: cfg, tokens: NewTokenService(cfg)}
}

func (s *Service) Login(ctx context.Context, req LoginRequest, meta RequestMeta) (*AuthResponse, error) {
	email := normalizeEmail(req.Email)
	if email == "" || strings.TrimSpace(req.Password) == "" {
		return nil, validationError("Email and password are required", nil)
	}

	var user AdminUser
	if err := s.db.WithContext(ctx).Where("email = ?", email).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, unauthorizedError()
		}
		return nil, err
	}
	if user.Status != StatusActive {
		return nil, ErrInactiveAdmin
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return nil, unauthorizedError()
	}

	now := time.Now().UTC()
	session, refreshToken, err := s.createSession(ctx, user.ID, meta, now)
	if err != nil {
		return nil, err
	}
	if err := s.db.WithContext(ctx).Model(&AdminUser{}).Where("id = ?", user.ID).Updates(map[string]any{
		"last_login_at": now,
		"updated_at":    now,
	}).Error; err != nil {
		return nil, err
	}
	user.LastLoginAt = &now

	accessToken, err := s.tokens.GenerateAccessToken(user, *session)
	if err != nil {
		return nil, err
	}
	permissions, err := s.Permissions(ctx, user.ID)
	if err != nil {
		return nil, err
	}
	return &AuthResponse{
		AccessToken:      accessToken,
		RefreshToken:     refreshToken,
		ExpiresIn:        int64(s.cfg.AccessTTL().Seconds()),
		RefreshExpiresIn: int64(s.cfg.RefreshTTL().Seconds()),
		Admin:            toAdminUserResponse(user, permissions),
	}, nil
}

func (s *Service) RefreshToken(ctx context.Context, req RefreshTokenRequest, meta RequestMeta) (*TokenResponse, error) {
	if strings.TrimSpace(req.RefreshToken) == "" {
		return nil, validationError("refreshToken is required", map[string]any{"field": "refreshToken"})
	}
	incomingHash := hashRefreshToken(req.RefreshToken)
	now := time.Now().UTC()
	var accessToken string
	var refreshToken string
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var session AdminSession
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Preload("AdminUser").
			Where("refresh_token_hash = ?", incomingHash).
			First(&session).Error; err != nil {
			return ErrInvalidRefresh
		}
		if session.RevokedAt != nil || session.ExpiresAt.Before(now) || session.AdminUser.Status != StatusActive {
			return ErrInvalidRefresh
		}
		nextRefresh, err := generateRefreshToken()
		if err != nil {
			return err
		}
		nextHash := hashRefreshToken(nextRefresh)
		if err := tx.Model(&AdminSession{}).Where("id = ?", session.ID).Updates(map[string]any{
			"refresh_token_hash": nextHash,
			"last_used_at":       now,
			"replaced_at":        now,
			"ip_address":         meta.IPAddress,
			"user_agent":         meta.UserAgent,
			"updated_at":         now,
		}).Error; err != nil {
			return err
		}
		session.RefreshTokenHash = nextHash
		session.LastUsedAt = &now
		session.ReplacedAt = &now
		token, err := s.tokens.GenerateAccessToken(session.AdminUser, session)
		if err != nil {
			return err
		}
		accessToken = token
		refreshToken = nextRefresh
		return nil
	})
	if err != nil {
		if errors.Is(err, ErrInvalidRefresh) {
			return nil, unauthorizedError()
		}
		return nil, err
	}
	return &TokenResponse{
		AccessToken:      accessToken,
		RefreshToken:     refreshToken,
		ExpiresIn:        int64(s.cfg.AccessTTL().Seconds()),
		RefreshExpiresIn: int64(s.cfg.RefreshTTL().Seconds()),
	}, nil
}

func (s *Service) Logout(ctx context.Context, adminID uint64, req LogoutRequest) error {
	now := time.Now().UTC()
	query := s.db.WithContext(ctx).Model(&AdminSession{}).Where("admin_user_id = ? AND revoked_at IS NULL", adminID)
	if strings.TrimSpace(req.RefreshToken) != "" {
		query = query.Where("refresh_token_hash = ?", hashRefreshToken(req.RefreshToken))
	}
	return query.Updates(map[string]any{"revoked_at": now, "updated_at": now}).Error
}

func (s *Service) Me(ctx context.Context, adminID uint64) (*AdminUserResponse, error) {
	var user AdminUser
	if err := s.db.WithContext(ctx).Where("id = ?", adminID).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, notFoundError("Admin user not found")
		}
		return nil, err
	}
	permissions, err := s.Permissions(ctx, adminID)
	if err != nil {
		return nil, err
	}
	response := toAdminUserResponse(user, permissions)
	return &response, nil
}

func (s *Service) Authenticate(ctx context.Context, rawToken string) (*AuthenticatedAdmin, error) {
	claims, err := s.tokens.ParseAccessToken(rawToken)
	if err != nil {
		return nil, ErrUnauthorized
	}
	adminUUID, err := uuid.Parse(claims.Subject)
	if err != nil {
		return nil, ErrUnauthorized
	}
	sessionUUID, err := uuid.Parse(claims.SessionID)
	if err != nil {
		return nil, ErrUnauthorized
	}
	var row struct {
		AdminUserID      uint64
		AdminUserUUID    uuid.UUID
		AdminSessionID   uint64
		AdminSessionUUID uuid.UUID
		Email            string
		Status           string
		ExpiresAt        time.Time
		RevokedAt        *time.Time
	}
	err = s.db.WithContext(ctx).Raw(`
		SELECT
		  au.id AS admin_user_id,
		  au.uuid AS admin_user_uuid,
		  s.id AS admin_session_id,
		  s.uuid AS admin_session_uuid,
		  au.email,
		  au.status,
		  s.expires_at,
		  s.revoked_at
		FROM admin_sessions s
		JOIN admin_users au ON au.id = s.admin_user_id
		WHERE au.uuid = ?
		  AND s.uuid = ?
	`, adminUUID, sessionUUID).Scan(&row).Error
	if err != nil {
		return nil, ErrUnauthorized
	}
	if row.AdminUserID == 0 || row.Status != StatusActive || row.RevokedAt != nil || row.ExpiresAt.Before(time.Now().UTC()) {
		return nil, ErrUnauthorized
	}
	return &AuthenticatedAdmin{
		AdminUserID:      row.AdminUserID,
		AdminUserUUID:    row.AdminUserUUID,
		AdminSessionID:   row.AdminSessionID,
		AdminSessionUUID: row.AdminSessionUUID,
		Email:            row.Email,
	}, nil
}

func (s *Service) HasPermission(ctx context.Context, adminID uint64, permission string) (bool, error) {
	var exists bool
	err := s.db.WithContext(ctx).Raw(`
		SELECT EXISTS (
		  SELECT 1
		  FROM admin_user_permissions
		  WHERE admin_user_id = ?
		    AND permission_code = ?
		)
	`, adminID, permission).Scan(&exists).Error
	return exists, err
}

func (s *Service) Permissions(ctx context.Context, adminID uint64) ([]string, error) {
	var permissions []string
	err := s.db.WithContext(ctx).Raw(`
		SELECT permission_code
		FROM admin_user_permissions
		WHERE admin_user_id = ?
		ORDER BY permission_code ASC
	`, adminID).Scan(&permissions).Error
	return permissions, err
}

func (s *Service) CreateSuperAdmin(ctx context.Context, email string, password string) error {
	email = normalizeEmail(email)
	if email == "" {
		return fmt.Errorf("email is required")
	}
	if len(password) < 12 {
		return fmt.Errorf("password must be at least 12 characters")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	permissions := []string{
		PermissionPaymentRequestsRead,
		PermissionPaymentRequestsApprove,
		PermissionPaymentRequestsReject,
	}
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		user := AdminUser{
			UUID:         uuid.New(),
			Email:        email,
			PasswordHash: string(hash),
			Status:       StatusActive,
			CreatedAt:    now,
			UpdatedAt:    now,
		}
		if err := tx.Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "email"}},
			DoUpdates: clause.Assignments(map[string]any{
				"password_hash": string(hash),
				"status":        StatusActive,
				"updated_at":    now,
			}),
		}).Create(&user).Error; err != nil {
			return err
		}
		if err := tx.Where("email = ?", email).First(&user).Error; err != nil {
			return err
		}
		for _, permission := range permissions {
			row := AdminUserPermission{
				UUID:           uuid.New(),
				AdminUserID:    user.ID,
				PermissionCode: permission,
				CreatedAt:      now,
			}
			if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&row).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (s *Service) createSession(ctx context.Context, adminID uint64, meta RequestMeta, now time.Time) (*AdminSession, string, error) {
	refreshToken, err := generateRefreshToken()
	if err != nil {
		return nil, "", err
	}
	session := AdminSession{
		UUID:             uuid.New(),
		AdminUserID:      adminID,
		RefreshTokenHash: hashRefreshToken(refreshToken),
		TokenFamilyID:    uuid.New(),
		IPAddress:        meta.IPAddress,
		UserAgent:        meta.UserAgent,
		ExpiresAt:        now.Add(s.cfg.RefreshTTL()),
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	if err := s.db.WithContext(ctx).Create(&session).Error; err != nil {
		return nil, "", err
	}
	return &session, refreshToken, nil
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

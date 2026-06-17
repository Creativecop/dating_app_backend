package admin

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/neoscoder/aura-backend/internal/config"
)

type Service struct {
	db                 *gorm.DB
	cfg                config.JWTConfig
	tokens             *TokenService
	socketDisconnecter UserSocketDisconnecter
}

func NewService(db *gorm.DB, cfg config.JWTConfig) *Service {
	return &Service{db: db, cfg: cfg, tokens: NewTokenService(cfg)}
}

type UserSocketDisconnecter interface {
	DisconnectUser(userID uint64) int
}

func (s *Service) SetUserSocketDisconnecter(disconnecter UserSocketDisconnecter) {
	s.socketDisconnecter = disconnecter
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
	if !loginAllowedStatus(user.Status) {
		return nil, ErrInactiveAdmin
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return nil, unauthorizedError()
	}
	roles, err := s.Roles(ctx, user.ID)
	if err != nil {
		return nil, err
	}
	if len(roles) == 0 {
		return nil, forbiddenError(CodeRoleRequired, "Admin role required", nil)
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

	accessToken, err := s.tokens.GenerateAccessToken(user, *session, roles)
	if err != nil {
		return nil, err
	}
	permissions := PermissionsForRoles(roles)
	return &AuthResponse{
		AccessToken:      accessToken,
		RefreshToken:     refreshToken,
		ExpiresIn:        int64(s.cfg.AccessTTL().Seconds()),
		RefreshExpiresIn: int64(s.cfg.RefreshTTL().Seconds()),
		Admin:            toAdminUserResponse(user, roles, permissions),
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
		if session.RevokedAt != nil || session.ExpiresAt.Before(now) || !loginAllowedStatus(session.AdminUser.Status) {
			return ErrInvalidRefresh
		}
		if !adminRefreshSessionStateValid(session.AdminUser, session.CreatedAt) {
			return ErrInvalidRefresh
		}
		roles, err := s.rolesWithDB(ctx, tx, session.AdminUserID)
		if err != nil {
			return err
		}
		if len(roles) == 0 {
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
		token, err := s.tokens.GenerateAccessToken(session.AdminUser, session, roles)
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

func (s *Service) ChangePassword(ctx context.Context, adminID uint64, req ChangePasswordRequest, meta RequestMeta) error {
	currentPassword := strings.TrimSpace(req.CurrentPassword)
	newPassword := strings.TrimSpace(req.NewPassword)
	if currentPassword == "" || newPassword == "" {
		return validationError("Current password and new password are required", nil)
	}
	if len(newPassword) < 12 {
		return validationError("newPassword must be at least 12 characters", map[string]any{"field": "newPassword"})
	}
	if currentPassword == newPassword {
		return validationError("newPassword must be different from currentPassword", map[string]any{"field": "newPassword"})
	}

	now := time.Now().UTC()
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var user AdminUser
		if err := tx.WithContext(ctx).Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ?", adminID).
			First(&user).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return notFoundError("Admin user not found")
			}
			return err
		}
		if !loginAllowedStatus(user.Status) {
			return ErrInactiveAdmin
		}
		if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(currentPassword)); err != nil {
			return unauthorizedError()
		}
		if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(newPassword)); err == nil {
			return validationError("newPassword must be different from currentPassword", map[string]any{"field": "newPassword"})
		}
		hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
		if err != nil {
			return err
		}
		nextStatus := user.Status
		if user.Status == StatusInvited {
			nextStatus = StatusActive
		}
		updates := map[string]any{
			"password_hash":        string(hash),
			"password_changed_at":  now,
			"must_change_password": false,
			"status":               nextStatus,
			"token_version":        gorm.Expr("token_version + 1"),
			"updated_at":           now,
		}
		if err := tx.WithContext(ctx).Model(&AdminUser{}).Where("id = ?", adminID).Updates(updates).Error; err != nil {
			return err
		}
		if err := tx.WithContext(ctx).Model(&AdminSession{}).
			Where("admin_user_id = ? AND revoked_at IS NULL", adminID).
			Updates(map[string]any{
				"revoked_at":     now,
				"revoked_reason": RevokedReasonPasswordChanged,
				"updated_at":     now,
			}).Error; err != nil {
			return err
		}
		return insertAdminAuditLogTx(ctx, tx, &adminID, AuditActorAdmin, "ADMIN_PASSWORD_CHANGED", "ADMIN_USER", &user.UUID, nil, map[string]any{
			"status":             user.Status,
			"mustChangePassword": user.MustChangePassword,
		}, map[string]any{
			"status":             nextStatus,
			"mustChangePassword": false,
			"sessionsRevoked":    true,
		}, meta)
	})
}

func (s *Service) Me(ctx context.Context, adminID uint64) (*AdminUserResponse, error) {
	var user AdminUser
	if err := s.db.WithContext(ctx).Where("id = ?", adminID).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, notFoundError("Admin user not found")
		}
		return nil, err
	}
	roles, err := s.Roles(ctx, adminID)
	if err != nil {
		return nil, err
	}
	permissions := PermissionsForRoles(roles)
	response := toAdminUserResponse(user, roles, permissions)
	return &response, nil
}

func (s *Service) Authenticate(ctx context.Context, rawToken string) (*AuthenticatedAdmin, error) {
	claims, err := s.tokens.ParseAccessToken(rawToken)
	if err != nil {
		return nil, ErrUnauthorized
	}
	if claims.TokenType != "admin_access" {
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
		AdminUserID        uint64
		AdminUserUUID      uuid.UUID
		AdminSessionID     uint64
		AdminSessionUUID   uuid.UUID
		Email              string
		Status             string
		MustChangePassword bool
		TokenVersion       int
		PasswordChangedAt  *time.Time
		ExpiresAt          time.Time
		RevokedAt          *time.Time
	}
	err = s.db.WithContext(ctx).Raw(`
		SELECT
		  au.id AS admin_user_id,
		  au.uuid AS admin_user_uuid,
		  s.id AS admin_session_id,
		  s.uuid AS admin_session_uuid,
		  au.email,
		  au.status,
		  au.must_change_password,
		  au.token_version,
		  au.password_changed_at,
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
	if row.AdminUserID == 0 || !loginAllowedStatus(row.Status) || row.RevokedAt != nil || row.ExpiresAt.Before(time.Now().UTC()) {
		return nil, ErrUnauthorized
	}
	if claims.TokenVersion != row.TokenVersion {
		return nil, ErrUnauthorized
	}
	if !adminTokenStateValid(AdminUser{TokenVersion: row.TokenVersion, PasswordChangedAt: row.PasswordChangedAt}, claims.IssuedAt) {
		return nil, ErrUnauthorized
	}
	roles, err := s.Roles(ctx, row.AdminUserID)
	if err != nil || len(roles) == 0 {
		return nil, ErrUnauthorized
	}
	return &AuthenticatedAdmin{
		AdminUserID:        row.AdminUserID,
		AdminUserUUID:      row.AdminUserUUID,
		AdminSessionID:     row.AdminSessionID,
		AdminSessionUUID:   row.AdminSessionUUID,
		Email:              row.Email,
		Status:             row.Status,
		MustChangePassword: row.MustChangePassword,
		Roles:              roles,
	}, nil
}

func (s *Service) HasPermission(ctx context.Context, adminID uint64, permission string) (bool, error) {
	roles, err := s.Roles(ctx, adminID)
	if err != nil {
		return false, err
	}
	for _, role := range roles {
		if RoleHasPermission(role, permission) {
			return true, nil
		}
	}
	return false, nil
}

func (s *Service) Permissions(ctx context.Context, adminID uint64) ([]string, error) {
	roles, err := s.Roles(ctx, adminID)
	if err != nil {
		return nil, err
	}
	return PermissionsForRoles(roles), nil
}

func (s *Service) Roles(ctx context.Context, adminID uint64) ([]string, error) {
	return s.rolesWithDB(ctx, s.db, adminID)
}

func (s *Service) rolesWithDB(ctx context.Context, db *gorm.DB, adminID uint64) ([]string, error) {
	var roles []string
	err := db.WithContext(ctx).Raw(`
		SELECT role
		FROM admin_role_assignments
		WHERE admin_user_id = ?
		  AND status = ?
		ORDER BY role ASC
	`, adminID, RoleAssignmentActive).Scan(&roles).Error
	return roles, err
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

func adminTokenStateValid(user AdminUser, issuedAt *jwt.NumericDate) bool {
	if user.TokenVersion < 1 {
		return false
	}
	if user.PasswordChangedAt == nil {
		return true
	}
	if issuedAt == nil {
		return false
	}
	return !issuedAt.Time.Before(*user.PasswordChangedAt)
}

func adminRefreshSessionStateValid(user AdminUser, sessionCreatedAt time.Time) bool {
	if user.TokenVersion < 1 {
		return false
	}
	if user.PasswordChangedAt == nil {
		return true
	}
	return !sessionCreatedAt.Before(*user.PasswordChangedAt)
}

func loginAllowedStatus(status string) bool {
	return status == StatusActive || status == StatusInvited
}

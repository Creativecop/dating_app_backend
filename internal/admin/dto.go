package admin

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

const ContextAdminKey = "admin_user"

type RequestMeta struct {
	IPAddress string
	UserAgent string
}

type AuthenticatedAdmin struct {
	AdminUserID      uint64
	AdminUserUUID    uuid.UUID
	AdminSessionID   uint64
	AdminSessionUUID uuid.UUID
	Email            string
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type RefreshTokenRequest struct {
	RefreshToken string `json:"refreshToken" binding:"required"`
}

type LogoutRequest struct {
	RefreshToken string `json:"refreshToken"`
}

type ChangePasswordRequest struct {
	CurrentPassword string `json:"currentPassword" binding:"required"`
	NewPassword     string `json:"newPassword" binding:"required"`
}

type AuthResponse struct {
	AccessToken      string            `json:"accessToken"`
	RefreshToken     string            `json:"refreshToken"`
	ExpiresIn        int64             `json:"expiresIn"`
	RefreshExpiresIn int64             `json:"refreshExpiresIn"`
	Admin            AdminUserResponse `json:"admin"`
}

type TokenResponse struct {
	AccessToken      string `json:"accessToken"`
	RefreshToken     string `json:"refreshToken"`
	ExpiresIn        int64  `json:"expiresIn"`
	RefreshExpiresIn int64  `json:"refreshExpiresIn"`
}

type AdminUserResponse struct {
	UUID        string     `json:"uuid"`
	Email       string     `json:"email"`
	Name        *string    `json:"name"`
	Status      string     `json:"status"`
	LastLoginAt *time.Time `json:"lastLoginAt"`
	Permissions []string   `json:"permissions,omitempty"`
}

func toAdminUserResponse(user AdminUser, permissions []string) AdminUserResponse {
	return AdminUserResponse{
		UUID:        user.UUID.String(),
		Email:       user.Email,
		Name:        user.Name,
		Status:      user.Status,
		LastLoginAt: user.LastLoginAt,
		Permissions: permissions,
	}
}

type AuditLogListQuery struct {
	AdminUserUUID string
	Action        string
	ResourceType  string
	ResourceUUID  string
	CreatedFrom   string
	CreatedTo     string
	Limit         string
	Cursor        string
}

type AuditLogListResponse struct {
	Items      []AuditLogResponse `json:"items"`
	NextCursor *string            `json:"nextCursor"`
}

type AuditLogResponse struct {
	AuditLogUUID   string          `json:"auditLogUuid"`
	AdminUserUUID  *string         `json:"adminUserUuid"`
	AdminEmail     *string         `json:"adminEmail"`
	Action         string          `json:"action"`
	ResourceType   string          `json:"resourceType"`
	ResourceUUID   *string         `json:"resourceUuid"`
	BeforeSnapshot json.RawMessage `json:"beforeSnapshot"`
	AfterSnapshot  json.RawMessage `json:"afterSnapshot"`
	IPAddress      *string         `json:"ipAddress"`
	UserAgent      *string         `json:"userAgent"`
	CreatedAt      time.Time       `json:"createdAt"`
}

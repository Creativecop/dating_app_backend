package admin

import (
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

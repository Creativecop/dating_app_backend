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
	AdminUserID        uint64
	AdminUserUUID      uuid.UUID
	AdminSessionID     uint64
	AdminSessionUUID   uuid.UUID
	Email              string
	Status             string
	MustChangePassword bool
	Roles              []string
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
	UUID               string     `json:"uuid"`
	Email              string     `json:"email"`
	Name               *string    `json:"name"`
	Status             string     `json:"status"`
	MustChangePassword bool       `json:"mustChangePassword"`
	LastLoginAt        *time.Time `json:"lastLoginAt"`
	Roles              []string   `json:"roles,omitempty"`
	Permissions        []string   `json:"permissions,omitempty"`
}

func toAdminUserResponse(user AdminUser, roles []string, permissions []string) AdminUserResponse {
	return AdminUserResponse{
		UUID:               user.UUID.String(),
		Email:              user.Email,
		Name:               user.Name,
		Status:             user.Status,
		MustChangePassword: user.MustChangePassword,
		LastLoginAt:        user.LastLoginAt,
		Roles:              roles,
		Permissions:        permissions,
	}
}

type RoleListResponse struct {
	Items []RoleResponse `json:"items"`
}

type RoleResponse struct {
	Role        string   `json:"role"`
	Permissions []string `json:"permissions"`
}

type AdminUserListResponse struct {
	Items []AdminUserResponse `json:"items"`
}

type CreateAdminUserRequest struct {
	Email             string   `json:"email" binding:"required"`
	Name              string   `json:"name"`
	TemporaryPassword string   `json:"temporaryPassword"`
	Roles             []string `json:"roles" binding:"required"`
	Reason            string   `json:"reason" binding:"required"`
}

type CreateAdminUserResponse struct {
	Admin             AdminUserResponse `json:"admin"`
	TemporaryPassword *string           `json:"temporaryPassword,omitempty"`
}

type AssignRoleRequest struct {
	Role   string `json:"role" binding:"required"`
	Reason string `json:"reason" binding:"required"`
}

type RemoveRoleRequest struct {
	Reason string `json:"reason" binding:"required"`
}

type UpdateAdminStatusRequest struct {
	Status string `json:"status" binding:"required"`
	Reason string `json:"reason" binding:"required"`
}

type AdminCapabilitiesResponse struct {
	Modules AdminModuleCapabilities `json:"modules"`
}

type AdminModuleCapabilities struct {
	TrustSafety    bool `json:"trustSafety"`
	Wallet         bool `json:"wallet"`
	Gift           bool `json:"gift"`
	Agency         bool `json:"agency"`
	Reseller       bool `json:"reseller"`
	Live           bool `json:"live"`
	LiveComments   bool `json:"liveComments"`
	ChatModeration bool `json:"chatModeration"`
}

type AdminMobileUserListQuery struct {
	Search      string
	Status      string
	CreatedFrom string
	CreatedTo   string
	Limit       string
	Cursor      string
}

type AdminMobileUserListResponse struct {
	Items      []AdminMobileUserListItem `json:"items"`
	NextCursor *string                   `json:"nextCursor"`
}

type AdminMobileUserListItem struct {
	UserUUID               string                   `json:"userUuid"`
	Phone                  *string                  `json:"phone,omitempty"`
	Email                  *string                  `json:"email,omitempty"`
	Status                 string                   `json:"status"`
	OnboardingStatus       string                   `json:"onboardingStatus"`
	Profile                *AdminUserProfileSummary `json:"profile,omitempty"`
	ActiveRestrictionCount int                      `json:"activeRestrictionCount"`
	LastLoginAt            *time.Time               `json:"lastLoginAt,omitempty"`
	CreatedAt              time.Time                `json:"createdAt"`
	UpdatedAt              time.Time                `json:"updatedAt"`
}

type AdminMobileUserDetailResponse struct {
	UserUUID         string                    `json:"userUuid"`
	Phone            *string                   `json:"phone,omitempty"`
	Email            *string                   `json:"email,omitempty"`
	Status           string                    `json:"status"`
	OnboardingStatus string                    `json:"onboardingStatus"`
	Profile          *AdminUserProfileSummary  `json:"profile,omitempty"`
	Restrictions     []UserRestrictionResponse `json:"activeRestrictions"`
	RecentReports    []AdminRecentReport       `json:"recentReports"`
	AuditHistory     []AuditLogResponse        `json:"auditHistory"`
	WalletSummary    any                       `json:"walletSummary"`
	LiveSummary      any                       `json:"liveSummary"`
	LastLoginAt      *time.Time                `json:"lastLoginAt,omitempty"`
	CreatedAt        time.Time                 `json:"createdAt"`
	UpdatedAt        time.Time                 `json:"updatedAt"`
}

type AdminUserProfileSummary struct {
	ProfileUUID   *string    `json:"profileUuid,omitempty"`
	DisplayName   *string    `json:"displayName,omitempty"`
	Gender        *string    `json:"gender,omitempty"`
	City          *string    `json:"city,omitempty"`
	Country       *string    `json:"country,omitempty"`
	ProfileStatus *string    `json:"profileStatus,omitempty"`
	CompletedAt   *time.Time `json:"completedAt,omitempty"`
}

type AdminRecentReport struct {
	ReportUUID string    `json:"reportUuid"`
	TargetType string    `json:"targetType"`
	ReasonCode string    `json:"reasonCode"`
	Status     string    `json:"status"`
	Severity   string    `json:"severity"`
	CreatedAt  time.Time `json:"createdAt"`
}

type UserRestrictionListResponse struct {
	Items []UserRestrictionResponse `json:"items"`
}

type UserRestrictionResponse struct {
	RestrictionUUID        string     `json:"restrictionUuid"`
	RestrictionType        string     `json:"restrictionType"`
	Status                 string     `json:"status"`
	Reason                 string     `json:"reason"`
	CreatedByAdminUserUUID *string    `json:"createdByAdminUserUuid,omitempty"`
	CreatedByAdminEmail    *string    `json:"createdByAdminEmail,omitempty"`
	RevokedByAdminUserUUID *string    `json:"revokedByAdminUserUuid,omitempty"`
	RevokedByAdminEmail    *string    `json:"revokedByAdminEmail,omitempty"`
	RevokedAt              *time.Time `json:"revokedAt,omitempty"`
	RevocationReason       *string    `json:"revocationReason,omitempty"`
	ExpiresAt              *time.Time `json:"expiresAt,omitempty"`
	CreatedAt              time.Time  `json:"createdAt"`
	UpdatedAt              time.Time  `json:"updatedAt"`
}

type CreateUserRestrictionRequest struct {
	RestrictionType string     `json:"restrictionType" binding:"required"`
	Reason          string     `json:"reason" binding:"required"`
	ExpiresAt       *time.Time `json:"expiresAt"`
}

type RevokeUserRestrictionRequest struct {
	Reason string `json:"reason" binding:"required"`
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
	ActorType      string          `json:"actorType"`
	Action         string          `json:"action"`
	ResourceType   string          `json:"resourceType"`
	ResourceUUID   *string         `json:"resourceUuid"`
	Reason         *string         `json:"reason,omitempty"`
	BeforeSnapshot json.RawMessage `json:"beforeSnapshot"`
	AfterSnapshot  json.RawMessage `json:"afterSnapshot"`
	IPAddress      *string         `json:"ipAddress"`
	UserAgent      *string         `json:"userAgent"`
	CreatedAt      time.Time       `json:"createdAt"`
}

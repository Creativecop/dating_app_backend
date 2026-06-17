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
	TrustSafety                  bool `json:"trustSafety"`
	SubscriptionPayments         bool `json:"subscriptionPayments"`
	SubscriptionPaymentAnalytics bool `json:"subscriptionPaymentAnalytics"`
	Wallet                       bool `json:"wallet"`
	Gift                         bool `json:"gift"`
	Agency                       bool `json:"agency"`
	Reseller                     bool `json:"reseller"`
	Live                         bool `json:"live"`
	LiveComments                 bool `json:"liveComments"`
	ChatModeration               bool `json:"chatModeration"`
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
	AdminUserUUID      string
	ActorAdminUserID   string
	ActorAdminUserUUID string
	Action             string
	ActionType         string
	ResourceType       string
	ResourceUUID       string
	CreatedFrom        string
	CreatedTo          string
	From               string
	To                 string
	Limit              string
	Cursor             string
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

type AnalyticsQuery struct {
	From string
	To   string
}

type AnalyticsPeriod struct {
	From time.Time `json:"from"`
	To   time.Time `json:"to"`
}

type DashboardSummaryResponse struct {
	Users                *DashboardUsersSummary                `json:"users,omitempty"`
	Reports              *DashboardReportsSummary              `json:"reports,omitempty"`
	Restrictions         *DashboardRestrictionsSummary         `json:"restrictions,omitempty"`
	Admin                *DashboardAdminSummary                `json:"admin,omitempty"`
	SubscriptionPayments *DashboardSubscriptionPaymentsSummary `json:"subscriptionPayments,omitempty"`
	Modules              AdminModuleCapabilities               `json:"modules"`
}

type DashboardUsersSummary struct {
	Total        int64 `json:"total"`
	NewToday     int64 `json:"newToday"`
	NewThisMonth int64 `json:"newThisMonth"`
}

type DashboardReportsSummary struct {
	Pending       int64 `json:"pending"`
	ReviewedToday int64 `json:"reviewedToday"`
	ActionedToday int64 `json:"actionedToday"`
}

type DashboardRestrictionsSummary struct {
	Active           int64 `json:"active"`
	FullPlatformBans int64 `json:"fullPlatformBans"`
	CommentBans      int64 `json:"commentBans"`
}

type DashboardAdminSummary struct {
	ActiveAdmins      int64 `json:"activeAdmins"`
	AuditActionsToday int64 `json:"auditActionsToday"`
}

type DashboardSubscriptionPaymentsSummary struct {
	Pending             int64 `json:"pending"`
	ApprovedToday       int64 `json:"approvedToday"`
	RejectedToday       int64 `json:"rejectedToday"`
	ApprovedAmountToday int64 `json:"approvedAmountToday"`
}

type UserAnalyticsResponse struct {
	Period           AnalyticsPeriod  `json:"period"`
	TotalUsers       int64            `json:"totalUsers"`
	NewUsers         int64            `json:"newUsers"`
	ActiveUsers      *int64           `json:"activeUsers"`
	ActivityNote     *string          `json:"activityNote,omitempty"`
	RestrictedUsers  int64            `json:"restrictedUsers"`
	FullPlatformBans int64            `json:"fullPlatformBans"`
	UsersByStatus    []AnalyticsCount `json:"usersByStatus"`
}

type ReportAnalyticsResponse struct {
	Period               AnalyticsPeriod     `json:"period"`
	ReportsCreated       int64               `json:"reportsCreated"`
	PendingReports       int64               `json:"pendingReports"`
	ReviewedReports      int64               `json:"reviewedReports"`
	DismissedReports     int64               `json:"dismissedReports"`
	ActionedReports      int64               `json:"actionedReports"`
	ReportsByTargetType  []AnalyticsCount    `json:"reportsByTargetType"`
	ReportsByReason      []ReportReasonCount `json:"reportsByReason"`
	AverageReviewMinutes *float64            `json:"averageReviewMinutes"`
}

type RestrictionAnalyticsResponse struct {
	Period              AnalyticsPeriod   `json:"period"`
	RestrictionsCreated int64             `json:"restrictionsCreated"`
	RestrictionsRevoked int64             `json:"restrictionsRevoked"`
	ActiveRestrictions  int64             `json:"activeRestrictions"`
	ExpiredRestrictions int64             `json:"expiredRestrictions"`
	RestrictionsByType  []AnalyticsCount  `json:"restrictionsByType"`
	RestrictionsByAdmin []AdminActorCount `json:"restrictionsByAdmin"`
}

type TrustSafetyAnalyticsResponse struct {
	Period       AnalyticsPeriod              `json:"period"`
	Reports      ReportAnalyticsResponse      `json:"reports"`
	Restrictions RestrictionAnalyticsResponse `json:"restrictions"`
}

type AdminActivityAnalyticsResponse struct {
	Period              AnalyticsPeriod   `json:"period"`
	AuditActionsByType  []AnalyticsCount  `json:"auditActionsByType"`
	AuditActionsByAdmin []AdminActorCount `json:"auditActionsByAdmin"`
	AdminLogins         *int64            `json:"adminLogins"`
	RoleChanges         int64             `json:"roleChanges"`
	RestrictionActions  int64             `json:"restrictionActions"`
	ReportReviewActions int64             `json:"reportReviewActions"`
}

type SubscriptionPaymentAnalyticsResponse struct {
	Period            AnalyticsPeriod           `json:"period"`
	Totals            SubscriptionPaymentTotals `json:"totals"`
	StatusBreakdown   []PaymentBreakdown        `json:"statusBreakdown"`
	ProviderBreakdown []PaymentBreakdown        `json:"providerBreakdown"`
	PlanBreakdown     []PaymentPlanBreakdown    `json:"planBreakdown"`
	ReviewSLA         PaymentReviewSLA          `json:"reviewSla"`
}

type SubscriptionPaymentTotals struct {
	PaymentCount     int64 `json:"paymentCount"`
	PendingCount     int64 `json:"pendingCount"`
	UnderReviewCount int64 `json:"underReviewCount"`
	ApprovedCount    int64 `json:"approvedCount"`
	RejectedCount    int64 `json:"rejectedCount"`
	CanceledCount    int64 `json:"canceledCount"`
	TotalAmount      int64 `json:"totalAmount"`
	ApprovedAmount   int64 `json:"approvedAmount"`
	RejectedAmount   int64 `json:"rejectedAmount"`
}

type PaymentBreakdown struct {
	Key    string `json:"key"`
	Count  int64  `json:"count"`
	Amount int64  `json:"amount"`
}

type PaymentPlanBreakdown struct {
	PlanID   *string `json:"planId,omitempty"`
	PlanCode string  `json:"planCode"`
	PlanName string  `json:"planName"`
	Count    int64   `json:"count"`
	Amount   int64   `json:"amount"`
}

type PaymentReviewSLA struct {
	AverageReviewMinutes *float64 `json:"averageReviewMinutes"`
	PendingOlderThan24h  int64    `json:"pendingOlderThan24h"`
}

type AnalyticsCount struct {
	Key   string `json:"key"`
	Count int64  `json:"count"`
}

type ReportReasonCount struct {
	ReasonCode  string `json:"reasonCode"`
	ReasonTitle string `json:"reasonTitle"`
	Count       int64  `json:"count"`
}

type AdminActorCount struct {
	AdminUserUUID *string `json:"adminUserUuid,omitempty"`
	AdminEmail    *string `json:"adminEmail,omitempty"`
	ActorType     *string `json:"actorType,omitempty"`
	Count         int64   `json:"count"`
}

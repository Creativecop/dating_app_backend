package admin

import (
	"time"

	"github.com/google/uuid"
)

const (
	StatusInvited  = "INVITED"
	StatusActive   = "ACTIVE"
	StatusDisabled = "DISABLED"
	StatusLocked   = "LOCKED"

	RoleSuperAdmin      = "SUPER_ADMIN"
	RolePlatformAdmin   = "PLATFORM_ADMIN"
	RoleOpsManager      = "OPS_MANAGER"
	RoleFinanceAdmin    = "FINANCE_ADMIN"
	RoleTrustSafety     = "TRUST_SAFETY"
	RoleSupportAgent    = "SUPPORT_AGENT"
	RoleGiftManager     = "GIFT_MANAGER"
	RoleAgencyManager   = "AGENCY_MANAGER"
	RoleResellerManager = "RESELLER_MANAGER"
	RoleAnalyst         = "ANALYST"

	RoleAssignmentActive  = "ACTIVE"
	RoleAssignmentRemoved = "REMOVED"

	AuditActorAdmin  = "ADMIN"
	AuditActorSystem = "SYSTEM"

	RestrictionActive  = "ACTIVE"
	RestrictionRevoked = "REVOKED"

	UserRestrictionFullPlatformBan      = "FULL_PLATFORM_BAN"
	UserRestrictionLiveCreateBan        = "LIVE_CREATE_BAN"
	UserRestrictionCommentBan           = "COMMENT_BAN"
	UserRestrictionGiftSendBan          = "GIFT_SEND_BAN"
	UserRestrictionResellerOperationBan = "RESELLER_OPERATION_BAN"
	UserRestrictionAgencyOperationBan   = "AGENCY_OPERATION_BAN"

	PermissionUsersRead        = "users.read"
	PermissionUsersRestrict    = "users.restrict"
	PermissionLivesMonitor     = "lives.monitor"
	PermissionLivesForceEnd    = "lives.force_end"
	PermissionReportsReview    = "reports.review"
	PermissionGiftsManage      = "gifts.manage"
	PermissionWalletAudit      = "wallet.audit"
	PermissionWalletAdjust     = "wallet.adjust"
	PermissionAgencyManage     = "agency.manage"
	PermissionResellerManage   = "reseller.manage"
	PermissionResellerAllocate = "reseller.allocate"
	PermissionAnalyticsRead    = "analytics.read"
	PermissionAuditRead        = "admin.audit.read"
	PermissionRolesManage      = "admin.roles.manage"
	PermissionAdminUsersRead   = "admin.users.read"
	PermissionAdminUsersCreate = "admin.users.create"
	PermissionAdminUsersManage = "admin.users.manage"

	PermissionPaymentRequestsRead    = "subscriptions.payment_requests.read"
	PermissionPaymentRequestsApprove = "subscriptions.payment_requests.approve"
	PermissionPaymentRequestsReject  = "subscriptions.payment_requests.reject"

	RevokedReasonPasswordChanged = "PASSWORD_CHANGED"
	RevokedReasonRoleChanged     = "ROLE_CHANGED"
	RevokedReasonStatusChanged   = "STATUS_CHANGED"
)

type AdminUser struct {
	ID                 uint64    `gorm:"primaryKey"`
	UUID               uuid.UUID `gorm:"type:uuid;uniqueIndex"`
	Email              string
	Name               *string
	PasswordHash       string
	Status             string
	MustChangePassword bool
	LastLoginAt        *time.Time
	PasswordChangedAt  *time.Time
	TokenVersion       int
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

func (AdminUser) TableName() string {
	return "admin_users"
}

type AdminSession struct {
	ID               uint64    `gorm:"primaryKey"`
	UUID             uuid.UUID `gorm:"type:uuid;uniqueIndex"`
	AdminUserID      uint64
	AdminUser        AdminUser
	RefreshTokenHash string
	TokenFamilyID    uuid.UUID `gorm:"type:uuid"`
	IPAddress        string
	UserAgent        string
	ExpiresAt        time.Time
	LastUsedAt       *time.Time
	RevokedAt        *time.Time
	RevokedReason    *string
	ReplacedAt       *time.Time
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

func (AdminSession) TableName() string {
	return "admin_sessions"
}

type AdminRoleAssignment struct {
	ID                    uint64    `gorm:"primaryKey"`
	UUID                  uuid.UUID `gorm:"type:uuid;uniqueIndex"`
	AdminUserID           uint64
	AdminUser             AdminUser
	Role                  string
	Status                string
	AssignedByAdminUserID *uint64
	AssignedByAdminUser   *AdminUser
	RemovedByAdminUserID  *uint64
	RemovedByAdminUser    *AdminUser
	AssignedAt            time.Time
	RemovedAt             *time.Time
	Reason                *string
	CreatedAt             time.Time
	UpdatedAt             time.Time
}

func (AdminRoleAssignment) TableName() string {
	return "admin_role_assignments"
}

type AdminUserPermission struct {
	ID             uint64    `gorm:"primaryKey"`
	UUID           uuid.UUID `gorm:"type:uuid;uniqueIndex"`
	AdminUserID    uint64
	PermissionCode string
	CreatedAt      time.Time
}

func (AdminUserPermission) TableName() string {
	return "admin_user_permissions"
}

type AdminAuditLog struct {
	ID             uint64    `gorm:"primaryKey"`
	UUID           uuid.UUID `gorm:"type:uuid;uniqueIndex"`
	AdminUserID    *uint64
	ActorType      string
	Action         string
	ResourceType   string
	ResourceUUID   *uuid.UUID `gorm:"type:uuid"`
	Reason         *string
	BeforeSnapshot []byte
	AfterSnapshot  []byte
	IPAddress      *string
	UserAgent      *string
	CreatedAt      time.Time
}

func (AdminAuditLog) TableName() string {
	return "admin_audit_logs"
}

type UserRestriction struct {
	ID                   uint64    `gorm:"primaryKey"`
	UUID                 uuid.UUID `gorm:"type:uuid;uniqueIndex"`
	UserID               uint64
	RestrictionType      string
	Status               string
	Reason               string
	CreatedByAdminUserID uint64
	RevokedByAdminUserID *uint64
	RevokedAt            *time.Time
	RevocationReason     *string
	ExpiresAt            *time.Time
	CreatedAt            time.Time
	UpdatedAt            time.Time
}

func (UserRestriction) TableName() string {
	return "user_restrictions"
}

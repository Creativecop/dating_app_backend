package admin

import "sort"

var validRoles = map[string]struct{}{
	RoleSuperAdmin:      {},
	RolePlatformAdmin:   {},
	RoleOpsManager:      {},
	RoleFinanceAdmin:    {},
	RoleTrustSafety:     {},
	RoleSupportAgent:    {},
	RoleGiftManager:     {},
	RoleAgencyManager:   {},
	RoleResellerManager: {},
	RoleAnalyst:         {},
}

var rolePermissions = map[string][]string{
	RolePlatformAdmin: {
		PermissionUsersRead,
		PermissionUsersRestrict,
		PermissionLivesMonitor,
		PermissionLivesForceEnd,
		PermissionReportsReview,
		PermissionCommentsModerate,
		PermissionGiftsManage,
		PermissionAgencyManage,
		PermissionResellerManage,
		PermissionResellerAllocate,
		PermissionAnalyticsRead,
		PermissionAnalyticsReportsRead,
		PermissionAnalyticsRestrictionsRead,
		PermissionAnalyticsTrustSafetyRead,
		PermissionAnalyticsAdminActivityRead,
		PermissionAnalyticsSubscriptionPaymentsRead,
		PermissionAuditRead,
		PermissionRolesManage,
		PermissionAdminUsersRead,
		PermissionAdminUsersCreate,
		PermissionAdminUsersManage,
		PermissionPaymentRequestsRead,
		PermissionPaymentRequestsApprove,
		PermissionPaymentRequestsReject,
	},
	RoleOpsManager: {
		PermissionUsersRead,
		PermissionUsersRestrict,
		PermissionLivesMonitor,
		PermissionLivesForceEnd,
		PermissionReportsReview,
		PermissionCommentsModerate,
		PermissionGiftsManage,
		PermissionAgencyManage,
		PermissionResellerManage,
		PermissionAnalyticsRead,
		PermissionAnalyticsReportsRead,
		PermissionAnalyticsRestrictionsRead,
		PermissionAnalyticsTrustSafetyRead,
		PermissionAnalyticsAdminActivityRead,
	},
	RoleFinanceAdmin: {
		PermissionWalletAudit,
		PermissionWalletAdjust,
		PermissionResellerAllocate,
		PermissionAnalyticsSubscriptionPaymentsRead,
		PermissionPaymentRequestsRead,
		PermissionPaymentRequestsApprove,
		PermissionPaymentRequestsReject,
	},
	RoleTrustSafety: {
		PermissionUsersRead,
		PermissionUsersRestrict,
		PermissionReportsReview,
		PermissionCommentsModerate,
		PermissionLivesMonitor,
		PermissionLivesForceEnd,
		PermissionAnalyticsReportsRead,
		PermissionAnalyticsRestrictionsRead,
		PermissionAnalyticsTrustSafetyRead,
	},
	RoleSupportAgent: {
		PermissionUsersRead,
		PermissionLivesMonitor,
		PermissionReportsReview,
	},
	RoleGiftManager: {
		PermissionGiftsManage,
	},
	RoleAgencyManager: {
		PermissionAgencyManage,
	},
	RoleResellerManager: {
		PermissionResellerManage,
		PermissionResellerAllocate,
	},
	RoleAnalyst: {
		PermissionAnalyticsRead,
		PermissionAnalyticsReportsRead,
		PermissionAnalyticsRestrictionsRead,
		PermissionAnalyticsTrustSafetyRead,
		PermissionAnalyticsAdminActivityRead,
		PermissionAnalyticsSubscriptionPaymentsRead,
	},
}

func IsValidRole(role string) bool {
	_, ok := validRoles[role]
	return ok
}

func AllRoles() []string {
	roles := make([]string, 0, len(validRoles))
	for role := range validRoles {
		roles = append(roles, role)
	}
	sort.Strings(roles)
	return roles
}

func AllPermissions() []string {
	set := map[string]struct{}{
		PermissionUsersRead:                         {},
		PermissionUsersRestrict:                     {},
		PermissionLivesMonitor:                      {},
		PermissionLivesForceEnd:                     {},
		PermissionReportsReview:                     {},
		PermissionCommentsModerate:                  {},
		PermissionGiftsManage:                       {},
		PermissionWalletAudit:                       {},
		PermissionWalletAdjust:                      {},
		PermissionAgencyManage:                      {},
		PermissionResellerManage:                    {},
		PermissionResellerAllocate:                  {},
		PermissionAnalyticsRead:                     {},
		PermissionAnalyticsReportsRead:              {},
		PermissionAnalyticsRestrictionsRead:         {},
		PermissionAnalyticsTrustSafetyRead:          {},
		PermissionAnalyticsAdminActivityRead:        {},
		PermissionAnalyticsSubscriptionPaymentsRead: {},
		PermissionAuditRead:                         {},
		PermissionRolesManage:                       {},
		PermissionAdminUsersRead:                    {},
		PermissionAdminUsersCreate:                  {},
		PermissionAdminUsersManage:                  {},
		PermissionPaymentRequestsRead:               {},
		PermissionPaymentRequestsApprove:            {},
		PermissionPaymentRequestsReject:             {},
	}
	permissions := make([]string, 0, len(set))
	for permission := range set {
		permissions = append(permissions, permission)
	}
	sort.Strings(permissions)
	return permissions
}

func PermissionsForRoles(roles []string) []string {
	if hasString(roles, RoleSuperAdmin) {
		return AllPermissions()
	}
	set := map[string]struct{}{}
	for _, role := range roles {
		for _, permission := range rolePermissions[role] {
			set[permission] = struct{}{}
		}
	}
	permissions := make([]string, 0, len(set))
	for permission := range set {
		permissions = append(permissions, permission)
	}
	sort.Strings(permissions)
	return permissions
}

func RoleHasPermission(role string, permission string) bool {
	if role == RoleSuperAdmin {
		return true
	}
	for _, rolePermission := range rolePermissions[role] {
		if rolePermission == permission {
			return true
		}
	}
	return false
}

func hasString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

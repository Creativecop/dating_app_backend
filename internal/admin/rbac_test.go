package admin

import "testing"

func TestSuperAdminGetsAllPermissions(t *testing.T) {
	permissions := PermissionsForRoles([]string{RoleSuperAdmin})
	if len(permissions) != len(AllPermissions()) {
		t.Fatalf("SUPER_ADMIN permissions = %d, want %d", len(permissions), len(AllPermissions()))
	}
	for _, permission := range AllPermissions() {
		if !RoleHasPermission(RoleSuperAdmin, permission) {
			t.Fatalf("SUPER_ADMIN missing permission %s", permission)
		}
	}
}

func TestRolePermissionMatrix(t *testing.T) {
	tests := []struct {
		role       string
		allowed    string
		disallowed string
	}{
		{RoleFinanceAdmin, PermissionWalletAdjust, PermissionRolesManage},
		{RoleTrustSafety, PermissionReportsReview, PermissionWalletAdjust},
		{RoleSupportAgent, PermissionUsersRead, PermissionUsersRestrict},
		{RoleGiftManager, PermissionGiftsManage, PermissionAgencyManage},
		{RoleAgencyManager, PermissionAgencyManage, PermissionGiftsManage},
		{RoleResellerManager, PermissionResellerAllocate, PermissionWalletAdjust},
		{RoleAnalyst, PermissionAnalyticsRead, PermissionUsersRead},
	}

	for _, test := range tests {
		t.Run(test.role, func(t *testing.T) {
			if !RoleHasPermission(test.role, test.allowed) {
				t.Fatalf("%s should allow %s", test.role, test.allowed)
			}
			if RoleHasPermission(test.role, test.disallowed) {
				t.Fatalf("%s should not allow %s", test.role, test.disallowed)
			}
		})
	}
}

func TestNormalizeRoleList(t *testing.T) {
	roles, err := normalizeRoleList([]string{"ops_manager", "OPS_MANAGER", "analyst"})
	if err != nil {
		t.Fatalf("normalizeRoleList returned error: %v", err)
	}
	if len(roles) != 2 || roles[0] != RoleOpsManager || roles[1] != RoleAnalyst {
		t.Fatalf("roles = %#v", roles)
	}
	if _, err := normalizeRoleList([]string{"NOPE"}); err == nil {
		t.Fatal("expected invalid role to fail")
	}
}

func TestAdminStatusValidation(t *testing.T) {
	for _, status := range []string{StatusInvited, StatusActive, StatusDisabled, StatusLocked} {
		if !isValidAdminStatus(status) {
			t.Fatalf("%s should be valid", status)
		}
	}
	if isValidAdminStatus("SUSPENDED") {
		t.Fatal("SUSPENDED should not be a Phase 13A admin status")
	}
}

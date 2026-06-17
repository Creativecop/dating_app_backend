package admin

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func (s *Service) ListRoles(ctx context.Context) (*RoleListResponse, error) {
	roles := AllRoles()
	items := make([]RoleResponse, 0, len(roles))
	for _, role := range roles {
		items = append(items, RoleResponse{
			Role:        role,
			Permissions: PermissionsForRoles([]string{role}),
		})
	}
	return &RoleListResponse{Items: items}, nil
}

func (s *Service) ListAdminUsers(ctx context.Context) (*AdminUserListResponse, error) {
	var users []AdminUser
	if err := s.db.WithContext(ctx).Order("created_at DESC, id DESC").Find(&users).Error; err != nil {
		return nil, err
	}
	items := make([]AdminUserResponse, 0, len(users))
	for _, user := range users {
		item, err := s.adminUserResponse(ctx, s.db, user)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return &AdminUserListResponse{Items: items}, nil
}

func (s *Service) AdminUserDetail(ctx context.Context, adminUserUUID string) (*AdminUserResponse, error) {
	parsed, err := uuid.Parse(strings.TrimSpace(adminUserUUID))
	if err != nil {
		return nil, validationError("adminUserUuid is invalid", map[string]any{"field": "adminUserUuid"})
	}
	var user AdminUser
	if err := s.db.WithContext(ctx).Where("uuid = ?", parsed).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, targetNotFoundError("Admin user not found")
		}
		return nil, err
	}
	response, err := s.adminUserResponse(ctx, s.db, user)
	if err != nil {
		return nil, err
	}
	return &response, nil
}

func (s *Service) CreateAdminUser(ctx context.Context, actorID uint64, req CreateAdminUserRequest, meta RequestMeta) (*CreateAdminUserResponse, error) {
	email := normalizeEmail(req.Email)
	if email == "" {
		return nil, validationError("email is required", map[string]any{"field": "email"})
	}
	reason, err := requireReason(req.Reason)
	if err != nil {
		return nil, err
	}
	roles, err := normalizeRoleList(req.Roles)
	if err != nil {
		return nil, err
	}
	temporaryPassword := strings.TrimSpace(req.TemporaryPassword)
	generatedPassword := false
	if temporaryPassword == "" {
		temporaryPassword, err = generateTemporaryPassword()
		if err != nil {
			return nil, err
		}
		generatedPassword = true
	}
	if err := validateAdminPassword(temporaryPassword, "temporaryPassword"); err != nil {
		return nil, err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(temporaryPassword), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	var created AdminUser
	if err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		actorIsSuper, err := s.hasActiveRoleTx(ctx, tx, actorID, RoleSuperAdmin)
		if err != nil {
			return err
		}
		if hasString(roles, RoleSuperAdmin) && !actorIsSuper {
			return forbiddenError(CodeRoleAssignmentNotAllowed, "Only SUPER_ADMIN can assign SUPER_ADMIN", nil)
		}

		var existing AdminUser
		if err := tx.WithContext(ctx).Where("email = ?", email).First(&existing).Error; err == nil {
			return conflictError("Admin email already exists")
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}

		var name *string
		if strings.TrimSpace(req.Name) != "" {
			value := strings.TrimSpace(req.Name)
			name = &value
		}
		created = AdminUser{
			UUID:               uuid.New(),
			Email:              email,
			Name:               name,
			PasswordHash:       string(hash),
			Status:             StatusInvited,
			MustChangePassword: true,
			TokenVersion:       1,
			CreatedAt:          now,
			UpdatedAt:          now,
		}
		if err := tx.WithContext(ctx).Create(&created).Error; err != nil {
			return err
		}

		for _, role := range roles {
			assignment := AdminRoleAssignment{
				UUID:                  uuid.New(),
				AdminUserID:           created.ID,
				Role:                  role,
				Status:                RoleAssignmentActive,
				AssignedByAdminUserID: &actorID,
				AssignedAt:            now,
				Reason:                &reason,
				CreatedAt:             now,
				UpdatedAt:             now,
			}
			if err := tx.WithContext(ctx).Create(&assignment).Error; err != nil {
				return err
			}
		}

		return insertAdminAuditLogTx(ctx, tx, &actorID, AuditActorAdmin, "ADMIN_USER_CREATED", "ADMIN_USER", &created.UUID, &reason, map[string]any{}, map[string]any{
			"email":              created.Email,
			"name":               created.Name,
			"status":             created.Status,
			"mustChangePassword": true,
			"roles":              roles,
		}, meta)
	}); err != nil {
		return nil, err
	}

	item, err := s.adminUserResponse(ctx, s.db, created)
	if err != nil {
		return nil, err
	}
	response := &CreateAdminUserResponse{Admin: item}
	if generatedPassword {
		response.TemporaryPassword = &temporaryPassword
	}
	return response, nil
}

func (s *Service) AssignAdminRole(ctx context.Context, actorID uint64, targetUUID string, req AssignRoleRequest, meta RequestMeta) (*AdminUserResponse, error) {
	role := strings.ToUpper(strings.TrimSpace(req.Role))
	if !IsValidRole(role) {
		return nil, validationError("role is invalid", map[string]any{"field": "role"})
	}
	reason, err := requireReason(req.Reason)
	if err != nil {
		return nil, err
	}

	var target AdminUser
	if err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var err error
		target, err = s.adminUserByUUIDTx(ctx, tx, targetUUID, true)
		if err != nil {
			return err
		}
		if err := s.ensureRoleMutationAllowedTx(ctx, tx, actorID, target.ID, role); err != nil {
			return err
		}

		var exists bool
		if err := tx.WithContext(ctx).Raw(`
			SELECT EXISTS (
			  SELECT 1
			  FROM admin_role_assignments
			  WHERE admin_user_id = ?
			    AND role = ?
			    AND status = ?
			)
		`, target.ID, role, RoleAssignmentActive).Scan(&exists).Error; err != nil {
			return err
		}
		if exists {
			return conflictCodeError(CodeRoleAlreadyAssigned, "Role is already assigned")
		}

		beforeRoles, err := s.rolesWithDB(ctx, tx, target.ID)
		if err != nil {
			return err
		}
		assignment := AdminRoleAssignment{
			UUID:                  uuid.New(),
			AdminUserID:           target.ID,
			Role:                  role,
			Status:                RoleAssignmentActive,
			AssignedByAdminUserID: &actorID,
			AssignedAt:            time.Now().UTC(),
			Reason:                &reason,
		}
		if err := tx.WithContext(ctx).Create(&assignment).Error; err != nil {
			return err
		}
		if err := s.invalidateAdminTokensTx(ctx, tx, target.ID, RevokedReasonRoleChanged); err != nil {
			return err
		}
		afterRoles, err := s.rolesWithDB(ctx, tx, target.ID)
		if err != nil {
			return err
		}
		return insertAdminAuditLogTx(ctx, tx, &actorID, AuditActorAdmin, "ADMIN_ROLE_ASSIGNED", "ADMIN_USER", &target.UUID, &reason, map[string]any{
			"roles": beforeRoles,
		}, map[string]any{
			"roles": afterRoles,
			"role":  role,
		}, meta)
	}); err != nil {
		return nil, err
	}
	response, err := s.adminUserResponse(ctx, s.db, target)
	if err != nil {
		return nil, err
	}
	return &response, nil
}

func (s *Service) RemoveAdminRole(ctx context.Context, actorID uint64, targetUUID string, role string, req RemoveRoleRequest, meta RequestMeta) (*AdminUserResponse, error) {
	role = strings.ToUpper(strings.TrimSpace(role))
	if !IsValidRole(role) {
		return nil, validationError("role is invalid", map[string]any{"field": "role"})
	}
	reason, err := requireReason(req.Reason)
	if err != nil {
		return nil, err
	}

	var target AdminUser
	if err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var err error
		target, err = s.adminUserByUUIDTx(ctx, tx, targetUUID, true)
		if err != nil {
			return err
		}
		if err := s.ensureRoleMutationAllowedTx(ctx, tx, actorID, target.ID, role); err != nil {
			return err
		}

		var assignment AdminRoleAssignment
		if err := tx.WithContext(ctx).Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("admin_user_id = ? AND role = ? AND status = ?", target.ID, role, RoleAssignmentActive).
			First(&assignment).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return notFoundError("Admin role assignment not found")
			}
			return err
		}
		if role == RoleSuperAdmin && target.Status == StatusActive {
			count, err := s.lockActiveSuperAdminRolesTx(ctx, tx)
			if err != nil {
				return err
			}
			if count <= 1 {
				return actionNotAllowedError("Cannot remove the last active SUPER_ADMIN")
			}
		}
		beforeRoles, err := s.rolesWithDB(ctx, tx, target.ID)
		if err != nil {
			return err
		}
		now := time.Now().UTC()
		if err := tx.WithContext(ctx).Model(&AdminRoleAssignment{}).Where("id = ?", assignment.ID).Updates(map[string]any{
			"status":                   RoleAssignmentRemoved,
			"removed_by_admin_user_id": actorID,
			"removed_at":               now,
			"reason":                   reason,
			"updated_at":               now,
		}).Error; err != nil {
			return err
		}
		if err := s.invalidateAdminTokensTx(ctx, tx, target.ID, RevokedReasonRoleChanged); err != nil {
			return err
		}
		afterRoles, err := s.rolesWithDB(ctx, tx, target.ID)
		if err != nil {
			return err
		}
		return insertAdminAuditLogTx(ctx, tx, &actorID, AuditActorAdmin, "ADMIN_ROLE_REMOVED", "ADMIN_USER", &target.UUID, &reason, map[string]any{
			"roles": beforeRoles,
		}, map[string]any{
			"roles": afterRoles,
			"role":  role,
		}, meta)
	}); err != nil {
		return nil, err
	}
	response, err := s.adminUserResponse(ctx, s.db, target)
	if err != nil {
		return nil, err
	}
	return &response, nil
}

func (s *Service) UpdateAdminStatus(ctx context.Context, actorID uint64, targetUUID string, req UpdateAdminStatusRequest, meta RequestMeta) (*AdminUserResponse, error) {
	nextStatus := strings.ToUpper(strings.TrimSpace(req.Status))
	if !isValidAdminStatus(nextStatus) {
		return nil, validationError("status is invalid", map[string]any{"field": "status"})
	}
	reason, err := requireReason(req.Reason)
	if err != nil {
		return nil, err
	}

	var target AdminUser
	if err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var err error
		target, err = s.adminUserByUUIDTx(ctx, tx, targetUUID, true)
		if err != nil {
			return err
		}
		targetHasSuper, err := s.hasActiveRoleTx(ctx, tx, target.ID, RoleSuperAdmin)
		if err != nil {
			return err
		}
		actorIsSuper, err := s.hasActiveRoleTx(ctx, tx, actorID, RoleSuperAdmin)
		if err != nil {
			return err
		}
		if targetHasSuper && !actorIsSuper {
			return forbiddenError(CodeCannotModifySuperAdmin, "Cannot modify SUPER_ADMIN", nil)
		}
		if targetHasSuper && target.Status == StatusActive && nextStatus != StatusActive {
			count, err := s.lockActiveSuperAdminRolesTx(ctx, tx)
			if err != nil {
				return err
			}
			if count <= 1 {
				return actionNotAllowedError("Cannot disable the last active SUPER_ADMIN")
			}
		}
		if target.Status == nextStatus {
			return conflictError("Admin user already has requested status")
		}

		before := map[string]any{
			"status":             target.Status,
			"mustChangePassword": target.MustChangePassword,
		}
		updates := map[string]any{
			"status":        nextStatus,
			"token_version": gorm.Expr("token_version + 1"),
			"updated_at":    time.Now().UTC(),
		}
		if nextStatus == StatusInvited {
			updates["must_change_password"] = true
		}
		if err := tx.WithContext(ctx).Model(&AdminUser{}).Where("id = ?", target.ID).Updates(updates).Error; err != nil {
			return err
		}
		if nextStatus == StatusDisabled || nextStatus == StatusLocked || nextStatus == StatusInvited {
			if err := s.revokeAdminSessionsTx(ctx, tx, target.ID, RevokedReasonStatusChanged); err != nil {
				return err
			}
		}
		if err := insertAdminAuditLogTx(ctx, tx, &actorID, AuditActorAdmin, "ADMIN_STATUS_CHANGED", "ADMIN_USER", &target.UUID, &reason, before, map[string]any{
			"status": nextStatus,
		}, meta); err != nil {
			return err
		}
		target.Status = nextStatus
		if value, ok := updates["must_change_password"].(bool); ok {
			target.MustChangePassword = value
		}
		return nil
	}); err != nil {
		return nil, err
	}
	response, err := s.adminUserResponse(ctx, s.db, target)
	if err != nil {
		return nil, err
	}
	return &response, nil
}

func (s *Service) adminUserResponse(ctx context.Context, db *gorm.DB, user AdminUser) (AdminUserResponse, error) {
	roles, err := s.rolesWithDB(ctx, db, user.ID)
	if err != nil {
		return AdminUserResponse{}, err
	}
	return toAdminUserResponse(user, roles, PermissionsForRoles(roles)), nil
}

func (s *Service) adminUserByUUIDTx(ctx context.Context, tx *gorm.DB, rawUUID string, lock bool) (AdminUser, error) {
	parsed, err := uuid.Parse(strings.TrimSpace(rawUUID))
	if err != nil {
		return AdminUser{}, validationError("adminUserUuid is invalid", map[string]any{"field": "adminUserUuid"})
	}
	query := tx.WithContext(ctx)
	if lock {
		query = query.Clauses(clause.Locking{Strength: "UPDATE"})
	}
	var user AdminUser
	if err := query.Where("uuid = ?", parsed).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return AdminUser{}, targetNotFoundError("Admin user not found")
		}
		return AdminUser{}, err
	}
	return user, nil
}

func (s *Service) ensureRoleMutationAllowedTx(ctx context.Context, tx *gorm.DB, actorID uint64, targetID uint64, role string) error {
	actorIsSuper, err := s.hasActiveRoleTx(ctx, tx, actorID, RoleSuperAdmin)
	if err != nil {
		return err
	}
	targetHasSuper, err := s.hasActiveRoleTx(ctx, tx, targetID, RoleSuperAdmin)
	if err != nil {
		return err
	}
	if targetHasSuper && !actorIsSuper {
		return forbiddenError(CodeCannotModifySuperAdmin, "Cannot modify SUPER_ADMIN", nil)
	}
	if role == RoleSuperAdmin && !actorIsSuper {
		return forbiddenError(CodeRoleAssignmentNotAllowed, "Only SUPER_ADMIN can modify SUPER_ADMIN role", nil)
	}
	return nil
}

func (s *Service) hasActiveRoleTx(ctx context.Context, tx *gorm.DB, adminID uint64, role string) (bool, error) {
	var exists bool
	err := tx.WithContext(ctx).Raw(`
		SELECT EXISTS (
		  SELECT 1
		  FROM admin_role_assignments
		  WHERE admin_user_id = ?
		    AND role = ?
		    AND status = ?
		)
	`, adminID, role, RoleAssignmentActive).Scan(&exists).Error
	return exists, err
}

func (s *Service) lockActiveSuperAdminRolesTx(ctx context.Context, tx *gorm.DB) (int, error) {
	var rows []struct {
		ID uint64
	}
	if err := tx.WithContext(ctx).Raw(`
		SELECT ara.id
		FROM admin_role_assignments ara
		JOIN admin_users au ON au.id = ara.admin_user_id
		WHERE ara.role = ?
		  AND ara.status = ?
		  AND au.status = ?
		FOR UPDATE OF ara
	`, RoleSuperAdmin, RoleAssignmentActive, StatusActive).Scan(&rows).Error; err != nil {
		return 0, err
	}
	return len(rows), nil
}

func (s *Service) invalidateAdminTokensTx(ctx context.Context, tx *gorm.DB, adminID uint64, revokedReason string) error {
	if err := tx.WithContext(ctx).Model(&AdminUser{}).Where("id = ?", adminID).Updates(map[string]any{
		"token_version": gorm.Expr("token_version + 1"),
		"updated_at":    time.Now().UTC(),
	}).Error; err != nil {
		return err
	}
	return s.revokeAdminSessionsTx(ctx, tx, adminID, revokedReason)
}

func (s *Service) revokeAdminSessionsTx(ctx context.Context, tx *gorm.DB, adminID uint64, revokedReason string) error {
	now := time.Now().UTC()
	return tx.WithContext(ctx).Model(&AdminSession{}).
		Where("admin_user_id = ? AND revoked_at IS NULL", adminID).
		Updates(map[string]any{
			"revoked_at":     now,
			"revoked_reason": revokedReason,
			"updated_at":     now,
		}).Error
}

func normalizeRoleList(rawRoles []string) ([]string, error) {
	seen := map[string]struct{}{}
	roles := make([]string, 0, len(rawRoles))
	for _, rawRole := range rawRoles {
		role := strings.ToUpper(strings.TrimSpace(rawRole))
		if role == "" {
			continue
		}
		if !IsValidRole(role) {
			return nil, validationError("role is invalid", map[string]any{"role": role})
		}
		if _, ok := seen[role]; ok {
			continue
		}
		seen[role] = struct{}{}
		roles = append(roles, role)
	}
	if len(roles) == 0 {
		return nil, validationError("roles are required", map[string]any{"field": "roles"})
	}
	return roles, nil
}

func requireReason(raw string) (string, error) {
	reason := strings.TrimSpace(raw)
	if reason == "" {
		return "", &ServiceError{Status: 400, Code: CodeReasonRequired, Message: "reason is required", Details: map[string]any{"field": "reason"}}
	}
	return reason, nil
}

func validateAdminPassword(password string, field string) error {
	if len(strings.TrimSpace(password)) < 12 {
		return validationError(field+" must be at least 12 characters", map[string]any{"field": field})
	}
	return nil
}

func isValidAdminStatus(status string) bool {
	return status == StatusInvited || status == StatusActive || status == StatusDisabled || status == StatusLocked
}

func generateTemporaryPassword() (string, error) {
	bytes := make([]byte, 24)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(bytes), nil
}

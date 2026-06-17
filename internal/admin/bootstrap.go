package admin

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type BootstrapSuperAdminInput struct {
	Email    string
	Name     string
	Password string
	Secret   string
}

func (s *Service) BootstrapSuperAdmin(ctx context.Context, input BootstrapSuperAdminInput, meta RequestMeta) (*AdminUserResponse, error) {
	if len(strings.TrimSpace(input.Secret)) < 32 {
		return nil, validationError("BOOTSTRAP_ADMIN_SECRET must be at least 32 characters", map[string]any{"field": "BOOTSTRAP_ADMIN_SECRET"})
	}
	email := normalizeEmail(input.Email)
	if email == "" {
		return nil, validationError("email is required", map[string]any{"field": "email"})
	}
	if err := validateAdminPassword(input.Password, "password"); err != nil {
		return nil, err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	reason := "bootstrap_cli"
	var user AdminUser
	if err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.WithContext(ctx).Exec(`LOCK TABLE admin_role_assignments IN EXCLUSIVE MODE`).Error; err != nil {
			return err
		}
		count, err := s.lockActiveSuperAdminRolesTx(ctx, tx)
		if err != nil {
			return err
		}
		if count > 0 {
			return actionNotAllowedError("An active SUPER_ADMIN already exists")
		}

		before := map[string]any{}
		findErr := tx.WithContext(ctx).Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("email = ?", email).
			First(&user).Error
		if findErr == nil {
			before = map[string]any{
				"email":              user.Email,
				"name":               user.Name,
				"status":             user.Status,
				"mustChangePassword": user.MustChangePassword,
			}
			updates := map[string]any{
				"password_hash":        string(hash),
				"status":               StatusActive,
				"must_change_password": false,
				"password_changed_at":  now,
				"token_version":        gorm.Expr("token_version + 1"),
				"updated_at":           now,
			}
			if strings.TrimSpace(input.Name) != "" {
				updates["name"] = strings.TrimSpace(input.Name)
			}
			if err := tx.WithContext(ctx).Model(&AdminUser{}).Where("id = ?", user.ID).Updates(updates).Error; err != nil {
				return err
			}
			if err := tx.WithContext(ctx).Where("id = ?", user.ID).First(&user).Error; err != nil {
				return err
			}
			if err := s.revokeAdminSessionsTx(ctx, tx, user.ID, RevokedReasonPasswordChanged); err != nil {
				return err
			}
		} else if errors.Is(findErr, gorm.ErrRecordNotFound) {
			var name *string
			if strings.TrimSpace(input.Name) != "" {
				value := strings.TrimSpace(input.Name)
				name = &value
			}
			user = AdminUser{
				UUID:               uuid.New(),
				Email:              email,
				Name:               name,
				PasswordHash:       string(hash),
				Status:             StatusActive,
				MustChangePassword: false,
				TokenVersion:       1,
				PasswordChangedAt:  &now,
				CreatedAt:          now,
				UpdatedAt:          now,
			}
			if err := tx.WithContext(ctx).Create(&user).Error; err != nil {
				return err
			}
		} else {
			return findErr
		}

		assignment := AdminRoleAssignment{
			UUID:        uuid.New(),
			AdminUserID: user.ID,
			Role:        RoleSuperAdmin,
			Status:      RoleAssignmentActive,
			AssignedAt:  now,
			Reason:      &reason,
			CreatedAt:   now,
			UpdatedAt:   now,
		}
		if err := tx.WithContext(ctx).Create(&assignment).Error; err != nil {
			return err
		}

		return insertAdminAuditLogTx(ctx, tx, nil, AuditActorSystem, "BOOTSTRAP_SUPER_ADMIN", "ADMIN_USER", &user.UUID, &reason, before, map[string]any{
			"email":              user.Email,
			"name":               user.Name,
			"status":             user.Status,
			"mustChangePassword": user.MustChangePassword,
			"roles":              []string{RoleSuperAdmin},
		}, meta)
	}); err != nil {
		return nil, err
	}

	response, err := s.adminUserResponse(ctx, s.db, user)
	if err != nil {
		return nil, err
	}
	return &response, nil
}

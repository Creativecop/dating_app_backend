package admin

import (
	"context"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/neoscoder/aura-backend/internal/response"
)

type Authenticator interface {
	Authenticate(ctx context.Context, rawToken string) (*AuthenticatedAdmin, error)
	HasPermission(ctx context.Context, adminID uint64, permission string) (bool, error)
}

func Auth(service Authenticator) gin.HandlerFunc {
	return func(c *gin.Context) {
		raw := c.GetHeader("Authorization")
		if !strings.HasPrefix(raw, "Bearer ") {
			response.Error(c, 401, "Unauthorized", CodeUnauthorized, nil)
			c.Abort()
			return
		}
		token := strings.TrimSpace(strings.TrimPrefix(raw, "Bearer "))
		if token == "" {
			response.Error(c, 401, "Unauthorized", CodeUnauthorized, nil)
			c.Abort()
			return
		}
		admin, err := service.Authenticate(c.Request.Context(), token)
		if err != nil {
			response.Error(c, 401, PublicErrorMessage(err), CodeUnauthorized, nil)
			c.Abort()
			return
		}
		c.Set(ContextAdminKey, *admin)
		c.Next()
	}
}

func RequirePermission(service Authenticator, permission string) gin.HandlerFunc {
	return RequirePermissionCode(service, permission, CodePermissionDenied)
}

func RequirePermissionCode(service Authenticator, permission string, code string) gin.HandlerFunc {
	return func(c *gin.Context) {
		adminUser, ok := CurrentAdmin(c)
		if !ok {
			response.Error(c, 401, "Unauthorized", CodeUnauthorized, nil)
			c.Abort()
			return
		}
		allowed, err := service.HasPermission(c.Request.Context(), adminUser.AdminUserID, permission)
		if err != nil {
			response.Internal(c)
			c.Abort()
			return
		}
		if !allowed {
			response.Error(c, 403, "Permission denied", code, nil)
			c.Abort()
			return
		}
		c.Next()
	}
}

func RequireAnyPermission(service Authenticator, permissions ...string) gin.HandlerFunc {
	return RequireAnyPermissionCode(service, CodePermissionDenied, permissions...)
}

func RequireAnyPermissionCode(service Authenticator, code string, permissions ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		adminUser, ok := CurrentAdmin(c)
		if !ok {
			response.Error(c, 401, "Unauthorized", CodeUnauthorized, nil)
			c.Abort()
			return
		}
		for _, permission := range permissions {
			allowed, err := service.HasPermission(c.Request.Context(), adminUser.AdminUserID, permission)
			if err != nil {
				response.Internal(c)
				c.Abort()
				return
			}
			if allowed {
				c.Next()
				return
			}
		}
		response.Error(c, 403, "Permission denied", code, nil)
		c.Abort()
	}
}

func RequirePasswordReady() gin.HandlerFunc {
	return func(c *gin.Context) {
		adminUser, ok := CurrentAdmin(c)
		if !ok {
			response.Error(c, 401, "Unauthorized", CodeUnauthorized, nil)
			c.Abort()
			return
		}
		if adminUser.Status == StatusInvited || adminUser.MustChangePassword {
			response.Error(c, 403, "Password change required", CodePasswordChangeRequired, nil)
			c.Abort()
			return
		}
		c.Next()
	}
}

func CurrentAdmin(c *gin.Context) (AuthenticatedAdmin, bool) {
	value, ok := c.Get(ContextAdminKey)
	if !ok {
		return AuthenticatedAdmin{}, false
	}
	adminUser, ok := value.(AuthenticatedAdmin)
	return adminUser, ok
}

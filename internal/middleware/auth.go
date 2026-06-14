package middleware

import (
	"context"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/neoscoder/aura-backend/internal/auth"
	"github.com/neoscoder/aura-backend/internal/response"
)

type Authenticator interface {
	Authenticate(ctx context.Context, rawToken string) (*auth.AuthenticatedUser, error)
}

func Auth(service Authenticator) gin.HandlerFunc {
	return func(c *gin.Context) {
		raw := c.GetHeader("Authorization")
		if !strings.HasPrefix(raw, "Bearer ") {
			response.Unauthorized(c, "Unauthorized.")
			c.Abort()
			return
		}

		token := strings.TrimSpace(strings.TrimPrefix(raw, "Bearer "))
		if token == "" {
			response.Unauthorized(c, "Unauthorized.")
			c.Abort()
			return
		}

		user, err := service.Authenticate(c.Request.Context(), token)
		if err != nil {
			response.Unauthorized(c, auth.PublicErrorMessage(err))
			c.Abort()
			return
		}

		c.Set(auth.ContextUserKey, *user)
		c.Next()
	}
}

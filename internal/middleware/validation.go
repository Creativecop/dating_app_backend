package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/neoscoder/aura-backend/internal/response"
)

func JSONBodyLimit(maxBytes int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		if maxBytes > 0 && c.Request != nil && c.Request.Body != nil && methodCanHaveBody(c.Request.Method) {
			if c.Request.ContentLength > maxBytes {
				response.Error(c, http.StatusRequestEntityTooLarge, "Request body too large", "REQUEST_BODY_TOO_LARGE", nil)
				c.Abort()
				return
			}
			c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBytes)
		}
		c.Next()
	}
}

func RequireIdempotencyKey() gin.HandlerFunc {
	return func(c *gin.Context) {
		if strings.TrimSpace(c.GetHeader("Idempotency-Key")) == "" {
			response.Error(c, http.StatusBadRequest, "Idempotency-Key header is required", "IDEMPOTENCY_KEY_REQUIRED", nil)
			c.Abort()
			return
		}
		c.Next()
	}
}

func ValidateUUIDParams(paramNames ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		for _, name := range paramNames {
			raw := strings.TrimSpace(c.Param(name))
			if raw == "" {
				continue
			}
			if _, err := uuid.Parse(raw); err != nil {
				response.Validation(c, map[string]any{"field": name, "message": name + " is invalid"})
				c.Abort()
				return
			}
		}
		c.Next()
	}
}

func methodCanHaveBody(method string) bool {
	switch strings.ToUpper(method) {
	case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
		return true
	default:
		return false
	}
}

package middleware

import (
	"log"

	"github.com/gin-gonic/gin"

	"github.com/neoscoder/aura-backend/internal/response"
)

func Recovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				log.Printf("panic request_id=%s error=%v", GetRequestID(c), err)
				response.Internal(c)
				c.Abort()
			}
		}()
		c.Next()
	}
}

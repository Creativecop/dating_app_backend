package middleware

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/gin-gonic/gin"

	"github.com/neoscoder/aura-backend/internal/response"
)

func Recovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				record := map[string]any{
					"event":      "panic_recovered",
					"request_id": GetRequestID(c),
					"method":     c.Request.Method,
					"path":       safeRequestPath(c),
					"route":      c.FullPath(),
					"error":      fmt.Sprint(err),
				}
				if encoded, jsonErr := json.Marshal(record); jsonErr == nil {
					log.Printf("%s", encoded)
				} else {
					log.Printf(`{"event":"panic_recovered","request_id":%q}`, GetRequestID(c))
				}
				response.Internal(c)
				c.Abort()
			}
		}()
		c.Next()
	}
}

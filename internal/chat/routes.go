package chat

import (
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/neoscoder/aura-backend/internal/config"
	appmiddleware "github.com/neoscoder/aura-backend/internal/middleware"
)

func RegisterRoutes(root *gin.Engine, v1 *gin.RouterGroup, authMiddleware gin.HandlerFunc, handler *Handler, limiter *appmiddleware.RateLimiter, rateCfg config.RateLimitConfig) {
	chats := v1.Group("/chats")
	chats.Use(authMiddleware)
	chats.GET("", handler.List)
	chats.GET("/:conversationUuid/messages", handler.Messages)
	chats.POST("/:conversationUuid/messages", handler.SendMessage)
	chats.POST("/:conversationUuid/read", handler.MarkRead)

	messages := v1.Group("/messages")
	messages.Use(authMiddleware)
	messages.DELETE("/:messageUuid", handler.DeleteMessage)

	root.GET("/ws", limiter.Limit(
		appmiddleware.RateLimitRule{Scope: "socket_connect_token_1m", Limit: rateCfg.SocketConnectUser1M, Window: time.Minute, Identifier: websocketTokenIdentifier(), FailClosed: false},
		appmiddleware.RateLimitRule{Scope: "socket_connect_ip_1m", Limit: rateCfg.SocketConnectIP1M, Window: time.Minute, Identifier: appmiddleware.IPIdentifier(), FailClosed: false},
	), handler.WebSocket)
}

func websocketTokenIdentifier() appmiddleware.RateLimitIdentifier {
	return func(c *gin.Context) string {
		raw := c.GetHeader("Authorization")
		if strings.HasPrefix(raw, "Bearer ") {
			return strings.TrimSpace(strings.TrimPrefix(raw, "Bearer "))
		}
		return strings.TrimSpace(c.Query("token"))
	}
}

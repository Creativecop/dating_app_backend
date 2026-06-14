package chat

import "github.com/gin-gonic/gin"

func RegisterRoutes(root *gin.Engine, v1 *gin.RouterGroup, authMiddleware gin.HandlerFunc, handler *Handler) {
	chats := v1.Group("/chats")
	chats.Use(authMiddleware)
	chats.GET("", handler.List)
	chats.GET("/:conversationUuid/messages", handler.Messages)
	chats.POST("/:conversationUuid/messages", handler.SendMessage)
	chats.POST("/:conversationUuid/read", handler.MarkRead)

	messages := v1.Group("/messages")
	messages.Use(authMiddleware)
	messages.DELETE("/:messageUuid", handler.DeleteMessage)

	root.GET("/ws", handler.WebSocket)
}

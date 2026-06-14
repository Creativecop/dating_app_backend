package notification

import "github.com/gin-gonic/gin"

func RegisterRoutes(v1 *gin.RouterGroup, authMiddleware gin.HandlerFunc, handler *Handler) {
	devices := v1.Group("/devices")
	devices.Use(authMiddleware)
	devices.POST("/fcm-token", handler.UpsertFCMToken)
	devices.DELETE("/fcm-token", handler.DeleteFCMToken)

	notifications := v1.Group("/notifications")
	notifications.Use(authMiddleware)
	notifications.GET("", handler.ListNotifications)
	notifications.PATCH("/read-all", handler.MarkAllRead)
	notifications.PATCH("/:notificationUuid/read", handler.MarkRead)

	settings := v1.Group("/notification-settings")
	settings.Use(authMiddleware)
	settings.GET("", handler.GetSettings)
	settings.PUT("", handler.UpdateSettings)
}

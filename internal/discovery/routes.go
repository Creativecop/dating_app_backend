package discovery

import (
	"github.com/gin-gonic/gin"

	"github.com/neoscoder/aura-backend/internal/middleware"
)

func RegisterRoutes(v1 *gin.RouterGroup, authMiddleware gin.HandlerFunc, handler *Handler) {
	routes := v1.Group("/discovery")
	routes.Use(authMiddleware)
	routes.GET("/preferences", handler.GetPreferences)
	routes.PUT("/preferences", handler.UpdatePreferences)
	routes.GET("/readiness", handler.Readiness)
	routes.GET("/feed", handler.Feed)
	routes.GET("/profiles/:userUuid", middleware.ValidateUUIDParams("userUuid"), handler.ProfileDetail)
	routes.POST("/impressions", handler.CreateImpressions)
	routes.POST("/actions", handler.CreateAction)
}

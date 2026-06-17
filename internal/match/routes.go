package match

import (
	"github.com/gin-gonic/gin"

	"github.com/neoscoder/aura-backend/internal/middleware"
)

func RegisterRoutes(v1 *gin.RouterGroup, authMiddleware gin.HandlerFunc, handler *Handler) {
	routes := v1.Group("/matches")
	routes.Use(authMiddleware)
	routes.GET("", handler.List)
	routes.GET("/:matchUuid", middleware.ValidateUUIDParams("matchUuid"), handler.Detail)
	routes.POST("/:matchUuid/seen", middleware.ValidateUUIDParams("matchUuid"), handler.Seen)
	routes.POST("/:matchUuid/unmatch", middleware.ValidateUUIDParams("matchUuid"), handler.Unmatch)
}

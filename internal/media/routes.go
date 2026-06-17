package media

import (
	"github.com/gin-gonic/gin"

	"github.com/neoscoder/aura-backend/internal/middleware"
)

func RegisterRoutes(v1 *gin.RouterGroup, authMiddleware gin.HandlerFunc, handler *Handler) {
	profileMedia := v1.Group("/profile/media")
	profileMedia.Use(authMiddleware)
	profileMedia.GET("/me", handler.ListMine)
	profileMedia.POST("/photos", handler.UploadPhoto)
	profileMedia.POST("/intro-video", handler.UploadIntroVideo)
	profileMedia.PATCH("/:mediaUuid/primary", middleware.ValidateUUIDParams("mediaUuid"), handler.SetPrimary)
	profileMedia.PATCH("/reorder", handler.Reorder)
	profileMedia.DELETE("/:mediaUuid", middleware.ValidateUUIDParams("mediaUuid"), handler.Delete)

	media := v1.Group("/media")
	media.Use(authMiddleware)
	media.GET("/:mediaUuid/:variant", middleware.ValidateUUIDParams("mediaUuid"), handler.Serve)
}

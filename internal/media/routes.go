package media

import "github.com/gin-gonic/gin"

func RegisterRoutes(v1 *gin.RouterGroup, authMiddleware gin.HandlerFunc, handler *Handler) {
	profileMedia := v1.Group("/profile/media")
	profileMedia.Use(authMiddleware)
	profileMedia.GET("/me", handler.ListMine)
	profileMedia.POST("/photos", handler.UploadPhoto)
	profileMedia.POST("/intro-video", handler.UploadIntroVideo)
	profileMedia.PATCH("/:mediaUuid/primary", handler.SetPrimary)
	profileMedia.PATCH("/reorder", handler.Reorder)
	profileMedia.DELETE("/:mediaUuid", handler.Delete)

	media := v1.Group("/media")
	media.Use(authMiddleware)
	media.GET("/:mediaUuid/:variant", handler.Serve)
}

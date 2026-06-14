package safety

import "github.com/gin-gonic/gin"

func RegisterRoutes(v1 *gin.RouterGroup, authMiddleware gin.HandlerFunc, handler *Handler) {
	reports := v1.Group("/reports")
	reports.Use(authMiddleware)
	reports.GET("/reasons", handler.ListReasons)
	reports.POST("", handler.CreateReport)
	reports.GET("/me", handler.MyReports)

	blocks := v1.Group("/blocks")
	blocks.Use(authMiddleware)
	blocks.POST("/:userUuid", handler.BlockUser)
	blocks.DELETE("/:userUuid", handler.UnblockUser)
	blocks.GET("", handler.ListBlocks)

	safety := v1.Group("/safety")
	safety.Use(authMiddleware)
	safety.POST("/block-and-report", handler.BlockAndReport)
	safety.GET("/settings", handler.GetSettings)
	safety.PUT("/settings", handler.UpdateSettings)
}

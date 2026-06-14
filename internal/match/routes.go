package match

import "github.com/gin-gonic/gin"

func RegisterRoutes(v1 *gin.RouterGroup, authMiddleware gin.HandlerFunc, handler *Handler) {
	routes := v1.Group("/matches")
	routes.Use(authMiddleware)
	routes.GET("", handler.List)
	routes.GET("/:matchUuid", handler.Detail)
	routes.POST("/:matchUuid/seen", handler.Seen)
	routes.POST("/:matchUuid/unmatch", handler.Unmatch)
}

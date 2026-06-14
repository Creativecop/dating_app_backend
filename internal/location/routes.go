package location

import "github.com/gin-gonic/gin"

func RegisterRoutes(v1 *gin.RouterGroup, authMiddleware gin.HandlerFunc, handler *Handler) {
	routes := v1.Group("/location")
	routes.Use(authMiddleware)
	routes.GET("/me", handler.GetMine)
	routes.PUT("", handler.Update)
}

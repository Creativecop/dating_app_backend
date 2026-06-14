package profile

import "github.com/gin-gonic/gin"

func RegisterRoutes(v1 *gin.RouterGroup, authMiddleware gin.HandlerFunc, handler *Handler) {
	routes := v1.Group("/profile")
	routes.Use(authMiddleware)

	catalog := routes.Group("/catalog")
	catalog.GET("/interests", handler.InterestCatalog)
	catalog.GET("/prompts", handler.PromptCatalog)
	catalog.GET("/lifestyle-questions", handler.LifestyleCatalog)

	routes.GET("/me", handler.GetMe)
	routes.PATCH("/me", handler.UpdateProfile)
	routes.PUT("/interests", handler.UpdateInterests)
	routes.PUT("/prompts", handler.UpdatePrompts)
	routes.PUT("/lifestyle", handler.UpdateLifestyle)
	routes.POST("/complete", handler.CompleteProfile)
}

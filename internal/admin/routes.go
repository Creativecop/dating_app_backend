package admin

import "github.com/gin-gonic/gin"

func RegisterRoutes(v1 *gin.RouterGroup, service *Service, handler *Handler) {
	adminGroup := v1.Group("/admin")

	authRoutes := adminGroup.Group("/auth")
	authRoutes.POST("/login", handler.Login)
	authRoutes.POST("/refresh-token", handler.RefreshToken)

	protected := authRoutes.Group("")
	protected.Use(Auth(service))
	protected.POST("/logout", handler.Logout)
	protected.GET("/me", handler.Me)
	protected.POST("/change-password", handler.ChangePassword)

	auditRoutes := adminGroup.Group("/audit-logs")
	auditRoutes.Use(Auth(service), RequirePermission(service, PermissionAuditRead))
	auditRoutes.GET("", handler.ListAuditLogs)
	auditRoutes.GET("/:auditLogUuid", handler.AuditLogDetail)
}

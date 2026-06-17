package admin

import "github.com/gin-gonic/gin"

func RegisterRoutes(v1 *gin.RouterGroup, service *Service, handler *Handler) {
	adminGroup := v1.Group("/admin")

	authRoutes := adminGroup.Group("/auth")
	authRoutes.POST("/login", handler.Login)
	authRoutes.POST("/refresh-token", handler.RefreshToken)

	passwordRoutes := authRoutes.Group("")
	passwordRoutes.Use(Auth(service))
	passwordRoutes.POST("/change-password", handler.ChangePassword)

	protected := authRoutes.Group("")
	protected.Use(Auth(service), RequirePasswordReady())
	protected.POST("/logout", handler.Logout)
	protected.GET("/me", handler.Me)

	capabilityRoutes := adminGroup.Group("/capabilities")
	capabilityRoutes.Use(Auth(service), RequirePasswordReady())
	capabilityRoutes.GET("", handler.Capabilities)

	auditRoutes := adminGroup.Group("/audit-logs")
	auditRoutes.Use(Auth(service), RequirePasswordReady(), RequirePermission(service, PermissionAuditRead))
	auditRoutes.GET("", handler.ListAuditLogs)
	auditRoutes.GET("/:auditLogUuid", handler.AuditLogDetail)

	roleRoutes := adminGroup.Group("/roles")
	roleRoutes.Use(Auth(service), RequirePasswordReady(), RequirePermission(service, PermissionRolesManage))
	roleRoutes.GET("", handler.ListRoles)

	adminUserRoutes := adminGroup.Group("/admin-users")
	adminUserRoutes.Use(Auth(service), RequirePasswordReady())
	adminUserRoutes.GET("", RequirePermission(service, PermissionAdminUsersRead), handler.ListAdminUsers)
	adminUserRoutes.POST("", RequirePermission(service, PermissionAdminUsersCreate), handler.CreateAdminUser)
	adminUserRoutes.GET("/:adminUserUuid", RequirePermission(service, PermissionAdminUsersRead), handler.AdminUserDetail)
	adminUserRoutes.POST("/:adminUserUuid/roles", RequirePermission(service, PermissionRolesManage), handler.AssignAdminRole)
	adminUserRoutes.DELETE("/:adminUserUuid/roles/:role", RequirePermission(service, PermissionRolesManage), handler.RemoveAdminRole)
	adminUserRoutes.PATCH("/:adminUserUuid/status", RequirePermission(service, PermissionAdminUsersManage), handler.UpdateAdminStatus)

	userRoutes := adminGroup.Group("/users")
	userRoutes.Use(Auth(service), RequirePasswordReady())
	userRoutes.GET("", RequirePermission(service, PermissionUsersRead), handler.ListUsers)
	userRoutes.GET("/:userId", RequirePermission(service, PermissionUsersRead), handler.UserDetail)
	userRoutes.GET("/:userId/restrictions", RequirePermission(service, PermissionUsersRead), handler.ListUserRestrictions)
	userRoutes.POST("/:userId/restrictions", RequirePermission(service, PermissionUsersRestrict), handler.CreateUserRestriction)
	userRoutes.DELETE("/:userId/restrictions/:restrictionId", RequirePermission(service, PermissionUsersRestrict), handler.RevokeUserRestriction)
}

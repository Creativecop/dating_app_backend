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

	dashboardRoutes := adminGroup.Group("/dashboard")
	dashboardRoutes.Use(Auth(service), RequirePasswordReady(), RequireAnyPermissionCode(
		service,
		CodeAdminAnalyticsAccessDenied,
		PermissionAnalyticsRead,
		PermissionAnalyticsReportsRead,
		PermissionAnalyticsRestrictionsRead,
		PermissionAnalyticsTrustSafetyRead,
		PermissionAnalyticsAdminActivityRead,
		PermissionAnalyticsSubscriptionPaymentsRead,
	))
	dashboardRoutes.GET("/summary", handler.DashboardSummary)

	analyticsRoutes := adminGroup.Group("/analytics")
	analyticsRoutes.Use(Auth(service), RequirePasswordReady())
	analyticsRoutes.GET("/users", RequirePermissionCode(service, PermissionAnalyticsRead, CodeAdminAnalyticsAccessDenied), handler.UserAnalytics)
	analyticsRoutes.GET("/reports", RequireAnyPermissionCode(service, CodeAdminAnalyticsAccessDenied, PermissionAnalyticsRead, PermissionAnalyticsReportsRead, PermissionAnalyticsTrustSafetyRead), handler.ReportAnalytics)
	analyticsRoutes.GET("/restrictions", RequireAnyPermissionCode(service, CodeAdminAnalyticsAccessDenied, PermissionAnalyticsRead, PermissionAnalyticsRestrictionsRead, PermissionAnalyticsTrustSafetyRead), handler.RestrictionAnalytics)
	analyticsRoutes.GET("/trust-safety", RequireAnyPermissionCode(service, CodeAdminAnalyticsAccessDenied, PermissionAnalyticsTrustSafetyRead, PermissionAnalyticsRead), handler.TrustSafetyAnalytics)
	analyticsRoutes.GET("/admin-activity", RequireAnyPermissionCode(service, CodeAdminAnalyticsAccessDenied, PermissionAnalyticsRead, PermissionAnalyticsAdminActivityRead), handler.AdminActivityAnalytics)
	analyticsRoutes.GET("/subscription-payments", RequirePermissionCode(service, PermissionAnalyticsSubscriptionPaymentsRead, CodeAdminAnalyticsAccessDenied), handler.SubscriptionPaymentAnalytics)

	auditRoutes := adminGroup.Group("/audit-logs")
	auditRoutes.Use(Auth(service), RequirePasswordReady(), RequirePermissionCode(service, PermissionAuditRead, CodeAdminAuditAccessDenied))
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

package admin

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/neoscoder/aura-backend/internal/config"
	appmiddleware "github.com/neoscoder/aura-backend/internal/middleware"
)

func RegisterRoutes(v1 *gin.RouterGroup, service *Service, handler *Handler, limiter *appmiddleware.RateLimiter, rateCfg config.RateLimitConfig, bodyLimit gin.HandlerFunc) {
	adminGroup := v1.Group("/admin")
	adminGroup.Use(bodyLimit)

	authRoutes := adminGroup.Group("/auth")
	authRoutes.POST("/login", limiter.Limit(
		appmiddleware.RateLimitRule{Scope: "admin_login_email_15m", Limit: rateCfg.AdminLoginEmail15M, Window: 15 * time.Minute, Identifier: appmiddleware.BodyFieldIdentifier("email"), FailClosed: true},
		appmiddleware.RateLimitRule{Scope: "admin_login_ip_15m", Limit: rateCfg.AdminLoginIP15M, Window: 15 * time.Minute, Identifier: appmiddleware.IPIdentifier(), FailClosed: true},
		appmiddleware.RateLimitRule{Scope: "admin_login_ip_1h", Limit: rateCfg.AdminLoginIP1H, Window: time.Hour, Identifier: appmiddleware.IPIdentifier(), FailClosed: true},
	), handler.Login)
	authRoutes.POST("/refresh-token", limiter.Limit(
		appmiddleware.RateLimitRule{Scope: "admin_refresh_subject_1m", Limit: rateCfg.RefreshSubject1M, Window: time.Minute, Identifier: appmiddleware.BodyFieldIdentifier("refreshToken"), FailClosed: false},
		appmiddleware.RateLimitRule{Scope: "admin_refresh_ip_1m", Limit: rateCfg.RefreshIP1M, Window: time.Minute, Identifier: appmiddleware.IPIdentifier(), FailClosed: false},
	), handler.RefreshToken)

	passwordRoutes := authRoutes.Group("")
	passwordRoutes.Use(Auth(service))
	passwordRoutes.POST("/change-password", handler.ChangePassword)

	protected := authRoutes.Group("")
	protected.Use(Auth(service), RequirePasswordReady())
	protected.POST("/logout", handler.Logout)
	protected.GET("/me", handler.Me)

	capabilityRoutes := adminGroup.Group("/capabilities")
	capabilityRoutes.Use(Auth(service), RequirePasswordReady(), adminReadRateLimit(limiter, rateCfg))
	capabilityRoutes.GET("", handler.Capabilities)

	dashboardRoutes := adminGroup.Group("/dashboard")
	dashboardRoutes.Use(Auth(service), RequirePasswordReady(), adminReadRateLimit(limiter, rateCfg), RequireAnyPermissionCode(
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
	analyticsRoutes.Use(Auth(service), RequirePasswordReady(), adminReadRateLimit(limiter, rateCfg))
	analyticsRoutes.GET("/users", RequirePermissionCode(service, PermissionAnalyticsRead, CodeAdminAnalyticsAccessDenied), handler.UserAnalytics)
	analyticsRoutes.GET("/reports", RequireAnyPermissionCode(service, CodeAdminAnalyticsAccessDenied, PermissionAnalyticsRead, PermissionAnalyticsReportsRead, PermissionAnalyticsTrustSafetyRead), handler.ReportAnalytics)
	analyticsRoutes.GET("/restrictions", RequireAnyPermissionCode(service, CodeAdminAnalyticsAccessDenied, PermissionAnalyticsRead, PermissionAnalyticsRestrictionsRead, PermissionAnalyticsTrustSafetyRead), handler.RestrictionAnalytics)
	analyticsRoutes.GET("/trust-safety", RequireAnyPermissionCode(service, CodeAdminAnalyticsAccessDenied, PermissionAnalyticsTrustSafetyRead, PermissionAnalyticsRead), handler.TrustSafetyAnalytics)
	analyticsRoutes.GET("/admin-activity", RequireAnyPermissionCode(service, CodeAdminAnalyticsAccessDenied, PermissionAnalyticsRead, PermissionAnalyticsAdminActivityRead), handler.AdminActivityAnalytics)
	analyticsRoutes.GET("/subscription-payments", RequirePermissionCode(service, PermissionAnalyticsSubscriptionPaymentsRead, CodeAdminAnalyticsAccessDenied), handler.SubscriptionPaymentAnalytics)

	auditRoutes := adminGroup.Group("/audit-logs")
	auditRoutes.Use(Auth(service), RequirePasswordReady(), adminReadRateLimit(limiter, rateCfg), RequirePermissionCode(service, PermissionAuditRead, CodeAdminAuditAccessDenied))
	auditRoutes.GET("", handler.ListAuditLogs)
	auditRoutes.GET("/:auditLogUuid", appmiddleware.ValidateUUIDParams("auditLogUuid"), handler.AuditLogDetail)

	roleRoutes := adminGroup.Group("/roles")
	roleRoutes.Use(Auth(service), RequirePasswordReady(), RequirePermission(service, PermissionRolesManage))
	roleRoutes.GET("", handler.ListRoles)

	adminUserRoutes := adminGroup.Group("/admin-users")
	adminUserRoutes.Use(Auth(service), RequirePasswordReady())
	adminUserRoutes.GET("", RequirePermission(service, PermissionAdminUsersRead), handler.ListAdminUsers)
	adminUserRoutes.POST("", adminIdentityMutationRateLimit(limiter, rateCfg), RequirePermission(service, PermissionAdminUsersCreate), handler.CreateAdminUser)
	adminUserRoutes.GET("/:adminUserUuid", appmiddleware.ValidateUUIDParams("adminUserUuid"), RequirePermission(service, PermissionAdminUsersRead), handler.AdminUserDetail)
	adminUserRoutes.POST("/:adminUserUuid/roles", appmiddleware.ValidateUUIDParams("adminUserUuid"), adminIdentityMutationRateLimit(limiter, rateCfg), RequirePermission(service, PermissionRolesManage), handler.AssignAdminRole)
	adminUserRoutes.DELETE("/:adminUserUuid/roles/:role", appmiddleware.ValidateUUIDParams("adminUserUuid"), adminIdentityMutationRateLimit(limiter, rateCfg), RequirePermission(service, PermissionRolesManage), handler.RemoveAdminRole)
	adminUserRoutes.PATCH("/:adminUserUuid/status", appmiddleware.ValidateUUIDParams("adminUserUuid"), adminIdentityMutationRateLimit(limiter, rateCfg), RequirePermission(service, PermissionAdminUsersManage), handler.UpdateAdminStatus)

	userRoutes := adminGroup.Group("/users")
	userRoutes.Use(Auth(service), RequirePasswordReady())
	userRoutes.GET("", RequirePermission(service, PermissionUsersRead), handler.ListUsers)
	userRoutes.GET("/:userId", appmiddleware.ValidateUUIDParams("userId"), RequirePermission(service, PermissionUsersRead), handler.UserDetail)
	userRoutes.GET("/:userId/restrictions", appmiddleware.ValidateUUIDParams("userId"), RequirePermission(service, PermissionUsersRead), handler.ListUserRestrictions)
	userRoutes.POST("/:userId/restrictions", appmiddleware.ValidateUUIDParams("userId"), userRestrictionMutationRateLimit(limiter, rateCfg), RequirePermission(service, PermissionUsersRestrict), handler.CreateUserRestriction)
	userRoutes.DELETE("/:userId/restrictions/:restrictionId", appmiddleware.ValidateUUIDParams("userId", "restrictionId"), userRestrictionMutationRateLimit(limiter, rateCfg), RequirePermission(service, PermissionUsersRestrict), handler.RevokeUserRestriction)
}

func adminReadRateLimit(limiter *appmiddleware.RateLimiter, cfg config.RateLimitConfig) gin.HandlerFunc {
	return limiter.Limit(
		appmiddleware.RateLimitRule{Scope: "admin_read_admin_1m", Limit: cfg.AdminReadAdmin1M, Window: time.Minute, Identifier: adminIDIdentifier(), FailClosed: false},
		appmiddleware.RateLimitRule{Scope: "admin_read_ip_1m", Limit: cfg.AdminReadIP1M, Window: time.Minute, Identifier: appmiddleware.IPIdentifier(), FailClosed: false},
	)
}

func adminIdentityMutationRateLimit(limiter *appmiddleware.RateLimiter, cfg config.RateLimitConfig) gin.HandlerFunc {
	return limiter.Limit(
		appmiddleware.RateLimitRule{Scope: "admin_identity_mutation_10m", Limit: cfg.AdminIdentityMutation10M, Window: 10 * time.Minute, Identifier: adminIDIdentifier(), FailClosed: false},
		appmiddleware.RateLimitRule{Scope: "admin_identity_mutation_1h", Limit: cfg.AdminIdentityMutation1H, Window: time.Hour, Identifier: adminIDIdentifier(), FailClosed: false},
	)
}

func userRestrictionMutationRateLimit(limiter *appmiddleware.RateLimiter, cfg config.RateLimitConfig) gin.HandlerFunc {
	return limiter.Limit(
		appmiddleware.RateLimitRule{Scope: "user_restriction_mutation_1m", Limit: cfg.AdminRestrictionMutation1M, Window: time.Minute, Identifier: adminIDIdentifier(), FailClosed: false},
		appmiddleware.RateLimitRule{Scope: "user_restriction_mutation_1h", Limit: cfg.AdminRestrictionMutation1H, Window: time.Hour, Identifier: adminIDIdentifier(), FailClosed: false},
	)
}

func adminIDIdentifier() appmiddleware.RateLimitIdentifier {
	return func(c *gin.Context) string {
		adminUser, ok := CurrentAdmin(c)
		if !ok {
			return ""
		}
		return strconv.FormatUint(adminUser.AdminUserID, 10)
	}
}

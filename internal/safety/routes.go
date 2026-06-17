package safety

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	adminpkg "github.com/neoscoder/aura-backend/internal/admin"
	"github.com/neoscoder/aura-backend/internal/auth"
	"github.com/neoscoder/aura-backend/internal/config"
	appmiddleware "github.com/neoscoder/aura-backend/internal/middleware"
)

func RegisterRoutes(v1 *gin.RouterGroup, authMiddleware gin.HandlerFunc, handler *Handler, limiter *appmiddleware.RateLimiter, rateCfg config.RateLimitConfig, bodyLimit gin.HandlerFunc) {
	reports := v1.Group("/reports")
	reports.Use(bodyLimit, authMiddleware)
	reports.GET("/reasons", handler.ListReasons)
	reports.POST("", reportCreateRateLimit(limiter, rateCfg), handler.CreateReport)
	reports.GET("/me", handler.MyReports)

	blocks := v1.Group("/blocks")
	blocks.Use(authMiddleware)
	blocks.POST("/:userUuid", appmiddleware.ValidateUUIDParams("userUuid"), handler.BlockUser)
	blocks.DELETE("/:userUuid", appmiddleware.ValidateUUIDParams("userUuid"), handler.UnblockUser)
	blocks.GET("", handler.ListBlocks)

	safety := v1.Group("/safety")
	safety.Use(bodyLimit, authMiddleware)
	safety.POST("/block-and-report", reportCreateRateLimit(limiter, rateCfg), handler.BlockAndReport)
	safety.GET("/settings", handler.GetSettings)
	safety.PUT("/settings", handler.UpdateSettings)
}

func RegisterAdminRoutes(v1 *gin.RouterGroup, adminService *adminpkg.Service, handler *Handler, limiter *appmiddleware.RateLimiter, rateCfg config.RateLimitConfig, bodyLimit gin.HandlerFunc) {
	reports := v1.Group("/admin/reports")
	reports.Use(bodyLimit, adminpkg.Auth(adminService), adminpkg.RequirePasswordReady(), adminpkg.RequirePermission(adminService, adminpkg.PermissionReportsReview))
	reports.GET("", handler.AdminListReports)
	reports.GET("/:reportId", appmiddleware.ValidateUUIDParams("reportId"), handler.AdminReportDetail)
	reports.POST("/:reportId/review", appmiddleware.ValidateUUIDParams("reportId"), reportReviewRateLimit(limiter, rateCfg), handler.AdminReviewReport)
}

func reportCreateRateLimit(limiter *appmiddleware.RateLimiter, cfg config.RateLimitConfig) gin.HandlerFunc {
	return limiter.Limit(
		appmiddleware.RateLimitRule{Scope: "report_create_user_1m", Limit: cfg.ReportCreateUser1M, Window: time.Minute, Identifier: userIDIdentifier(), FailClosed: false},
		appmiddleware.RateLimitRule{Scope: "report_create_user_1h", Limit: cfg.ReportCreateUser1H, Window: time.Hour, Identifier: userIDIdentifier(), FailClosed: false},
		appmiddleware.RateLimitRule{Scope: "report_create_ip_1h", Limit: cfg.ReportCreateIP1H, Window: time.Hour, Identifier: appmiddleware.IPIdentifier(), FailClosed: false},
	)
}

func reportReviewRateLimit(limiter *appmiddleware.RateLimiter, cfg config.RateLimitConfig) gin.HandlerFunc {
	return limiter.Limit(
		appmiddleware.RateLimitRule{Scope: "report_review_admin_1m", Limit: cfg.AdminReview1M, Window: time.Minute, Identifier: adminIDIdentifier(), FailClosed: false},
		appmiddleware.RateLimitRule{Scope: "report_review_admin_1h", Limit: cfg.AdminReview1H, Window: time.Hour, Identifier: adminIDIdentifier(), FailClosed: false},
	)
}

func userIDIdentifier() appmiddleware.RateLimitIdentifier {
	return func(c *gin.Context) string {
		user, ok := auth.CurrentUser(c)
		if !ok {
			return ""
		}
		return strconv.FormatUint(user.UserID, 10)
	}
}

func adminIDIdentifier() appmiddleware.RateLimitIdentifier {
	return func(c *gin.Context) string {
		adminUser, ok := adminpkg.CurrentAdmin(c)
		if !ok {
			return ""
		}
		return strconv.FormatUint(adminUser.AdminUserID, 10)
	}
}

package subscription

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/neoscoder/aura-backend/internal/admin"
	"github.com/neoscoder/aura-backend/internal/auth"
	"github.com/neoscoder/aura-backend/internal/config"
	appmiddleware "github.com/neoscoder/aura-backend/internal/middleware"
)

func RegisterRoutes(v1 *gin.RouterGroup, authMiddleware gin.HandlerFunc, handler *Handler, limiter *appmiddleware.RateLimiter, rateCfg config.RateLimitConfig, bodyLimit gin.HandlerFunc) {
	routes := v1.Group("/subscriptions")
	routes.Use(bodyLimit, authMiddleware)
	routes.GET("/plans", handler.ListPlans)
	routes.GET("/me", handler.CurrentSubscription)
	routes.GET("/entitlements", handler.Entitlements)
	routes.GET("/usage", handler.Usage)
	routes.GET("/premium-status", handler.PremiumStatus)
	routes.POST("/manual-payment-requests", appmiddleware.RequireIdempotencyKey(), subscriptionSubmitRateLimit(limiter, rateCfg), handler.CreateManualPaymentRequest)
	routes.GET("/manual-payment-requests", handler.ListManualPaymentRequests)
}

func RegisterAdminRoutes(v1 *gin.RouterGroup, adminService *admin.Service, handler *Handler, limiter *appmiddleware.RateLimiter, rateCfg config.RateLimitConfig, bodyLimit gin.HandlerFunc) {
	routes := v1.Group("/admin/subscriptions/payment-requests")
	routes.Use(bodyLimit, admin.Auth(adminService), admin.RequirePasswordReady())

	routes.GET("",
		admin.RequirePermission(adminService, admin.PermissionPaymentRequestsRead),
		handler.AdminListPaymentRequests,
	)
	routes.GET("/:paymentRequestUuid",
		appmiddleware.ValidateUUIDParams("paymentRequestUuid"),
		admin.RequirePermission(adminService, admin.PermissionPaymentRequestsRead),
		handler.AdminPaymentRequestDetail,
	)
	routes.POST("/:paymentRequestUuid/approve",
		appmiddleware.ValidateUUIDParams("paymentRequestUuid"),
		appmiddleware.RequireIdempotencyKey(),
		subscriptionReviewRateLimit(limiter, rateCfg),
		admin.RequirePermission(adminService, admin.PermissionPaymentRequestsApprove),
		handler.AdminApprovePaymentRequest,
	)
	routes.POST("/:paymentRequestUuid/reject",
		appmiddleware.ValidateUUIDParams("paymentRequestUuid"),
		appmiddleware.RequireIdempotencyKey(),
		subscriptionReviewRateLimit(limiter, rateCfg),
		admin.RequirePermission(adminService, admin.PermissionPaymentRequestsReject),
		handler.AdminRejectPaymentRequest,
	)
}

func subscriptionSubmitRateLimit(limiter *appmiddleware.RateLimiter, cfg config.RateLimitConfig) gin.HandlerFunc {
	return limiter.Limit(
		appmiddleware.RateLimitRule{Scope: "subscription_submit_user_10m", Limit: cfg.SubscriptionSubmitUser10M, Window: 10 * time.Minute, Identifier: userIDIdentifier(), FailClosed: false},
		appmiddleware.RateLimitRule{Scope: "subscription_submit_user_1h", Limit: cfg.SubscriptionSubmitUser1H, Window: time.Hour, Identifier: userIDIdentifier(), FailClosed: false},
		appmiddleware.RateLimitRule{Scope: "subscription_submit_ip_1h", Limit: cfg.SubscriptionSubmitIP1H, Window: time.Hour, Identifier: appmiddleware.IPIdentifier(), FailClosed: false},
	)
}

func subscriptionReviewRateLimit(limiter *appmiddleware.RateLimiter, cfg config.RateLimitConfig) gin.HandlerFunc {
	return limiter.Limit(
		appmiddleware.RateLimitRule{Scope: "subscription_review_admin_1m", Limit: cfg.SubscriptionReviewAdmin1M, Window: time.Minute, Identifier: adminIDIdentifier(), FailClosed: false},
		appmiddleware.RateLimitRule{Scope: "subscription_review_admin_1h", Limit: cfg.SubscriptionReviewAdmin1H, Window: time.Hour, Identifier: adminIDIdentifier(), FailClosed: false},
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
		adminUser, ok := admin.CurrentAdmin(c)
		if !ok {
			return ""
		}
		return strconv.FormatUint(adminUser.AdminUserID, 10)
	}
}

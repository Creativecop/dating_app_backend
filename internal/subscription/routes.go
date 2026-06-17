package subscription

import (
	"github.com/gin-gonic/gin"

	"github.com/neoscoder/aura-backend/internal/admin"
)

func RegisterRoutes(v1 *gin.RouterGroup, authMiddleware gin.HandlerFunc, handler *Handler) {
	routes := v1.Group("/subscriptions")
	routes.Use(authMiddleware)
	routes.GET("/plans", handler.ListPlans)
	routes.GET("/me", handler.CurrentSubscription)
	routes.GET("/entitlements", handler.Entitlements)
	routes.GET("/usage", handler.Usage)
	routes.GET("/premium-status", handler.PremiumStatus)
	routes.POST("/manual-payment-requests", handler.CreateManualPaymentRequest)
	routes.GET("/manual-payment-requests", handler.ListManualPaymentRequests)
}

func RegisterAdminRoutes(v1 *gin.RouterGroup, adminService *admin.Service, handler *Handler) {
	routes := v1.Group("/admin/subscriptions/payment-requests")
	routes.Use(admin.Auth(adminService), admin.RequirePasswordReady())

	routes.GET("",
		admin.RequirePermission(adminService, admin.PermissionPaymentRequestsRead),
		handler.AdminListPaymentRequests,
	)
	routes.GET("/:paymentRequestUuid",
		admin.RequirePermission(adminService, admin.PermissionPaymentRequestsRead),
		handler.AdminPaymentRequestDetail,
	)
	routes.POST("/:paymentRequestUuid/approve",
		admin.RequirePermission(adminService, admin.PermissionPaymentRequestsApprove),
		handler.AdminApprovePaymentRequest,
	)
	routes.POST("/:paymentRequestUuid/reject",
		admin.RequirePermission(adminService, admin.PermissionPaymentRequestsReject),
		handler.AdminRejectPaymentRequest,
	)
}

package subscription

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/neoscoder/aura-backend/internal/admin"
	"github.com/neoscoder/aura-backend/internal/auth"
	"github.com/neoscoder/aura-backend/internal/middleware"
	"github.com/neoscoder/aura-backend/internal/response"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) ListPlans(c *gin.Context) {
	result, err := h.service.ListPlans(c.Request.Context())
	if err != nil {
		h.writeError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Subscription plans fetched successfully", result)
}

func (h *Handler) CurrentSubscription(c *gin.Context) {
	user, ok := currentUser(c)
	if !ok {
		return
	}
	result, err := h.service.CurrentSubscription(c.Request.Context(), user.UserID)
	if err != nil {
		h.writeError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Subscription fetched successfully", result)
}

func (h *Handler) Entitlements(c *gin.Context) {
	user, ok := currentUser(c)
	if !ok {
		return
	}
	result, err := h.service.GetEntitlements(c.Request.Context(), user.UserID)
	if err != nil {
		h.writeError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Entitlements fetched successfully", result)
}

func (h *Handler) Usage(c *gin.Context) {
	user, ok := currentUser(c)
	if !ok {
		return
	}
	result, err := h.service.Usage(c.Request.Context(), user.UserID)
	if err != nil {
		h.writeError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Usage fetched successfully", result)
}

func (h *Handler) PremiumStatus(c *gin.Context) {
	user, ok := currentUser(c)
	if !ok {
		return
	}
	result, err := h.service.PremiumStatus(c.Request.Context(), user.UserID)
	if err != nil {
		h.writeError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Premium status fetched successfully", result)
}

func (h *Handler) CreateManualPaymentRequest(c *gin.Context) {
	user, ok := currentUser(c)
	if !ok {
		return
	}
	var req CreateManualPaymentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Validation(c, err.Error())
		return
	}
	result, err := h.service.CreateManualPaymentRequest(c.Request.Context(), user.UserID, req)
	if err != nil {
		h.writeError(c, err)
		return
	}
	response.Success(c, http.StatusCreated, "Payment request submitted successfully", result)
}

func (h *Handler) ListManualPaymentRequests(c *gin.Context) {
	user, ok := currentUser(c)
	if !ok {
		return
	}
	result, err := h.service.ListManualPaymentRequests(c.Request.Context(), user.UserID)
	if err != nil {
		h.writeError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Payment requests fetched successfully", result)
}

func (h *Handler) AdminListPaymentRequests(c *gin.Context) {
	result, err := h.service.AdminListPaymentRequests(c.Request.Context(), c.Query("status"))
	if err != nil {
		h.writeError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Payment requests fetched successfully", result)
}

func (h *Handler) AdminPaymentRequestDetail(c *gin.Context) {
	result, err := h.service.AdminPaymentRequestDetail(c.Request.Context(), c.Param("paymentRequestUuid"))
	if err != nil {
		h.writeError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Payment request fetched successfully", result)
}

func (h *Handler) AdminApprovePaymentRequest(c *gin.Context) {
	adminUser, ok := currentAdmin(c)
	if !ok {
		return
	}
	var req ReviewPaymentRequest
	_ = c.ShouldBindJSON(&req)
	result, err := h.service.ApprovePaymentRequest(c.Request.Context(), adminUser.AdminUserID, c.Param("paymentRequestUuid"), req, adminMeta(c))
	if err != nil {
		h.writeError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Payment request approved successfully", result)
}

func (h *Handler) AdminRejectPaymentRequest(c *gin.Context) {
	adminUser, ok := currentAdmin(c)
	if !ok {
		return
	}
	var req RejectPaymentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Validation(c, err.Error())
		return
	}
	result, err := h.service.RejectPaymentRequest(c.Request.Context(), adminUser.AdminUserID, c.Param("paymentRequestUuid"), req, adminMeta(c))
	if err != nil {
		h.writeError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Payment request rejected successfully", result)
}

func (h *Handler) writeError(c *gin.Context, err error) {
	if serviceErr, ok := AsServiceError(err); ok {
		response.Error(c, serviceErr.Status, serviceErr.Message, serviceErr.Code, serviceErr.Details)
		return
	}
	response.Internal(c)
}

func currentUser(c *gin.Context) (auth.AuthenticatedUser, bool) {
	user, ok := auth.CurrentUser(c)
	if !ok {
		response.Unauthorized(c, "Unauthorized.")
		return auth.AuthenticatedUser{}, false
	}
	return user, true
}

func currentAdmin(c *gin.Context) (admin.AuthenticatedAdmin, bool) {
	adminUser, ok := admin.CurrentAdmin(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, "Unauthorized", admin.CodeUnauthorized, nil)
		return admin.AuthenticatedAdmin{}, false
	}
	return adminUser, true
}

func adminMeta(c *gin.Context) AdminMeta {
	return AdminMeta{IPAddress: c.ClientIP(), UserAgent: c.Request.UserAgent(), RequestID: middleware.GetRequestID(c)}
}

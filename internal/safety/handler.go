package safety

import (
	"net/http"

	"github.com/gin-gonic/gin"

	adminpkg "github.com/neoscoder/aura-backend/internal/admin"
	"github.com/neoscoder/aura-backend/internal/auth"
	"github.com/neoscoder/aura-backend/internal/response"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) ListReasons(c *gin.Context) {
	result, err := h.service.ListReasons(c.Request.Context())
	if err != nil {
		h.writeError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Report reasons fetched successfully", result)
}

func (h *Handler) CreateReport(c *gin.Context) {
	user, ok := currentUser(c)
	if !ok {
		return
	}
	var req CreateReportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Validation(c, err.Error())
		return
	}
	result, err := h.service.CreateReport(c.Request.Context(), user.UserID, req)
	if err != nil {
		h.writeError(c, err)
		return
	}
	response.Success(c, http.StatusCreated, "Report submitted successfully", result)
}

func (h *Handler) MyReports(c *gin.Context) {
	user, ok := currentUser(c)
	if !ok {
		return
	}
	result, err := h.service.MyReports(c.Request.Context(), user.UserID)
	if err != nil {
		h.writeError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Reports fetched successfully", result)
}

func (h *Handler) BlockAndReport(c *gin.Context) {
	user, ok := currentUser(c)
	if !ok {
		return
	}
	var req CreateReportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Validation(c, err.Error())
		return
	}
	req.BlockUser = true
	result, err := h.service.CreateReport(c.Request.Context(), user.UserID, req)
	if err != nil {
		h.writeError(c, err)
		return
	}
	response.Success(c, http.StatusCreated, "User blocked and report submitted successfully", result)
}

func (h *Handler) BlockUser(c *gin.Context) {
	user, ok := currentUser(c)
	if !ok {
		return
	}
	var req BlockUserRequest
	_ = c.ShouldBindJSON(&req)
	if err := h.service.BlockUser(c.Request.Context(), user.UserID, c.Param("userUuid"), req, BlockSourceManual, nil); err != nil {
		h.writeError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "User blocked successfully", nil)
}

func (h *Handler) UnblockUser(c *gin.Context) {
	user, ok := currentUser(c)
	if !ok {
		return
	}
	if err := h.service.UnblockUser(c.Request.Context(), user.UserID, c.Param("userUuid")); err != nil {
		h.writeError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "User unblocked successfully", nil)
}

func (h *Handler) ListBlocks(c *gin.Context) {
	user, ok := currentUser(c)
	if !ok {
		return
	}
	result, err := h.service.ListBlocks(c.Request.Context(), user.UserID)
	if err != nil {
		h.writeError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Blocked users fetched successfully", result)
}

func (h *Handler) GetSettings(c *gin.Context) {
	user, ok := currentUser(c)
	if !ok {
		return
	}
	result, err := h.service.GetSettings(c.Request.Context(), user.UserID)
	if err != nil {
		h.writeError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Safety settings fetched successfully", result)
}

func (h *Handler) UpdateSettings(c *gin.Context) {
	user, ok := currentUser(c)
	if !ok {
		return
	}
	var req UpdateSafetySettingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Validation(c, err.Error())
		return
	}
	result, err := h.service.UpdateSettings(c.Request.Context(), user.UserID, req)
	if err != nil {
		h.writeError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Safety settings updated successfully", result)
}

func (h *Handler) AdminListReports(c *gin.Context) {
	result, err := h.service.AdminListReports(c.Request.Context(), AdminReportListQuery{
		Status:      c.Query("status"),
		TargetType:  c.Query("targetType"),
		Severity:    c.Query("severity"),
		CreatedFrom: c.Query("createdFrom"),
		CreatedTo:   c.Query("createdTo"),
		Limit:       c.Query("limit"),
		Cursor:      c.Query("cursor"),
	})
	if err != nil {
		h.writeError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Reports fetched successfully", result)
}

func (h *Handler) AdminReportDetail(c *gin.Context) {
	result, err := h.service.AdminReportDetail(c.Request.Context(), c.Param("reportId"))
	if err != nil {
		h.writeError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Report fetched successfully", result)
}

func (h *Handler) AdminReviewReport(c *gin.Context) {
	adminUser, ok := currentAdmin(c)
	if !ok {
		return
	}
	var req ReviewReportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Validation(c, err.Error())
		return
	}
	result, err := h.service.AdminReviewReport(c.Request.Context(), adminUser.AdminUserID, c.Param("reportId"), req, adminMeta(c))
	if err != nil {
		h.writeError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Report reviewed successfully", result)
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

func currentAdmin(c *gin.Context) (adminpkg.AuthenticatedAdmin, bool) {
	adminUser, ok := adminpkg.CurrentAdmin(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, "Unauthorized", adminpkg.CodeUnauthorized, nil)
		return adminpkg.AuthenticatedAdmin{}, false
	}
	return adminUser, true
}

func adminMeta(c *gin.Context) adminpkg.RequestMeta {
	return adminpkg.RequestMeta{IPAddress: c.ClientIP(), UserAgent: c.Request.UserAgent()}
}

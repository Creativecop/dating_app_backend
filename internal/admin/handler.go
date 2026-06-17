package admin

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/neoscoder/aura-backend/internal/response"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Validation(c, err.Error())
		return
	}
	result, err := h.service.Login(c.Request.Context(), req, requestMeta(c))
	if err != nil {
		h.writeError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Admin login successful", result)
}

func (h *Handler) RefreshToken(c *gin.Context) {
	var req RefreshTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Validation(c, err.Error())
		return
	}
	result, err := h.service.RefreshToken(c.Request.Context(), req, requestMeta(c))
	if err != nil {
		h.writeError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Admin token refreshed successfully", result)
}

func (h *Handler) Logout(c *gin.Context) {
	adminUser, ok := CurrentAdmin(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, "Unauthorized", CodeUnauthorized, nil)
		return
	}
	var req LogoutRequest
	_ = c.ShouldBindJSON(&req)
	if err := h.service.Logout(c.Request.Context(), adminUser.AdminUserID, req); err != nil {
		h.writeError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Admin logged out successfully", nil)
}

func (h *Handler) ChangePassword(c *gin.Context) {
	adminUser, ok := CurrentAdmin(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, "Unauthorized", CodeUnauthorized, nil)
		return
	}
	var req ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Validation(c, err.Error())
		return
	}
	if err := h.service.ChangePassword(c.Request.Context(), adminUser.AdminUserID, req, requestMeta(c)); err != nil {
		h.writeError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Password changed successfully. Please log in again.", nil)
}

func (h *Handler) Me(c *gin.Context) {
	adminUser, ok := CurrentAdmin(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, "Unauthorized", CodeUnauthorized, nil)
		return
	}
	result, err := h.service.Me(c.Request.Context(), adminUser.AdminUserID)
	if err != nil {
		h.writeError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Admin fetched successfully", result)
}

func (h *Handler) Capabilities(c *gin.Context) {
	result, err := h.service.Capabilities(c.Request.Context())
	if err != nil {
		h.writeError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Admin capabilities fetched successfully", result)
}

func (h *Handler) ListRoles(c *gin.Context) {
	result, err := h.service.ListRoles(c.Request.Context())
	if err != nil {
		h.writeError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Admin roles fetched successfully", result)
}

func (h *Handler) ListAdminUsers(c *gin.Context) {
	result, err := h.service.ListAdminUsers(c.Request.Context())
	if err != nil {
		h.writeError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Admin users fetched successfully", result)
}

func (h *Handler) AdminUserDetail(c *gin.Context) {
	result, err := h.service.AdminUserDetail(c.Request.Context(), c.Param("adminUserUuid"))
	if err != nil {
		h.writeError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Admin user fetched successfully", result)
}

func (h *Handler) CreateAdminUser(c *gin.Context) {
	adminUser, ok := CurrentAdmin(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, "Unauthorized", CodeUnauthorized, nil)
		return
	}
	var req CreateAdminUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Validation(c, err.Error())
		return
	}
	result, err := h.service.CreateAdminUser(c.Request.Context(), adminUser.AdminUserID, req, requestMeta(c))
	if err != nil {
		h.writeError(c, err)
		return
	}
	response.Success(c, http.StatusCreated, "Admin user created successfully", result)
}

func (h *Handler) AssignAdminRole(c *gin.Context) {
	adminUser, ok := CurrentAdmin(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, "Unauthorized", CodeUnauthorized, nil)
		return
	}
	var req AssignRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Validation(c, err.Error())
		return
	}
	result, err := h.service.AssignAdminRole(c.Request.Context(), adminUser.AdminUserID, c.Param("adminUserUuid"), req, requestMeta(c))
	if err != nil {
		h.writeError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Admin role assigned successfully", result)
}

func (h *Handler) RemoveAdminRole(c *gin.Context) {
	adminUser, ok := CurrentAdmin(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, "Unauthorized", CodeUnauthorized, nil)
		return
	}
	var req RemoveRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Validation(c, err.Error())
		return
	}
	result, err := h.service.RemoveAdminRole(c.Request.Context(), adminUser.AdminUserID, c.Param("adminUserUuid"), c.Param("role"), req, requestMeta(c))
	if err != nil {
		h.writeError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Admin role removed successfully", result)
}

func (h *Handler) UpdateAdminStatus(c *gin.Context) {
	adminUser, ok := CurrentAdmin(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, "Unauthorized", CodeUnauthorized, nil)
		return
	}
	var req UpdateAdminStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Validation(c, err.Error())
		return
	}
	result, err := h.service.UpdateAdminStatus(c.Request.Context(), adminUser.AdminUserID, c.Param("adminUserUuid"), req, requestMeta(c))
	if err != nil {
		h.writeError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Admin status updated successfully", result)
}

func (h *Handler) ListUsers(c *gin.Context) {
	result, err := h.service.ListUsers(c.Request.Context(), AdminMobileUserListQuery{
		Search:      c.Query("search"),
		Status:      c.Query("status"),
		CreatedFrom: c.Query("createdFrom"),
		CreatedTo:   c.Query("createdTo"),
		Limit:       c.Query("limit"),
		Cursor:      c.Query("cursor"),
	})
	if err != nil {
		h.writeError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Users fetched successfully", result)
}

func (h *Handler) UserDetail(c *gin.Context) {
	result, err := h.service.UserDetail(c.Request.Context(), c.Param("userId"))
	if err != nil {
		h.writeError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "User fetched successfully", result)
}

func (h *Handler) ListUserRestrictions(c *gin.Context) {
	result, err := h.service.ListUserRestrictions(c.Request.Context(), c.Param("userId"), c.Query("status"))
	if err != nil {
		h.writeError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "User restrictions fetched successfully", result)
}

func (h *Handler) CreateUserRestriction(c *gin.Context) {
	adminUser, ok := CurrentAdmin(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, "Unauthorized", CodeUnauthorized, nil)
		return
	}
	var req CreateUserRestrictionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Validation(c, err.Error())
		return
	}
	result, err := h.service.CreateUserRestriction(c.Request.Context(), adminUser.AdminUserID, c.Param("userId"), req, requestMeta(c))
	if err != nil {
		h.writeError(c, err)
		return
	}
	response.Success(c, http.StatusCreated, "User restriction created successfully", result)
}

func (h *Handler) RevokeUserRestriction(c *gin.Context) {
	adminUser, ok := CurrentAdmin(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, "Unauthorized", CodeUnauthorized, nil)
		return
	}
	var req RevokeUserRestrictionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Validation(c, err.Error())
		return
	}
	result, err := h.service.RevokeUserRestriction(c.Request.Context(), adminUser.AdminUserID, c.Param("userId"), c.Param("restrictionId"), req, requestMeta(c))
	if err != nil {
		h.writeError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "User restriction revoked successfully", result)
}

func (h *Handler) ListAuditLogs(c *gin.Context) {
	result, err := h.service.ListAuditLogs(c.Request.Context(), AuditLogListQuery{
		AdminUserUUID: c.Query("adminUserUuid"),
		Action:        c.Query("action"),
		ResourceType:  c.Query("resourceType"),
		ResourceUUID:  c.Query("resourceUuid"),
		CreatedFrom:   c.Query("createdFrom"),
		CreatedTo:     c.Query("createdTo"),
		Limit:         c.Query("limit"),
		Cursor:        c.Query("cursor"),
	})
	if err != nil {
		h.writeError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Audit logs fetched successfully", result)
}

func (h *Handler) AuditLogDetail(c *gin.Context) {
	result, err := h.service.AuditLogDetail(c.Request.Context(), c.Param("auditLogUuid"))
	if err != nil {
		h.writeError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Audit log fetched successfully", result)
}

func (h *Handler) writeError(c *gin.Context, err error) {
	if serviceErr, ok := AsServiceError(err); ok {
		response.Error(c, serviceErr.Status, serviceErr.Message, serviceErr.Code, serviceErr.Details)
		return
	}
	if err == ErrInactiveAdmin || err == ErrUnauthorized || err == ErrInvalidRefresh {
		response.Error(c, http.StatusUnauthorized, PublicErrorMessage(err), CodeUnauthorized, nil)
		return
	}
	response.Internal(c)
}

func requestMeta(c *gin.Context) RequestMeta {
	return RequestMeta{IPAddress: c.ClientIP(), UserAgent: c.Request.UserAgent()}
}

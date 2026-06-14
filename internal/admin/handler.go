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

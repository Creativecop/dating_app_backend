package auth

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/neoscoder/aura-backend/internal/response"
)

type Handler struct {
	service *Service
	appEnv  string
}

func NewHandler(service *Service, appEnv string) *Handler {
	return &Handler{service: service, appEnv: appEnv}
}

func (h *Handler) RequestOTP(c *gin.Context) {
	var req RequestOTPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Validation(c, err.Error())
		return
	}

	if err := h.service.RequestOTP(c.Request.Context(), req, requestMeta(c)); err != nil {
		h.writeError(c, err)
		return
	}

	message := "OTP has been sent successfully."
	if h.appEnv == "production" {
		message = "If the account is valid, an OTP has been sent."
	}
	response.Success(c, http.StatusOK, message, nil)
}

func (h *Handler) VerifyOTP(c *gin.Context) {
	var req VerifyOTPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Validation(c, err.Error())
		return
	}

	result, err := h.service.VerifyOTP(c.Request.Context(), req, requestMeta(c))
	if err != nil {
		h.writeError(c, err)
		return
	}

	response.Success(c, http.StatusOK, "Login successful", result)
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

	response.Success(c, http.StatusOK, "Token refreshed successfully", result)
}

func (h *Handler) Logout(c *gin.Context) {
	user, ok := CurrentUser(c)
	if !ok {
		response.Unauthorized(c, "Unauthorized.")
		return
	}

	var req LogoutRequest
	_ = c.ShouldBindJSON(&req)

	if err := h.service.Logout(c.Request.Context(), user, req); err != nil {
		h.writeError(c, err)
		return
	}

	response.Success(c, http.StatusOK, "Logged out successfully", nil)
}

func (h *Handler) LogoutAll(c *gin.Context) {
	user, ok := CurrentUser(c)
	if !ok {
		response.Unauthorized(c, "Unauthorized.")
		return
	}

	if err := h.service.LogoutAll(c.Request.Context(), user); err != nil {
		h.writeError(c, err)
		return
	}

	response.Success(c, http.StatusOK, "Logged out from all devices successfully", nil)
}

func (h *Handler) Me(c *gin.Context) {
	user, ok := CurrentUser(c)
	if !ok {
		response.Unauthorized(c, "Unauthorized.")
		return
	}

	result, err := h.service.Me(c.Request.Context(), user)
	if err != nil {
		h.writeError(c, err)
		return
	}

	response.Success(c, http.StatusOK, "User fetched successfully", result)
}

func (h *Handler) DeleteAccount(c *gin.Context) {
	user, ok := CurrentUser(c)
	if !ok {
		response.Unauthorized(c, "Unauthorized.")
		return
	}

	if err := h.service.DeleteAccount(c.Request.Context(), user); err != nil {
		h.writeError(c, err)
		return
	}

	response.Success(c, http.StatusOK, "Account deleted successfully", nil)
}

func (h *Handler) writeError(c *gin.Context, err error) {
	status := PublicStatusCode(err)
	if status >= 500 {
		response.Internal(c)
		return
	}
	response.Error(c, status, PublicErrorMessage(err), PublicErrorCode(err), nil)
}

func requestMeta(c *gin.Context) RequestMeta {
	return RequestMeta{
		IPAddress: c.ClientIP(),
		UserAgent: c.Request.UserAgent(),
	}
}

func CurrentUser(c *gin.Context) (AuthenticatedUser, bool) {
	value, ok := c.Get(ContextUserKey)
	if !ok {
		return AuthenticatedUser{}, false
	}
	user, ok := value.(AuthenticatedUser)
	return user, ok
}

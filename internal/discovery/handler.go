package discovery

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/neoscoder/aura-backend/internal/auth"
	"github.com/neoscoder/aura-backend/internal/response"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) GetPreferences(c *gin.Context) {
	user, ok := currentUser(c)
	if !ok {
		return
	}

	result, err := h.service.GetPreferences(c.Request.Context(), user.UserID)
	if err != nil {
		h.writeError(c, err)
		return
	}

	response.Success(c, http.StatusOK, "Discovery preferences fetched successfully", result)
}

func (h *Handler) UpdatePreferences(c *gin.Context) {
	user, ok := currentUser(c)
	if !ok {
		return
	}

	var req UpdatePreferencesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Validation(c, err.Error())
		return
	}

	result, err := h.service.UpdatePreferences(c.Request.Context(), user.UserID, req)
	if err != nil {
		h.writeError(c, err)
		return
	}

	response.Success(c, http.StatusOK, "Discovery preferences updated successfully", result)
}

func (h *Handler) Readiness(c *gin.Context) {
	user, ok := currentUser(c)
	if !ok {
		return
	}

	result, err := h.service.Readiness(c.Request.Context(), user.UserID)
	if err != nil {
		h.writeError(c, err)
		return
	}

	response.Success(c, http.StatusOK, "Discovery readiness fetched successfully", result)
}

func (h *Handler) Feed(c *gin.Context) {
	user, ok := currentUser(c)
	if !ok {
		return
	}

	result, err := h.service.Feed(c.Request.Context(), user.UserID, c.Query("limit"), c.Query("cursor"))
	if err != nil {
		h.writeError(c, err)
		return
	}

	response.Success(c, http.StatusOK, "Discovery feed fetched successfully", result)
}

func (h *Handler) ProfileDetail(c *gin.Context) {
	user, ok := currentUser(c)
	if !ok {
		return
	}

	result, err := h.service.ProfileDetail(c.Request.Context(), user.UserID, c.Param("userUuid"))
	if err != nil {
		h.writeError(c, err)
		return
	}

	response.Success(c, http.StatusOK, "Discovery profile fetched successfully", result)
}

func (h *Handler) CreateImpressions(c *gin.Context) {
	user, ok := currentUser(c)
	if !ok {
		return
	}

	var req CreateImpressionsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Validation(c, err.Error())
		return
	}

	result, err := h.service.CreateImpressions(c.Request.Context(), user.UserID, req)
	if err != nil {
		h.writeError(c, err)
		return
	}

	response.Success(c, http.StatusOK, "Discovery impressions saved successfully", result)
}

func (h *Handler) CreateAction(c *gin.Context) {
	user, ok := currentUser(c)
	if !ok {
		return
	}

	var req CreateActionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Validation(c, err.Error())
		return
	}

	result, matched, err := h.service.CreateAction(c.Request.Context(), user.UserID, req)
	if err != nil {
		h.writeError(c, err)
		return
	}

	if matched {
		response.Success(c, http.StatusCreated, "It's a match!", result)
		return
	}
	response.Success(c, http.StatusOK, "Action saved successfully", result)
}

func (h *Handler) BlockUser(c *gin.Context) {
	user, ok := currentUser(c)
	if !ok {
		return
	}

	var req BlockUserRequest
	_ = c.ShouldBindJSON(&req)

	if err := h.service.BlockUser(c.Request.Context(), user.UserID, c.Param("userUuid"), req); err != nil {
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

func (h *Handler) writeError(c *gin.Context, err error) {
	if serviceErr, ok := AsServiceError(err); ok {
		response.Error(c, serviceErr.Status, serviceErr.Message, serviceErr.Code, serviceErr.Details)
		return
	}
	if codedErr, ok := err.(interface {
		StatusCode() int
		ErrorCode() string
		ErrorDetails() any
	}); ok {
		response.Error(c, codedErr.StatusCode(), err.Error(), codedErr.ErrorCode(), codedErr.ErrorDetails())
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

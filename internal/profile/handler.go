package profile

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

func (h *Handler) GetMe(c *gin.Context) {
	user, ok := currentUser(c)
	if !ok {
		return
	}

	result, err := h.service.GetMe(c.Request.Context(), user.UserID)
	if err != nil {
		h.writeError(c, err)
		return
	}

	response.Success(c, http.StatusOK, "Profile fetched successfully", result)
}

func (h *Handler) UpdateProfile(c *gin.Context) {
	user, ok := currentUser(c)
	if !ok {
		return
	}

	var req PatchProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Validation(c, err.Error())
		return
	}

	result, err := h.service.UpdateProfile(c.Request.Context(), user.UserID, req)
	if err != nil {
		h.writeError(c, err)
		return
	}

	response.Success(c, http.StatusOK, "Profile updated successfully", result)
}

func (h *Handler) InterestCatalog(c *gin.Context) {
	if _, ok := currentUser(c); !ok {
		return
	}

	result, err := h.service.InterestCatalog(c.Request.Context())
	if err != nil {
		h.writeError(c, err)
		return
	}

	response.Success(c, http.StatusOK, "Interests fetched successfully", result)
}

func (h *Handler) UpdateInterests(c *gin.Context) {
	user, ok := currentUser(c)
	if !ok {
		return
	}

	var req UpdateInterestsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Validation(c, err.Error())
		return
	}

	result, err := h.service.UpdateInterests(c.Request.Context(), user.UserID, req)
	if err != nil {
		h.writeError(c, err)
		return
	}

	response.Success(c, http.StatusOK, "Interests updated successfully", result)
}

func (h *Handler) PromptCatalog(c *gin.Context) {
	if _, ok := currentUser(c); !ok {
		return
	}

	result, err := h.service.PromptCatalog(c.Request.Context())
	if err != nil {
		h.writeError(c, err)
		return
	}

	response.Success(c, http.StatusOK, "Profile prompts fetched successfully", result)
}

func (h *Handler) UpdatePrompts(c *gin.Context) {
	user, ok := currentUser(c)
	if !ok {
		return
	}

	var req UpdatePromptsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Validation(c, err.Error())
		return
	}

	result, err := h.service.UpdatePrompts(c.Request.Context(), user.UserID, req)
	if err != nil {
		h.writeError(c, err)
		return
	}

	response.Success(c, http.StatusOK, "Profile prompts updated successfully", result)
}

func (h *Handler) LifestyleCatalog(c *gin.Context) {
	if _, ok := currentUser(c); !ok {
		return
	}

	result, err := h.service.LifestyleCatalog(c.Request.Context())
	if err != nil {
		h.writeError(c, err)
		return
	}

	response.Success(c, http.StatusOK, "Lifestyle questions fetched successfully", result)
}

func (h *Handler) UpdateLifestyle(c *gin.Context) {
	user, ok := currentUser(c)
	if !ok {
		return
	}

	var req UpdateLifestyleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Validation(c, err.Error())
		return
	}

	result, err := h.service.UpdateLifestyle(c.Request.Context(), user.UserID, req)
	if err != nil {
		h.writeError(c, err)
		return
	}

	response.Success(c, http.StatusOK, "Lifestyle answers updated successfully", result)
}

func (h *Handler) CompleteProfile(c *gin.Context) {
	user, ok := currentUser(c)
	if !ok {
		return
	}

	result, err := h.service.CompleteProfile(c.Request.Context(), user.UserID)
	if err != nil {
		h.writeError(c, err)
		return
	}

	response.Success(c, http.StatusOK, "Profile completed successfully", result)
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

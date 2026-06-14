package notification

import (
	"net/http"
	"strconv"

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

func (h *Handler) UpsertFCMToken(c *gin.Context) {
	user, ok := currentUser(c)
	if !ok {
		return
	}
	var req UpsertFCMTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Validation(c, err.Error())
		return
	}
	result, err := h.service.UpsertFCMToken(c.Request.Context(), user.UserID, req)
	if err != nil {
		h.writeError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Device token saved successfully", result)
}

func (h *Handler) DeleteFCMToken(c *gin.Context) {
	user, ok := currentUser(c)
	if !ok {
		return
	}
	var req DeleteFCMTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Validation(c, err.Error())
		return
	}
	result, err := h.service.DeleteFCMToken(c.Request.Context(), user.UserID, req)
	if err != nil {
		h.writeError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Device token removed successfully", result)
}

func (h *Handler) ListNotifications(c *gin.Context) {
	user, ok := currentUser(c)
	if !ok {
		return
	}
	limit, _ := strconv.Atoi(c.Query("limit"))
	result, err := h.service.ListNotifications(c.Request.Context(), user.UserID, limit, c.Query("cursor"))
	if err != nil {
		h.writeError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Notifications fetched successfully", result)
}

func (h *Handler) MarkRead(c *gin.Context) {
	user, ok := currentUser(c)
	if !ok {
		return
	}
	result, err := h.service.MarkRead(c.Request.Context(), user.UserID, c.Param("notificationUuid"))
	if err != nil {
		h.writeError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Notification marked as read", result)
}

func (h *Handler) MarkAllRead(c *gin.Context) {
	user, ok := currentUser(c)
	if !ok {
		return
	}
	result, err := h.service.MarkAllRead(c.Request.Context(), user.UserID)
	if err != nil {
		h.writeError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Notifications marked as read", result)
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
	response.Success(c, http.StatusOK, "Notification settings fetched successfully", result)
}

func (h *Handler) UpdateSettings(c *gin.Context) {
	user, ok := currentUser(c)
	if !ok {
		return
	}
	var req UpdateSettingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Validation(c, err.Error())
		return
	}
	result, err := h.service.UpdateSettings(c.Request.Context(), user.UserID, req)
	if err != nil {
		h.writeError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Notification settings updated successfully", result)
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

package media

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/neoscoder/aura-backend/internal/auth"
	"github.com/neoscoder/aura-backend/internal/config"
	"github.com/neoscoder/aura-backend/internal/response"
)

type Handler struct {
	service *Service
	cfg     config.MediaConfig
}

func NewHandler(service *Service, cfg config.MediaConfig) *Handler {
	return &Handler{service: service, cfg: cfg}
}

func (h *Handler) ListMine(c *gin.Context) {
	user, ok := currentUser(c)
	if !ok {
		return
	}
	result, err := h.service.ListMine(c.Request.Context(), user)
	if err != nil {
		h.writeError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Media fetched successfully", result)
}

func (h *Handler) UploadPhoto(c *gin.Context) {
	user, ok := currentUser(c)
	if !ok {
		return
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, mbToBytes(h.cfg.MaxPhotoSizeMB)+1024*1024)
	file, err := c.FormFile("file")
	if err != nil {
		response.Validation(c, "file is required")
		return
	}
	isPrimary, _ := strconv.ParseBool(c.PostForm("isPrimary"))
	result, err := h.service.UploadPhoto(c.Request.Context(), user, file, isPrimary)
	if err != nil {
		h.writeError(c, err)
		return
	}
	response.Success(c, http.StatusAccepted, "Photo uploaded and is being processed.", result)
}

func (h *Handler) UploadIntroVideo(c *gin.Context) {
	user, ok := currentUser(c)
	if !ok {
		return
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, mbToBytes(h.cfg.MaxVideoSizeMB)+1024*1024)
	file, err := c.FormFile("file")
	if err != nil {
		response.Validation(c, "file is required")
		return
	}
	result, err := h.service.UploadIntroVideo(c.Request.Context(), user, file)
	if err != nil {
		h.writeError(c, err)
		return
	}
	response.Success(c, http.StatusAccepted, "Intro video uploaded and is being processed.", result)
}

func (h *Handler) SetPrimary(c *gin.Context) {
	user, ok := currentUser(c)
	if !ok {
		return
	}
	result, err := h.service.SetPrimary(c.Request.Context(), user.UserID, c.Param("mediaUuid"))
	if err != nil {
		h.writeError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Primary photo updated successfully", result)
}

func (h *Handler) Reorder(c *gin.Context) {
	user, ok := currentUser(c)
	if !ok {
		return
	}
	var req ReorderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Validation(c, err.Error())
		return
	}
	result, err := h.service.Reorder(c.Request.Context(), user.UserID, req)
	if err != nil {
		h.writeError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Media reordered successfully", result)
}

func (h *Handler) Delete(c *gin.Context) {
	user, ok := currentUser(c)
	if !ok {
		return
	}
	if err := h.service.Delete(c.Request.Context(), user.UserID, c.Param("mediaUuid")); err != nil {
		h.writeError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "Media deleted successfully", nil)
}

func (h *Handler) Serve(c *gin.Context) {
	user, ok := currentUser(c)
	if !ok {
		return
	}
	result, err := h.service.ServeVariant(c.Request.Context(), user.UserID, c.Param("mediaUuid"), c.Param("variant"))
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.Header("Content-Type", result.MimeType)
	c.Header("Cache-Control", "private, max-age=3600")
	c.Header("X-Content-Type-Options", "nosniff")
	if h.cfg.ServeMode == "x_accel" {
		c.Header("X-Accel-Redirect", result.AccelPath)
		c.Status(http.StatusOK)
		return
	}
	c.File(result.LocalPath)
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

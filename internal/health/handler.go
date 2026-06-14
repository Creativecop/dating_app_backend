package health

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	goredis "github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"github.com/neoscoder/aura-backend/internal/response"
)

type Handler struct {
	DB    *gorm.DB
	Redis *goredis.Client
}

func NewHandler(db *gorm.DB, redis *goredis.Client) *Handler {
	return &Handler{DB: db, Redis: redis}
}

func (h *Handler) Handle(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
	defer cancel()

	data := gin.H{
		"api":      "ok",
		"postgres": "ok",
		"redis":    "ok",
	}

	healthy := true

	sqlDB, err := h.DB.DB()
	if err != nil || sqlDB.PingContext(ctx) != nil {
		healthy = false
		data["postgres"] = "unavailable"
	}

	if h.Redis == nil || h.Redis.Ping(ctx).Err() != nil {
		healthy = false
		data["redis"] = "unavailable"
	}

	if !healthy {
		response.Error(c, http.StatusServiceUnavailable, "Service unhealthy", "SERVICE_UNHEALTHY", data)
		return
	}

	response.Success(c, http.StatusOK, "Service healthy", data)
}

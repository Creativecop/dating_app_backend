package router

import (
	"time"

	"github.com/gin-gonic/gin"
	goredis "github.com/redis/go-redis/v9"

	"github.com/neoscoder/aura-backend/internal/admin"
	"github.com/neoscoder/aura-backend/internal/auth"
	"github.com/neoscoder/aura-backend/internal/chat"
	"github.com/neoscoder/aura-backend/internal/config"
	"github.com/neoscoder/aura-backend/internal/discovery"
	"github.com/neoscoder/aura-backend/internal/health"
	"github.com/neoscoder/aura-backend/internal/location"
	appmatch "github.com/neoscoder/aura-backend/internal/match"
	"github.com/neoscoder/aura-backend/internal/media"
	"github.com/neoscoder/aura-backend/internal/middleware"
	"github.com/neoscoder/aura-backend/internal/notification"
	"github.com/neoscoder/aura-backend/internal/profile"
	"github.com/neoscoder/aura-backend/internal/safety"
	"github.com/neoscoder/aura-backend/internal/subscription"
)

type Dependencies struct {
	Config              *config.Config
	RedisClient         *goredis.Client
	HealthHandler       *health.Handler
	AuthHandler         *auth.Handler
	AuthService         *auth.Service
	AdminHandler        *admin.Handler
	AdminService        *admin.Service
	ProfileHandler      *profile.Handler
	MediaHandler        *media.Handler
	MatchHandler        *appmatch.Handler
	ChatHandler         *chat.Handler
	LocationHandler     *location.Handler
	DiscoveryHandler    *discovery.Handler
	NotificationHandler *notification.Handler
	SafetyHandler       *safety.Handler
	SubscriptionHandler *subscription.Handler
}

func New(deps Dependencies) *gin.Engine {
	if deps.Config.App.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	r.MaxMultipartMemory = int64(deps.Config.Media.MaxMultipartMemoryMB) << 20
	r.Use(
		middleware.RequestID(),
		middleware.Recovery(),
		middleware.Logger(deps.Config.App.Env),
		middleware.CORS(deps.Config.CORS.AllowedOrigins),
	)
	limiter := middleware.NewRateLimiter(deps.RedisClient, deps.Config.RateLimit.Enabled)
	bodyLimit := middleware.JSONBodyLimit(deps.Config.Request.JSONBodyLimitBytes)

	v1 := r.Group("/api/v1")
	r.GET("/health", deps.HealthHandler.Handle)
	v1.GET("/health", deps.HealthHandler.Handle)

	r.StaticFile("/swagger.html", "docs/swagger.html")
	r.StaticFile("/swagger", "docs/swagger.html")
	r.StaticFile("/openapi.yaml", "docs/openapi.yaml")
	r.StaticFile("/docs/API_MOBILE.md", "docs/API_MOBILE.md")

	authRoutes := v1.Group("/auth")
	authRoutes.Use(bodyLimit)
	authRoutes.POST("/request-otp", limiter.Limit(
		middleware.RateLimitRule{Scope: "otp_request_identifier_10m", Limit: deps.Config.RateLimit.OTPRequestIdentifier10M, Window: 10 * time.Minute, Identifier: middleware.BodyFieldIdentifier("phone", "email"), FailClosed: deps.Config.RateLimit.RedisRequiredForAuth},
		middleware.RateLimitRule{Scope: "otp_request_identifier_1h", Limit: deps.Config.RateLimit.OTPRequestIdentifier1H, Window: time.Hour, Identifier: middleware.BodyFieldIdentifier("phone", "email"), FailClosed: deps.Config.RateLimit.RedisRequiredForAuth},
		middleware.RateLimitRule{Scope: "otp_request_ip_1h", Limit: deps.Config.RateLimit.OTPRequestIP1H, Window: time.Hour, Identifier: middleware.IPIdentifier(), FailClosed: deps.Config.RateLimit.RedisRequiredForAuth},
	), deps.AuthHandler.RequestOTP)
	authRoutes.POST("/verify-otp", limiter.Limit(
		middleware.RateLimitRule{Scope: "otp_verify_identifier_10m", Limit: deps.Config.RateLimit.OTPVerifyIdentifier10M, Window: 10 * time.Minute, Identifier: middleware.BodyFieldIdentifier("phone", "email"), FailClosed: true},
		middleware.RateLimitRule{Scope: "otp_verify_ip_1h", Limit: deps.Config.RateLimit.OTPVerifyIP1H, Window: time.Hour, Identifier: middleware.IPIdentifier(), FailClosed: true},
	), deps.AuthHandler.VerifyOTP)
	authRoutes.POST("/refresh-token", limiter.Limit(
		middleware.RateLimitRule{Scope: "mobile_refresh_subject_1m", Limit: deps.Config.RateLimit.RefreshSubject1M, Window: time.Minute, Identifier: middleware.BodyFieldIdentifier("refreshToken"), FailClosed: false},
		middleware.RateLimitRule{Scope: "mobile_refresh_ip_1m", Limit: deps.Config.RateLimit.RefreshIP1M, Window: time.Minute, Identifier: middleware.IPIdentifier(), FailClosed: false},
	), deps.AuthHandler.RefreshToken)

	protectedAuth := authRoutes.Group("")
	protectedAuth.Use(middleware.Auth(deps.AuthService))
	protectedAuth.POST("/logout", deps.AuthHandler.Logout)
	protectedAuth.POST("/logout-all", deps.AuthHandler.LogoutAll)
	protectedAuth.GET("/me", deps.AuthHandler.Me)
	protectedAuth.DELETE("/account", deps.AuthHandler.DeleteAccount)

	profile.RegisterRoutes(v1, middleware.Auth(deps.AuthService), deps.ProfileHandler)
	media.RegisterRoutes(v1, middleware.Auth(deps.AuthService), deps.MediaHandler)
	appmatch.RegisterRoutes(v1, middleware.Auth(deps.AuthService), deps.MatchHandler)
	chat.RegisterRoutes(r, v1, middleware.Auth(deps.AuthService), deps.ChatHandler, limiter, deps.Config.RateLimit)
	location.RegisterRoutes(v1, middleware.Auth(deps.AuthService), deps.LocationHandler)
	discovery.RegisterRoutes(v1, middleware.Auth(deps.AuthService), deps.DiscoveryHandler)
	notification.RegisterRoutes(v1, middleware.Auth(deps.AuthService), deps.NotificationHandler)
	safety.RegisterRoutes(v1, middleware.Auth(deps.AuthService), deps.SafetyHandler, limiter, deps.Config.RateLimit, bodyLimit)
	admin.RegisterRoutes(v1, deps.AdminService, deps.AdminHandler, limiter, deps.Config.RateLimit, bodyLimit)
	safety.RegisterAdminRoutes(v1, deps.AdminService, deps.SafetyHandler, limiter, deps.Config.RateLimit, bodyLimit)
	subscription.RegisterRoutes(v1, middleware.Auth(deps.AuthService), deps.SubscriptionHandler, limiter, deps.Config.RateLimit, bodyLimit)
	subscription.RegisterAdminRoutes(v1, deps.AdminService, deps.SubscriptionHandler, limiter, deps.Config.RateLimit, bodyLimit)

	return r
}

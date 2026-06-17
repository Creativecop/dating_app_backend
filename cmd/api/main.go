package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/neoscoder/aura-backend/internal/admin"
	"github.com/neoscoder/aura-backend/internal/auth"
	"github.com/neoscoder/aura-backend/internal/chat"
	"github.com/neoscoder/aura-backend/internal/config"
	"github.com/neoscoder/aura-backend/internal/database"
	"github.com/neoscoder/aura-backend/internal/discovery"
	"github.com/neoscoder/aura-backend/internal/health"
	"github.com/neoscoder/aura-backend/internal/location"
	appmatch "github.com/neoscoder/aura-backend/internal/match"
	"github.com/neoscoder/aura-backend/internal/media"
	"github.com/neoscoder/aura-backend/internal/notification"
	"github.com/neoscoder/aura-backend/internal/otp"
	"github.com/neoscoder/aura-backend/internal/profile"
	"github.com/neoscoder/aura-backend/internal/queue"
	appredis "github.com/neoscoder/aura-backend/internal/redis"
	"github.com/neoscoder/aura-backend/internal/restriction"
	"github.com/neoscoder/aura-backend/internal/router"
	"github.com/neoscoder/aura-backend/internal/safety"
	"github.com/neoscoder/aura-backend/internal/storage"
	"github.com/neoscoder/aura-backend/internal/subscription"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	db, err := database.Connect(ctx, cfg.Database)
	if err != nil {
		log.Fatalf("connect postgres: %v", err)
	}
	defer func() {
		if err := database.Close(db); err != nil {
			log.Printf("close postgres: %v", err)
		}
	}()

	redisClient := appredis.NewClient(cfg.Redis)
	if err := appredis.Ping(ctx, redisClient); err != nil {
		log.Fatalf("connect redis: %v", err)
	}
	defer func() {
		if err := redisClient.Close(); err != nil {
			log.Printf("close redis: %v", err)
		}
	}()

	queueClient := queue.NewClient(cfg.Redis)
	defer queueClient.Close()

	storageProvider, err := storage.NewLocalProvider(cfg.Media.LocalRoot)
	if err != nil {
		log.Fatalf("create storage provider: %v", err)
	}

	otpService := otp.NewService(db, redisClient, queueClient, cfg.OTP)
	profileService := profile.NewService(db, cfg.Discovery)
	locationService := location.NewService(db)
	locationService.SetDiscoveryEligibilityRefresher(profileService)
	discoveryService := discovery.NewService(db, cfg.Discovery)
	discoveryService.SetDiscoveryEligibilityRefresher(profileService)
	adminService := admin.NewService(db, cfg.AdminJWT)
	restrictionService := restriction.NewService(db)
	subscriptionService := subscription.NewService(db)
	discoveryService.SetActionUsageLimiter(subscriptionService)
	matchService := appmatch.NewService(db)
	chatHub := chat.NewHub()
	chatService := chat.NewService(db, chatHub)
	notificationService := notification.NewService(db, queueClient, notification.NewConfigAdapter(
		cfg.Notification.PushEnabled,
		cfg.Notification.Provider,
		cfg.Notification.PushMaxRetry,
		cfg.Notification.PushTimeoutSeconds,
		cfg.Notification.PushGraceSeconds,
		cfg.Notification.DefaultTimezone,
	))
	safetyService := safety.NewService(db)
	discoveryService.SetMatchNotificationDispatcher(notificationService)
	discoveryService.SetBlockChecker(safetyService)
	matchService.SetBlockChecker(safetyService)
	chatService.SetNotificationDispatcher(notificationService)
	mediaService := media.NewService(db, storageProvider, queueClient, cfg.Media)
	mediaService.SetDiscoveryEligibilityRefresher(profileService)
	mediaService.SetMediaVisibilityAuthorizer(discoveryService)
	mediaService.AddMediaVisibilityAuthorizer(matchService)
	authService := auth.NewService(db, cfg, otpService)
	authService.SetRestrictionChecker(restrictionService)
	authService.SetProfileEnsurer(profileService)
	authService.SetDiscoveryPreferencesEnsurer(discoveryService)
	authService.SetNotificationSettingsEnsurer(notificationService)
	authService.SetSafetySettingsEnsurer(safetyService)
	authHandler := auth.NewHandler(authService, cfg.App.Env)
	adminHandler := admin.NewHandler(adminService)
	subscriptionHandler := subscription.NewHandler(subscriptionService)
	profileHandler := profile.NewHandler(profileService)
	locationHandler := location.NewHandler(locationService)
	discoveryHandler := discovery.NewHandler(discoveryService)
	matchHandler := appmatch.NewHandler(matchService)
	chatHandler := chat.NewHandler(chatService, chatHub, authService)
	chatHandler.SetRestrictionChecker(restrictionService)
	adminService.SetUserSocketDisconnecter(chatHub)
	notificationHandler := notification.NewHandler(notificationService)
	safetyHandler := safety.NewHandler(safetyService)
	mediaHandler := media.NewHandler(mediaService, cfg.Media)
	healthHandler := health.NewHandler(db, redisClient)

	engine := router.New(router.Dependencies{
		Config:              cfg,
		HealthHandler:       healthHandler,
		AuthHandler:         authHandler,
		AuthService:         authService,
		AdminHandler:        adminHandler,
		AdminService:        adminService,
		ProfileHandler:      profileHandler,
		MediaHandler:        mediaHandler,
		MatchHandler:        matchHandler,
		ChatHandler:         chatHandler,
		LocationHandler:     locationHandler,
		DiscoveryHandler:    discoveryHandler,
		NotificationHandler: notificationHandler,
		SafetyHandler:       safetyHandler,
		SubscriptionHandler: subscriptionHandler,
	})

	server := &http.Server{
		Addr:              ":" + cfg.App.Port,
		Handler:           engine,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		log.Printf("%s API listening on :%s", cfg.App.Name, cfg.App.Port)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("api server: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("shutdown api server: %v", err)
	}
}

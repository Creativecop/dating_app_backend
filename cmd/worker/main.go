package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/hibiken/asynq"

	"github.com/neoscoder/aura-backend/internal/config"
	"github.com/neoscoder/aura-backend/internal/database"
	"github.com/neoscoder/aura-backend/internal/media"
	"github.com/neoscoder/aura-backend/internal/notification"
	"github.com/neoscoder/aura-backend/internal/otp"
	"github.com/neoscoder/aura-backend/internal/otp/provider"
	"github.com/neoscoder/aura-backend/internal/profile"
	"github.com/neoscoder/aura-backend/internal/queue"
	"github.com/neoscoder/aura-backend/internal/storage"
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

	storageProvider, err := storage.NewLocalProvider(cfg.Media.LocalRoot)
	if err != nil {
		log.Fatalf("create storage provider: %v", err)
	}

	server := queue.NewServer(cfg.Redis)
	mux := asynq.NewServeMux()
	log.Printf("[OTP] worker_config otp_provider=%s whatsapp_enabled=%t whatsapp_template=%s", cfg.OTP.Provider, cfg.WhatsApp.Enabled, cfg.WhatsApp.TemplateName)
	otpProcessor := otp.NewProcessor(db, provider.NewRegistry(cfg))
	otpProcessor.Register(mux)
	profileService := profile.NewService(db, cfg.Discovery)
	mediaProcessor := media.NewProcessor(db, storageProvider, cfg.Media, nil)
	mediaProcessor.SetDiscoveryEligibilityRefresher(profileService)
	mediaProcessor.Register(mux)
	queueClient := queue.NewClient(cfg.Redis)
	defer queueClient.Close()
	notificationService := notification.NewService(db, queueClient, notification.NewConfigAdapter(
		cfg.Notification.PushEnabled,
		cfg.Notification.Provider,
		cfg.Notification.PushMaxRetry,
		cfg.Notification.PushTimeoutSeconds,
		cfg.Notification.PushGraceSeconds,
		cfg.Notification.DefaultTimezone,
	))
	pushProvider, err := notificationProvider(ctx, cfg.Notification)
	if err != nil {
		log.Fatalf("create notification provider: %v", err)
	}
	notificationService.SetProvider(pushProvider)
	notification.NewProcessor(notificationService).Register(mux)

	go func() {
		log.Printf("%s worker started", cfg.App.Name)
		if err := server.Run(mux); err != nil {
			log.Fatalf("worker server: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	server.Shutdown()
}

func notificationProvider(ctx context.Context, cfg config.NotificationConfig) (notification.Provider, error) {
	switch cfg.Provider {
	case "fcm":
		return notification.NewFCMProvider(ctx, cfg)
	default:
		return notification.NewNoopProvider(), nil
	}
}

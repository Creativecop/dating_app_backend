package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	App          AppConfig
	Database     DatabaseConfig
	Redis        RedisConfig
	JWT          JWTConfig
	AdminJWT     JWTConfig
	OTP          OTPConfig
	WhatsApp     WhatsAppConfig
	Email        EmailConfig
	Media        MediaConfig
	Discovery    DiscoveryConfig
	Notification NotificationConfig
	CORS         CORSConfig
}

type AppConfig struct {
	Name string
	Env  string
	Port string
}

type DatabaseConfig struct {
	URL      string
	Host     string
	Port     string
	User     string
	Password string
	Name     string
	SSLMode  string
}

type RedisConfig struct {
	Host     string
	Port     string
	Password string
	DB       int
}

type JWTConfig struct {
	Secret              string
	AccessExpireMinutes int
	RefreshExpireDays   int
	Issuer              string
	Audience            string
}

type OTPConfig struct {
	Secret                string
	Length                int
	ExpireMinutes         int
	ResendCooldownSeconds int
	MaxAttempts           int
	MaxPerIdentifierHour  int
	MaxPerIPHour          int
	Provider              string
}

type WhatsAppConfig struct {
	Enabled       bool
	Provider      string
	PhoneNumberID string
	AccessToken   string
	TemplateName  string
	LanguageCode  string
}

type EmailConfig struct {
	Enabled      bool
	Provider     string
	SMTPHost     string
	SMTPPort     int
	SMTPUsername string
	SMTPPassword string
	FromEmail    string
	FromName     string
}

type CORSConfig struct {
	AllowedOrigins []string
}

type MediaConfig struct {
	StorageProvider               string
	BaseURL                       string
	LocalRoot                     string
	ServeMode                     string
	NginxAccelPrefix              string
	AutoApprove                   bool
	MaxMultipartMemoryMB          int
	MaxProfilePhotos              int
	MaxPhotoSizeMB                int
	MaxVideoSizeMB                int
	MaxIntroVideoSeconds          int
	VideoDurationToleranceSeconds float64
	FFProbePath                   string
	FFmpegPath                    string
}

type DiscoveryConfig struct {
	LocationMaxAgeDays int
}

type NotificationConfig struct {
	PushEnabled             bool
	Provider                string
	FirebaseProjectID       string
	FirebaseCredentialsFile string
	PushMaxRetry            int
	PushTimeoutSeconds      int
	PushGraceSeconds        int
	DefaultTimezone         string
}

func Load() (*Config, error) {
	_ = godotenv.Load()

	cfg := &Config{
		App: AppConfig{
			Name: env("APP_NAME", "Aura"),
			Env:  env("APP_ENV", "development"),
			Port: env("APP_PORT", "8080"),
		},
		Database: DatabaseConfig{
			URL:      env("DATABASE_URL", ""),
			Host:     env("DB_HOST", "localhost"),
			Port:     env("DB_PORT", "5433"),
			User:     env("DB_USER", "aura_user"),
			Password: env("DB_PASSWORD", "aura_password"),
			Name:     env("DB_NAME", "aura_db"),
			SSLMode:  env("DB_SSLMODE", "disable"),
		},
		Redis: RedisConfig{
			Host:     env("REDIS_HOST", "localhost"),
			Port:     env("REDIS_PORT", "6379"),
			Password: env("REDIS_PASSWORD", ""),
			DB:       envInt("REDIS_DB", 0),
		},
		JWT: JWTConfig{
			Secret:              env("JWT_SECRET", "change_this_secret"),
			AccessExpireMinutes: envInt("JWT_ACCESS_EXPIRE_MINUTES", 15),
			RefreshExpireDays:   envInt("JWT_REFRESH_EXPIRE_DAYS", 30),
			Issuer:              env("JWT_ISSUER", "aura-api"),
			Audience:            env("JWT_AUDIENCE", "aura-mobile"),
		},
		AdminJWT: JWTConfig{
			Secret:              env("ADMIN_JWT_SECRET", "change_this_admin_secret"),
			AccessExpireMinutes: envInt("ADMIN_JWT_ACCESS_EXPIRE_MINUTES", 15),
			RefreshExpireDays:   envInt("ADMIN_JWT_REFRESH_EXPIRE_DAYS", 7),
			Issuer:              env("ADMIN_JWT_ISSUER", "aura-admin-api"),
			Audience:            env("ADMIN_JWT_AUDIENCE", "aura-admin-panel"),
		},
		OTP: OTPConfig{
			Secret:                env("OTP_SECRET", "change_this_otp_secret"),
			Length:                envInt("OTP_LENGTH", 6),
			ExpireMinutes:         envInt("OTP_EXPIRE_MINUTES", 5),
			ResendCooldownSeconds: envInt("OTP_RESEND_COOLDOWN_SECONDS", 60),
			MaxAttempts:           envInt("OTP_MAX_ATTEMPTS", 5),
			MaxPerIdentifierHour:  envInt("OTP_MAX_PER_IDENTIFIER_HOUR", 5),
			MaxPerIPHour:          envInt("OTP_MAX_PER_IP_HOUR", 20),
			Provider:              strings.ToLower(env("OTP_PROVIDER", "noop")),
		},
		WhatsApp: WhatsAppConfig{
			Enabled:       envBool("WHATSAPP_ENABLED", false),
			Provider:      env("WHATSAPP_PROVIDER", "meta"),
			PhoneNumberID: env("WHATSAPP_PHONE_NUMBER_ID", ""),
			AccessToken:   env("WHATSAPP_ACCESS_TOKEN", ""),
			TemplateName:  env("WHATSAPP_TEMPLATE_NAME", "auth_code"),
			LanguageCode:  env("WHATSAPP_LANGUAGE_CODE", "en_US"),
		},
		Email: EmailConfig{
			Enabled:      envBool("EMAIL_ENABLED", false),
			Provider:     env("EMAIL_PROVIDER", "smtp"),
			SMTPHost:     env("SMTP_HOST", ""),
			SMTPPort:     envInt("SMTP_PORT", 587),
			SMTPUsername: env("SMTP_USERNAME", ""),
			SMTPPassword: env("SMTP_PASSWORD", ""),
			FromEmail:    env("SMTP_FROM_EMAIL", "no-reply@aura.com"),
			FromName:     env("SMTP_FROM_NAME", "Aura"),
		},
		Media: MediaConfig{
			StorageProvider:               strings.ToLower(env("MEDIA_STORAGE_PROVIDER", "local")),
			BaseURL:                       strings.TrimRight(env("MEDIA_BASE_URL", "http://localhost:8080/api/v1/media"), "/"),
			LocalRoot:                     env("MEDIA_LOCAL_ROOT", "./storage/uploads"),
			ServeMode:                     strings.ToLower(env("MEDIA_SERVE_MODE", "local")),
			NginxAccelPrefix:              strings.TrimRight(env("MEDIA_NGINX_ACCEL_PREFIX", "/protected-media"), "/"),
			AutoApprove:                   envBool("MEDIA_AUTO_APPROVE", true),
			MaxMultipartMemoryMB:          envInt("MEDIA_MAX_MULTIPART_MEMORY_MB", 16),
			MaxProfilePhotos:              envInt("MEDIA_MAX_PROFILE_PHOTOS", 6),
			MaxPhotoSizeMB:                envInt("MEDIA_MAX_PHOTO_SIZE_MB", 10),
			MaxVideoSizeMB:                envInt("MEDIA_MAX_VIDEO_SIZE_MB", 100),
			MaxIntroVideoSeconds:          envInt("MEDIA_MAX_INTRO_VIDEO_SECONDS", 30),
			VideoDurationToleranceSeconds: envFloat("MEDIA_VIDEO_DURATION_TOLERANCE_SECONDS", 0.5),
			FFProbePath:                   env("MEDIA_FFPROBE_PATH", "ffprobe"),
			FFmpegPath:                    env("MEDIA_FFMPEG_PATH", "ffmpeg"),
		},
		Discovery: DiscoveryConfig{
			LocationMaxAgeDays: envInt("DISCOVERY_LOCATION_MAX_AGE_DAYS", 30),
		},
		Notification: NotificationConfig{
			PushEnabled:             envBool("NOTIFICATION_PUSH_ENABLED", true),
			Provider:                strings.ToLower(env("NOTIFICATION_PROVIDER", "noop")),
			FirebaseProjectID:       env("FIREBASE_PROJECT_ID", ""),
			FirebaseCredentialsFile: env("FIREBASE_CREDENTIALS_FILE", "./firebase-service-account.json"),
			PushMaxRetry:            envInt("PUSH_MAX_RETRY", 3),
			PushTimeoutSeconds:      envInt("PUSH_TIMEOUT_SECONDS", 15),
			PushGraceSeconds:        envInt("PUSH_GRACE_SECONDS", 3),
			DefaultTimezone:         env("NOTIFICATION_DEFAULT_TIMEZONE", "Asia/Dhaka"),
		},
		CORS: CORSConfig{
			AllowedOrigins: splitCSV(env("CORS_ALLOWED_ORIGINS", "*")),
		},
	}

	if cfg.OTP.Length < 4 || cfg.OTP.Length > 10 {
		return nil, fmt.Errorf("OTP_LENGTH must be between 4 and 10")
	}
	if cfg.Discovery.LocationMaxAgeDays < 1 {
		return nil, fmt.Errorf("DISCOVERY_LOCATION_MAX_AGE_DAYS must be at least 1")
	}
	if cfg.Notification.PushMaxRetry < 0 {
		return nil, fmt.Errorf("PUSH_MAX_RETRY cannot be negative")
	}
	if cfg.Notification.PushTimeoutSeconds < 1 {
		return nil, fmt.Errorf("PUSH_TIMEOUT_SECONDS must be at least 1")
	}
	if cfg.Notification.PushGraceSeconds < 0 {
		return nil, fmt.Errorf("PUSH_GRACE_SECONDS cannot be negative")
	}
	if cfg.Notification.Provider != "noop" && cfg.Notification.Provider != "fcm" {
		return nil, fmt.Errorf("NOTIFICATION_PROVIDER must be noop or fcm")
	}
	if _, err := time.LoadLocation(cfg.Notification.DefaultTimezone); err != nil {
		return nil, fmt.Errorf("NOTIFICATION_DEFAULT_TIMEZONE is invalid: %w", err)
	}

	return cfg, nil
}

func (c DatabaseConfig) DSN() string {
	if c.URL != "" {
		return c.URL
	}
	return fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		c.Host,
		c.Port,
		c.User,
		c.Password,
		c.Name,
		c.SSLMode,
	)
}

func (c RedisConfig) Addr() string {
	return c.Host + ":" + c.Port
}

func (c JWTConfig) AccessTTL() time.Duration {
	return time.Duration(c.AccessExpireMinutes) * time.Minute
}

func (c JWTConfig) RefreshTTL() time.Duration {
	return time.Duration(c.RefreshExpireDays) * 24 * time.Hour
}

func (c OTPConfig) ExpiryTTL() time.Duration {
	return time.Duration(c.ExpireMinutes) * time.Minute
}

func (c OTPConfig) ResendCooldownTTL() time.Duration {
	return time.Duration(c.ResendCooldownSeconds) * time.Second
}

func env(key, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}

func envInt(key string, fallback int) int {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}
	return value
}

func envFloat(key string, fallback float64) float64 {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback
	}
	value, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return fallback
	}
	return value
}

func envBool(key string, fallback bool) bool {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback
	}
	value, err := strconv.ParseBool(raw)
	if err != nil {
		return fallback
	}
	return value
}

func splitCSV(raw string) []string {
	parts := strings.Split(raw, ",")
	values := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			values = append(values, part)
		}
	}
	if len(values) == 0 {
		return []string{"*"}
	}
	return values
}

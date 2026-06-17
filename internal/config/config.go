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
	RateLimit    RateLimitConfig
	Request      RequestConfig
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
	DevBypassEnabled      bool
	DevBypassCode         string
}

type WhatsAppConfig struct {
	Enabled           bool
	Provider          string
	BusinessAccountID string
	PhoneNumberID     string
	AccessToken       string
	TemplateName      string
	LanguageCode      string
	GraphAPIVersion   string
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

type RateLimitConfig struct {
	Enabled              bool
	RedisRequiredForAuth bool

	AdminLoginEmail15M int
	AdminLoginIP15M    int
	AdminLoginIP1H     int

	OTPRequestIdentifier10M int
	OTPRequestIdentifier1H  int
	OTPRequestIP1H          int
	OTPVerifyIdentifier10M  int
	OTPVerifyIP1H           int

	RefreshSubject1M int
	RefreshIP1M      int

	ReportCreateUser1M int
	ReportCreateUser1H int
	ReportCreateIP1H   int

	AdminReview1M              int
	AdminReview1H              int
	AdminRestrictionMutation1M int
	AdminRestrictionMutation1H int
	AdminIdentityMutation10M   int
	AdminIdentityMutation1H    int
	AdminReadAdmin1M           int
	AdminReadIP1M              int

	SubscriptionSubmitUser10M int
	SubscriptionSubmitUser1H  int
	SubscriptionSubmitIP1H    int
	SubscriptionReviewAdmin1M int
	SubscriptionReviewAdmin1H int

	SocketConnectUser1M int
	SocketConnectIP1M   int
}

type RequestConfig struct {
	JSONBodyLimitBytes int64
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
			MaxPerIdentifierHour:  envInt("OTP_MAX_PER_IDENTIFIER_HOUR", 10),
			MaxPerIPHour:          envInt("OTP_MAX_PER_IP_HOUR", 30),
			Provider:              strings.ToLower(env("OTP_PROVIDER", "noop")),
			DevBypassEnabled:      envBool("OTP_DEV_BYPASS_ENABLED", false),
			DevBypassCode:         env("OTP_DEV_BYPASS_CODE", "123456"),
		},
		WhatsApp: WhatsAppConfig{
			Enabled:           envBool("WHATSAPP_ENABLED", false),
			Provider:          env("WHATSAPP_PROVIDER", "meta"),
			BusinessAccountID: env("WHATSAPP_BUSINESS_ACCOUNT_ID", ""),
			PhoneNumberID:     env("WHATSAPP_PHONE_NUMBER_ID", ""),
			AccessToken:       env("WHATSAPP_ACCESS_TOKEN", ""),
			TemplateName:      env("WHATSAPP_TEMPLATE_NAME", "auth_code"),
			LanguageCode:      env("WHATSAPP_LANGUAGE_CODE", "en_US"),
			GraphAPIVersion:   env("WHATSAPP_GRAPH_API_VERSION", "v25.0"),
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
		RateLimit: RateLimitConfig{
			Enabled:              envBool("RATE_LIMIT_ENABLED", true),
			RedisRequiredForAuth: envBool("RATE_LIMIT_REDIS_REQUIRED_FOR_AUTH", true),

			AdminLoginEmail15M: envInt("RATE_LIMIT_ADMIN_LOGIN_EMAIL_15M", 5),
			AdminLoginIP15M:    envInt("RATE_LIMIT_ADMIN_LOGIN_IP_15M", 10),
			AdminLoginIP1H:     envInt("RATE_LIMIT_ADMIN_LOGIN_IP_1H", 30),

			OTPRequestIdentifier10M: envInt("RATE_LIMIT_OTP_REQUEST_IDENTIFIER_10M", 3),
			OTPRequestIdentifier1H:  envInt("RATE_LIMIT_OTP_REQUEST_IDENTIFIER_1H", 10),
			OTPRequestIP1H:          envInt("RATE_LIMIT_OTP_REQUEST_IP_1H", 30),
			OTPVerifyIdentifier10M:  envInt("RATE_LIMIT_OTP_VERIFY_IDENTIFIER_10M", 5),
			OTPVerifyIP1H:           envInt("RATE_LIMIT_OTP_VERIFY_IP_1H", 20),

			RefreshSubject1M: envInt("RATE_LIMIT_REFRESH_SUBJECT_1M", 30),
			RefreshIP1M:      envInt("RATE_LIMIT_REFRESH_IP_1M", 120),

			ReportCreateUser1M: envInt("RATE_LIMIT_REPORT_CREATE_USER_1M", 5),
			ReportCreateUser1H: envInt("RATE_LIMIT_REPORT_CREATE_USER_1H", 20),
			ReportCreateIP1H:   envInt("RATE_LIMIT_REPORT_CREATE_IP_1H", 60),

			AdminReview1M:              envInt("RATE_LIMIT_ADMIN_REVIEW_1M", 30),
			AdminReview1H:              envInt("RATE_LIMIT_ADMIN_REVIEW_1H", 300),
			AdminRestrictionMutation1M: envInt("RATE_LIMIT_ADMIN_RESTRICTION_MUTATION_1M", 20),
			AdminRestrictionMutation1H: envInt("RATE_LIMIT_ADMIN_RESTRICTION_MUTATION_1H", 100),
			AdminIdentityMutation10M:   envInt("RATE_LIMIT_ADMIN_IDENTITY_MUTATION_10M", 10),
			AdminIdentityMutation1H:    envInt("RATE_LIMIT_ADMIN_IDENTITY_MUTATION_1H", 30),
			AdminReadAdmin1M:           envInt("RATE_LIMIT_ADMIN_READ_ADMIN_1M", 120),
			AdminReadIP1M:              envInt("RATE_LIMIT_ADMIN_READ_IP_1M", 600),

			SubscriptionSubmitUser10M: envInt("RATE_LIMIT_SUBSCRIPTION_SUBMIT_USER_10M", 3),
			SubscriptionSubmitUser1H:  envInt("RATE_LIMIT_SUBSCRIPTION_SUBMIT_USER_1H", 10),
			SubscriptionSubmitIP1H:    envInt("RATE_LIMIT_SUBSCRIPTION_SUBMIT_IP_1H", 30),
			SubscriptionReviewAdmin1M: envInt("RATE_LIMIT_SUBSCRIPTION_REVIEW_ADMIN_1M", 30),
			SubscriptionReviewAdmin1H: envInt("RATE_LIMIT_SUBSCRIPTION_REVIEW_ADMIN_1H", 300),

			SocketConnectUser1M: envInt("RATE_LIMIT_SOCKET_CONNECT_USER_1M", 20),
			SocketConnectIP1M:   envInt("RATE_LIMIT_SOCKET_CONNECT_IP_1M", 60),
		},
		Request: RequestConfig{
			JSONBodyLimitBytes: int64(envInt("REQUEST_JSON_BODY_LIMIT_BYTES", 1048576)),
		},
	}

	if cfg.OTP.Length < 4 || cfg.OTP.Length > 10 {
		return nil, fmt.Errorf("OTP_LENGTH must be between 4 and 10")
	}
	if cfg.App.Env == "production" && cfg.OTP.DevBypassEnabled {
		return nil, fmt.Errorf("OTP_DEV_BYPASS_ENABLED cannot be true in production")
	}
	if cfg.App.Env == "production" {
		if !databaseConfiguredForProduction() {
			return nil, fmt.Errorf("DATABASE_URL or database connection settings are required in production")
		}
		if !redisConfiguredForProduction() {
			return nil, fmt.Errorf("REDIS_HOST and REDIS_PORT are required in production")
		}
		if cfg.CORS.allowsWildcard() {
			return nil, fmt.Errorf("CORS_ALLOWED_ORIGINS cannot be * in production")
		}
		if err := validateProductionSecret("JWT_SECRET", cfg.JWT.Secret, "change_this_secret"); err != nil {
			return nil, err
		}
		if err := validateProductionSecret("ADMIN_JWT_SECRET", cfg.AdminJWT.Secret, "change_this_admin_secret"); err != nil {
			return nil, err
		}
		if err := validateProductionSecret("OTP_SECRET", cfg.OTP.Secret, "change_this_otp_secret"); err != nil {
			return nil, err
		}
		if cfg.JWT.Secret == cfg.AdminJWT.Secret || cfg.JWT.Secret == cfg.OTP.Secret || cfg.AdminJWT.Secret == cfg.OTP.Secret {
			return nil, fmt.Errorf("JWT_SECRET, ADMIN_JWT_SECRET, and OTP_SECRET must be distinct in production")
		}
		if !cfg.RateLimit.Enabled {
			return nil, fmt.Errorf("RATE_LIMIT_ENABLED must be true in production")
		}
	}
	if cfg.Request.JSONBodyLimitBytes < 1024 {
		return nil, fmt.Errorf("REQUEST_JSON_BODY_LIMIT_BYTES must be at least 1024")
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

func (c CORSConfig) allowsWildcard() bool {
	for _, origin := range c.AllowedOrigins {
		if strings.TrimSpace(origin) == "*" {
			return true
		}
	}
	return false
}

func validateProductionSecret(name string, value string, placeholder string) error {
	value = strings.TrimSpace(value)
	if value == "" || value == placeholder || strings.HasPrefix(value, "change_this") {
		return fmt.Errorf("%s must be set to a strong production secret", name)
	}
	if len(value) < 32 {
		return fmt.Errorf("%s must be at least 32 characters in production", name)
	}
	return nil
}

func databaseConfiguredForProduction() bool {
	if strings.TrimSpace(os.Getenv("DATABASE_URL")) != "" {
		return true
	}
	required := []string{"DB_HOST", "DB_PORT", "DB_USER", "DB_PASSWORD", "DB_NAME"}
	for _, key := range required {
		if strings.TrimSpace(os.Getenv(key)) == "" {
			return false
		}
	}
	return true
}

func redisConfiguredForProduction() bool {
	return strings.TrimSpace(os.Getenv("REDIS_HOST")) != "" && strings.TrimSpace(os.Getenv("REDIS_PORT")) != ""
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

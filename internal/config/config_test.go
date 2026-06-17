package config

import (
	"strings"
	"testing"
)

const strongSecret = "12345678901234567890123456789012"

func setValidProductionEnv(t *testing.T) {
	t.Helper()
	t.Setenv("APP_ENV", "production")
	t.Setenv("DATABASE_URL", "postgres://user:pass@db:5432/aura?sslmode=require")
	t.Setenv("DB_HOST", "")
	t.Setenv("DB_PORT", "")
	t.Setenv("DB_USER", "")
	t.Setenv("DB_PASSWORD", "")
	t.Setenv("DB_NAME", "")
	t.Setenv("REDIS_HOST", "redis")
	t.Setenv("REDIS_PORT", "6379")
	t.Setenv("JWT_SECRET", strongSecret)
	t.Setenv("ADMIN_JWT_SECRET", strongSecret+"admin")
	t.Setenv("OTP_SECRET", strongSecret+"otp")
	t.Setenv("CORS_ALLOWED_ORIGINS", "https://app.example.com,https://admin.example.com")
	t.Setenv("RATE_LIMIT_ENABLED", "true")
	t.Setenv("OTP_DEV_BYPASS_ENABLED", "false")
	t.Setenv("REQUEST_JSON_BODY_LIMIT_BYTES", "1048576")
}

func TestLoadProductionRejectsWildcardCORS(t *testing.T) {
	setValidProductionEnv(t)
	t.Setenv("CORS_ALLOWED_ORIGINS", "*")

	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), "CORS_ALLOWED_ORIGINS") {
		t.Fatalf("expected CORS production error, got %v", err)
	}
}

func TestLoadProductionRejectsShortJWTSecret(t *testing.T) {
	setValidProductionEnv(t)
	t.Setenv("JWT_SECRET", "short")

	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), "JWT_SECRET") {
		t.Fatalf("expected JWT_SECRET production error, got %v", err)
	}
}

func TestLoadProductionRejectsRepeatedSecrets(t *testing.T) {
	setValidProductionEnv(t)
	t.Setenv("ADMIN_JWT_SECRET", strongSecret)

	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), "must be distinct") {
		t.Fatalf("expected distinct secret production error, got %v", err)
	}
}

func TestLoadProductionRejectsMissingDatabaseConfig(t *testing.T) {
	setValidProductionEnv(t)
	t.Setenv("DATABASE_URL", "")

	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), "DATABASE_URL") {
		t.Fatalf("expected database production error, got %v", err)
	}
}

func TestLoadProductionRejectsMissingRedisConfig(t *testing.T) {
	setValidProductionEnv(t)
	t.Setenv("REDIS_HOST", "")

	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), "REDIS_HOST") {
		t.Fatalf("expected Redis production error, got %v", err)
	}
}

func TestLoadProductionRejectsDisabledRateLimit(t *testing.T) {
	setValidProductionEnv(t)
	t.Setenv("RATE_LIMIT_ENABLED", "false")

	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), "RATE_LIMIT_ENABLED") {
		t.Fatalf("expected rate limit production error, got %v", err)
	}
}

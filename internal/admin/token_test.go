package admin

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	authpkg "github.com/neoscoder/aura-backend/internal/auth"
	"github.com/neoscoder/aura-backend/internal/config"
)

func TestAdminAccessTokenIncludesTokenVersion(t *testing.T) {
	service := NewTokenService(config.JWTConfig{
		Secret:              "test-secret",
		AccessExpireMinutes: 15,
		Issuer:              "admin-test",
		Audience:            "admin-panel",
	})
	raw, err := service.GenerateAccessToken(AdminUser{
		UUID:         uuid.New(),
		Email:        "admin@example.com",
		TokenVersion: 3,
	}, AdminSession{UUID: uuid.New()}, []string{RoleSuperAdmin})
	if err != nil {
		t.Fatalf("GenerateAccessToken returned error: %v", err)
	}
	claims, err := service.ParseAccessToken(raw)
	if err != nil {
		t.Fatalf("ParseAccessToken returned error: %v", err)
	}
	if claims.TokenVersion != 3 {
		t.Fatalf("TokenVersion = %d, want 3", claims.TokenVersion)
	}
	if claims.TokenType != "admin_access" {
		t.Fatalf("TokenType = %q, want admin_access", claims.TokenType)
	}
	if len(claims.Roles) != 1 || claims.Roles[0] != RoleSuperAdmin {
		t.Fatalf("Roles = %#v, want SUPER_ADMIN", claims.Roles)
	}
}

func TestAdminTokenStateValidRejectsOldIssuedAt(t *testing.T) {
	changedAt := time.Now().UTC()
	issuedAt := jwt.NewNumericDate(changedAt.Add(-time.Second))

	if adminTokenStateValid(AdminUser{TokenVersion: 1, PasswordChangedAt: &changedAt}, issuedAt) {
		t.Fatal("old issuedAt was accepted")
	}
}

func TestAdminRefreshSessionStateValidRejectsOldSession(t *testing.T) {
	changedAt := time.Now().UTC()

	if adminRefreshSessionStateValid(AdminUser{TokenVersion: 1, PasswordChangedAt: &changedAt}, changedAt.Add(-time.Second)) {
		t.Fatal("old refresh session was accepted")
	}
}

func TestAdminAndMobileAccessTokensAreSeparated(t *testing.T) {
	adminTokenService := NewTokenService(config.JWTConfig{
		Secret:              "admin-secret-for-token-separation",
		AccessExpireMinutes: 15,
		Issuer:              "admin-test",
		Audience:            "admin-panel",
	})
	mobileTokenService := authpkg.NewTokenService(config.JWTConfig{
		Secret:              "mobile-secret-for-token-separation",
		AccessExpireMinutes: 15,
		Issuer:              "mobile-test",
		Audience:            "mobile-app",
	})

	adminRaw, err := adminTokenService.GenerateAccessToken(AdminUser{
		UUID:         uuid.New(),
		Email:        "admin@example.com",
		TokenVersion: 1,
	}, AdminSession{UUID: uuid.New()}, []string{RoleSuperAdmin})
	if err != nil {
		t.Fatalf("GenerateAccessToken(admin) returned error: %v", err)
	}
	mobileRaw, err := mobileTokenService.GenerateAccessToken(authpkg.User{
		ID:               1,
		UUID:             uuid.New(),
		Status:           authpkg.UserStatusActive,
		OnboardingStatus: authpkg.OnboardingCompleted,
	}, authpkg.UserSession{ID: 10, UUID: uuid.New(), UserID: 1})
	if err != nil {
		t.Fatalf("GenerateAccessToken(mobile) returned error: %v", err)
	}

	if _, err := adminTokenService.ParseAccessToken(mobileRaw); err == nil {
		t.Fatal("admin token parser accepted mobile access token")
	}
	if _, err := mobileTokenService.ParseAccessToken(adminRaw); err == nil {
		t.Fatal("mobile token parser accepted admin access token")
	}
}

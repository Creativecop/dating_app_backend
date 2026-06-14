package auth

import (
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/neoscoder/aura-backend/internal/config"
)

func TestRefreshTokenHashIsStableAndOneWay(t *testing.T) {
	token, err := GenerateRefreshToken()
	if err != nil {
		t.Fatalf("GenerateRefreshToken returned error: %v", err)
	}
	if token == "" {
		t.Fatal("expected refresh token")
	}

	hashA := HashRefreshToken(token)
	hashB := HashRefreshToken(token)
	if hashA != hashB {
		t.Fatal("expected stable refresh token hash")
	}
	if hashA == token {
		t.Fatal("hash must not equal raw refresh token")
	}
	if HashRefreshToken(token+"x") == hashA {
		t.Fatal("different token should produce different hash")
	}
}

func TestAccessTokenRoundTrip(t *testing.T) {
	now := time.Now().UTC()
	tokenService := NewTokenService(config.JWTConfig{
		Secret:              "test-secret",
		AccessExpireMinutes: 15,
		RefreshExpireDays:   30,
		Issuer:              "aura-api",
		Audience:            "aura-mobile",
	})

	user := User{
		ID:               1,
		UUID:             uuid.New(),
		Status:           UserStatusActive,
		OnboardingStatus: OnboardingProfileRequired,
		PhoneVerifiedAt:  &now,
	}
	session := UserSession{
		ID:     10,
		UUID:   uuid.New(),
		UserID: user.ID,
	}

	raw, err := tokenService.GenerateAccessToken(user, session)
	if err != nil {
		t.Fatalf("GenerateAccessToken returned error: %v", err)
	}

	claims, err := tokenService.ParseAccessToken(raw)
	if err != nil {
		t.Fatalf("ParseAccessToken returned error: %v", err)
	}
	if claims.Subject != user.UUID.String() {
		t.Fatalf("unexpected subject: %s", claims.Subject)
	}
	if claims.SessionID != session.UUID.String() {
		t.Fatalf("unexpected session id: %s", claims.SessionID)
	}
	if !claims.PhoneVerified {
		t.Fatal("expected phoneVerified claim")
	}
	if claims.ProfileComplete {
		t.Fatal("profile should not be complete")
	}
}

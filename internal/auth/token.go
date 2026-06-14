package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"github.com/neoscoder/aura-backend/internal/config"
)

type TokenService struct {
	cfg config.JWTConfig
}

type AccessClaims struct {
	SessionID       string `json:"sid"`
	PhoneVerified   bool   `json:"phoneVerified"`
	EmailVerified   bool   `json:"emailVerified"`
	ProfileComplete bool   `json:"profileCompleted"`
	jwt.RegisteredClaims
}

func NewTokenService(cfg config.JWTConfig) *TokenService {
	return &TokenService{cfg: cfg}
}

func (s *TokenService) GenerateAccessToken(user User, session UserSession) (string, error) {
	now := time.Now().UTC()
	claims := AccessClaims{
		SessionID:       session.UUID.String(),
		PhoneVerified:   user.PhoneVerifiedAt != nil,
		EmailVerified:   user.EmailVerifiedAt != nil,
		ProfileComplete: user.OnboardingStatus == OnboardingCompleted,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   user.UUID.String(),
			ID:        uuid.NewString(),
			Issuer:    s.cfg.Issuer,
			Audience:  jwt.ClaimStrings{s.cfg.Audience},
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(s.cfg.AccessTTL())),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.cfg.Secret))
}

func (s *TokenService) ParseAccessToken(raw string) (*AccessClaims, error) {
	claims := &AccessClaims{}
	token, err := jwt.ParseWithClaims(
		raw,
		claims,
		func(token *jwt.Token) (any, error) {
			if token.Method.Alg() != jwt.SigningMethodHS256.Alg() {
				return nil, fmt.Errorf("unexpected jwt signing method")
			}
			return []byte(s.cfg.Secret), nil
		},
		jwt.WithIssuer(s.cfg.Issuer),
		jwt.WithAudience(s.cfg.Audience),
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
	)
	if err != nil {
		return nil, err
	}
	if !token.Valid {
		return nil, fmt.Errorf("invalid access token")
	}
	return claims, nil
}

func GenerateRefreshToken() (string, error) {
	bytes := make([]byte, 64)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("generate refresh token: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(bytes), nil
}

func HashRefreshToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

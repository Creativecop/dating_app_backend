package auth

import "github.com/google/uuid"

const ContextUserKey = "auth_user"

type RequestOTPRequest struct {
	Channel  string  `json:"channel" binding:"required"`
	Phone    *string `json:"phone"`
	Email    *string `json:"email"`
	Purpose  string  `json:"purpose"`
	DeviceID string  `json:"deviceId"`
}

type VerifyOTPRequest struct {
	Channel    string  `json:"channel" binding:"required"`
	Phone      *string `json:"phone"`
	Email      *string `json:"email"`
	Purpose    string  `json:"purpose"`
	Code       string  `json:"code" binding:"required"`
	DeviceID   string  `json:"deviceId"`
	DeviceName string  `json:"deviceName"`
	Platform   string  `json:"platform"`
	FCMToken   string  `json:"fcmToken"`
	AppVersion string  `json:"appVersion"`
	OSVersion  string  `json:"osVersion"`
}

type RefreshTokenRequest struct {
	RefreshToken string `json:"refreshToken" binding:"required"`
	DeviceID     string `json:"deviceId"`
}

type LogoutRequest struct {
	RefreshToken string `json:"refreshToken"`
	DeviceID     string `json:"deviceId"`
}

type AuthenticatedUser struct {
	UserID      uint64
	UserUUID    uuid.UUID
	SessionID   uint64
	SessionUUID uuid.UUID
}

type AuthResponse struct {
	AccessToken      string       `json:"accessToken"`
	RefreshToken     string       `json:"refreshToken"`
	ExpiresIn        int64        `json:"expiresIn"`
	RefreshExpiresIn int64        `json:"refreshExpiresIn"`
	User             UserResponse `json:"user"`
}

type TokenResponse struct {
	AccessToken      string `json:"accessToken"`
	RefreshToken     string `json:"refreshToken"`
	ExpiresIn        int64  `json:"expiresIn"`
	RefreshExpiresIn int64  `json:"refreshExpiresIn"`
}

type UserResponse struct {
	UUID             string  `json:"uuid"`
	Phone            *string `json:"phone"`
	Email            *string `json:"email"`
	PhoneVerified    bool    `json:"phoneVerified"`
	EmailVerified    bool    `json:"emailVerified"`
	Status           string  `json:"status"`
	OnboardingStatus string  `json:"onboardingStatus"`
}

func ToUserResponse(user User) UserResponse {
	return UserResponse{
		UUID:             user.UUID.String(),
		Phone:            user.Phone,
		Email:            user.Email,
		PhoneVerified:    user.PhoneVerifiedAt != nil,
		EmailVerified:    user.EmailVerifiedAt != nil,
		Status:           user.Status,
		OnboardingStatus: user.OnboardingStatus,
	}
}

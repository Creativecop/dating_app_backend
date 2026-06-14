package otp

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"math/big"
	"net/mail"
	"strings"
)

func GenerateNumericCode(length int) (string, error) {
	var builder strings.Builder
	builder.Grow(length)

	for i := 0; i < length; i++ {
		n, err := rand.Int(rand.Reader, big.NewInt(10))
		if err != nil {
			return "", fmt.Errorf("generate otp digit: %w", err)
		}
		builder.WriteByte(byte('0' + n.Int64()))
	}

	return builder.String(), nil
}

func NormalizeChannel(channel string) (string, error) {
	switch strings.ToUpper(strings.TrimSpace(channel)) {
	case ChannelWhatsApp:
		return ChannelWhatsApp, nil
	case ChannelEmail:
		return ChannelEmail, nil
	default:
		return "", ErrInvalidChannel
	}
}

func NormalizePurpose(purpose string) (string, error) {
	if strings.TrimSpace(purpose) == "" {
		return PurposeLogin, nil
	}

	switch strings.ToUpper(strings.TrimSpace(purpose)) {
	case PurposeLogin:
		return PurposeLogin, nil
	case PurposeRegister:
		return PurposeRegister, nil
	case PurposeVerifyPhone:
		return PurposeVerifyPhone, nil
	case PurposeVerifyEmail:
		return PurposeVerifyEmail, nil
	case PurposeReset:
		return PurposeReset, nil
	default:
		return "", ErrInvalidPurpose
	}
}

func NormalizeIdentifier(channel, phone, email string) (string, error) {
	switch channel {
	case ChannelWhatsApp:
		phone = strings.TrimSpace(phone)
		if phone == "" || !strings.HasPrefix(phone, "+") {
			return "", ErrInvalidIdentifier
		}
		return phone, nil
	case ChannelEmail:
		email = strings.ToLower(strings.TrimSpace(email))
		if email == "" {
			return "", ErrInvalidIdentifier
		}
		if _, err := mail.ParseAddress(email); err != nil {
			return "", ErrInvalidIdentifier
		}
		return email, nil
	default:
		return "", ErrInvalidChannel
	}
}

func HashIdentifier(secret, identifier string) string {
	return hmacHex(secret, identifier)
}

func HashCode(secret, code, identifier, purpose string) string {
	return hmacHex(secret, code+"|"+identifier+"|"+purpose)
}

func CompareCodeHash(expectedHash, secret, code, identifier, purpose string) bool {
	actualHash := HashCode(secret, code, identifier, purpose)
	return subtle.ConstantTimeCompare([]byte(expectedHash), []byte(actualHash)) == 1
}

func hmacHex(secret, value string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(value))
	return hex.EncodeToString(mac.Sum(nil))
}

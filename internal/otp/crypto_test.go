package otp

import (
	"crypto/sha256"
	"encoding/hex"
	"testing"
)

func TestGenerateNumericCode(t *testing.T) {
	code, err := GenerateNumericCode(6)
	if err != nil {
		t.Fatalf("GenerateNumericCode returned error: %v", err)
	}
	if len(code) != 6 {
		t.Fatalf("expected 6 digits, got %q", code)
	}
	for _, char := range code {
		if char < '0' || char > '9' {
			t.Fatalf("expected numeric code, got %q", code)
		}
	}
}

func TestHMACOTPHashIncludesIdentifierAndPurpose(t *testing.T) {
	secret := "otp-secret"
	code := "123456"
	identifier := "+8801712345678"

	loginHash := HashCode(secret, code, identifier, PurposeLogin)
	verifyHash := HashCode(secret, code, identifier, PurposeVerifyPhone)
	otherIdentifierHash := HashCode(secret, code, "+8801712345679", PurposeLogin)

	if loginHash == verifyHash {
		t.Fatal("expected purpose to change OTP hash")
	}
	if loginHash == otherIdentifierHash {
		t.Fatal("expected identifier to change OTP hash")
	}
	if !CompareCodeHash(loginHash, secret, code, identifier, PurposeLogin) {
		t.Fatal("expected matching HMAC hash to validate")
	}
	if CompareCodeHash(loginHash, secret, "000000", identifier, PurposeLogin) {
		t.Fatal("expected wrong code to fail validation")
	}

	plain := sha256.Sum256([]byte(code))
	if loginHash == hex.EncodeToString(plain[:]) {
		t.Fatal("OTP hash must not be plain SHA-256 of the code")
	}
}

func TestNormalizeIdentifier(t *testing.T) {
	phone, err := NormalizeIdentifier(ChannelWhatsApp, "+8801712345678", "")
	if err != nil {
		t.Fatalf("expected valid phone: %v", err)
	}
	if phone != "+8801712345678" {
		t.Fatalf("unexpected normalized phone: %s", phone)
	}

	email, err := NormalizeIdentifier(ChannelEmail, "", "USER@EXAMPLE.COM")
	if err != nil {
		t.Fatalf("expected valid email: %v", err)
	}
	if email != "user@example.com" {
		t.Fatalf("unexpected normalized email: %s", email)
	}

	if _, err := NormalizeIdentifier(ChannelWhatsApp, "01712345678", ""); err == nil {
		t.Fatal("expected local phone without country prefix to fail")
	}
}

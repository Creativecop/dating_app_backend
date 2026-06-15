package otp

import (
	"context"
	"testing"

	"github.com/neoscoder/aura-backend/internal/config"
)

func TestVerifyDevBypassAcceptsFixedCode(t *testing.T) {
	service := NewService(nil, nil, nil, config.OTPConfig{
		Secret:           "test-secret",
		DevBypassEnabled: true,
		DevBypassCode:    "123456",
	})

	code, err := service.Verify(context.Background(), VerifyInput{
		Channel: ChannelWhatsApp,
		Phone:   "+8801625930011",
		Purpose: PurposeLogin,
		Code:    "123456",
	})
	if err != nil {
		t.Fatalf("Verify returned error: %v", err)
	}
	if code.Channel != ChannelWhatsApp {
		t.Fatalf("channel = %q", code.Channel)
	}
	if code.Phone == nil || *code.Phone != "+8801625930011" {
		t.Fatalf("phone = %#v", code.Phone)
	}
	if code.ConsumedAt == nil {
		t.Fatal("ConsumedAt is nil")
	}
}

func TestVerifyDevBypassRejectsOtherCode(t *testing.T) {
	service := NewService(nil, nil, nil, config.OTPConfig{
		Secret:           "test-secret",
		DevBypassEnabled: true,
		DevBypassCode:    "123456",
	})

	if _, err := service.Verify(context.Background(), VerifyInput{
		Channel: ChannelWhatsApp,
		Phone:   "+8801625930011",
		Purpose: PurposeLogin,
		Code:    "000000",
	}); err == nil {
		t.Fatal("Verify returned nil error")
	}
}

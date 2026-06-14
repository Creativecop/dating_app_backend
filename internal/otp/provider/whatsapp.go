package provider

import (
	"context"
	"fmt"

	"github.com/neoscoder/aura-backend/internal/config"
)

type WhatsAppProvider struct {
	cfg config.WhatsAppConfig
}

func NewWhatsAppProvider(cfg config.WhatsAppConfig) *WhatsAppProvider {
	return &WhatsAppProvider{cfg: cfg}
}

func (p *WhatsAppProvider) SendOTP(ctx context.Context, to string, code string, purpose string) (string, error) {
	if !p.cfg.Enabled {
		return "", fmt.Errorf("whatsapp OTP provider is disabled")
	}
	if p.cfg.PhoneNumberID == "" || p.cfg.AccessToken == "" || p.cfg.TemplateName == "" {
		return "", fmt.Errorf("whatsapp provider credentials are incomplete")
	}

	select {
	case <-ctx.Done():
		return "", ctx.Err()
	default:
	}

	return "", fmt.Errorf("whatsapp provider is configured but not implemented; use OTP_PROVIDER=noop for development")
}

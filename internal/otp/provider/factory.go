package provider

import (
	"github.com/neoscoder/aura-backend/internal/config"
)

func NewRegistry(cfg *config.Config) map[string]OTPProvider {
	selected := cfg.OTP.Provider
	noopProvider := NewNoopProvider(cfg.App.Env)

	registry := map[string]OTPProvider{
		"WHATSAPP": noopProvider,
		"EMAIL":    noopProvider,
	}

	switch selected {
	case "whatsapp":
		registry["WHATSAPP"] = NewWhatsAppProvider(cfg.WhatsApp)
		registry["EMAIL"] = noopProvider
	case "email", "smtp":
		registry["WHATSAPP"] = noopProvider
		registry["EMAIL"] = NewEmailProvider(cfg.Email)
	case "noop", "":
		return registry
	}

	return registry
}

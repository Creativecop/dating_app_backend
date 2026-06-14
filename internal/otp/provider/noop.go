package provider

import (
	"context"
	"fmt"
	"log"

	"github.com/google/uuid"
)

type NoopProvider struct {
	Environment string
}

func NewNoopProvider(environment string) *NoopProvider {
	return &NoopProvider{Environment: environment}
}

func (p *NoopProvider) SendOTP(_ context.Context, to string, code string, purpose string) (string, error) {
	if p.Environment == "production" {
		return "", fmt.Errorf("noop OTP provider cannot send in production")
	}
	log.Printf("[DEV OTP] to=%s purpose=%s code=%s", to, purpose, code)
	return "dev-" + uuid.NewString(), nil
}

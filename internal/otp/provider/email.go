package provider

import (
	"context"
	"fmt"
	"net/smtp"
	"strings"

	"github.com/google/uuid"

	"github.com/neoscoder/aura-backend/internal/config"
)

type EmailProvider struct {
	cfg config.EmailConfig
}

func NewEmailProvider(cfg config.EmailConfig) *EmailProvider {
	return &EmailProvider{cfg: cfg}
}

func (p *EmailProvider) SendOTP(ctx context.Context, to string, code string, purpose string) (string, error) {
	if !p.cfg.Enabled {
		return "", fmt.Errorf("email OTP provider is disabled")
	}
	if p.cfg.SMTPHost == "" {
		return "", fmt.Errorf("SMTP_HOST is required")
	}

	select {
	case <-ctx.Done():
		return "", ctx.Err()
	default:
	}

	addr := fmt.Sprintf("%s:%d", p.cfg.SMTPHost, p.cfg.SMTPPort)
	subject := "Your Aura verification code"
	body := fmt.Sprintf("Your Aura verification code is %s.\r\nThis code will expire in a few minutes.\r\nPurpose: %s\r\n", code, purpose)
	message := strings.Join([]string{
		fmt.Sprintf("From: %s <%s>", p.cfg.FromName, p.cfg.FromEmail),
		"To: " + to,
		"Subject: " + subject,
		"MIME-Version: 1.0",
		"Content-Type: text/plain; charset=UTF-8",
		"",
		body,
	}, "\r\n")

	var auth smtp.Auth
	if p.cfg.SMTPUsername != "" || p.cfg.SMTPPassword != "" {
		auth = smtp.PlainAuth("", p.cfg.SMTPUsername, p.cfg.SMTPPassword, p.cfg.SMTPHost)
	}

	if err := smtp.SendMail(addr, auth, p.cfg.FromEmail, []string{to}, []byte(message)); err != nil {
		return "", err
	}

	return "smtp-" + uuid.NewString(), nil
}

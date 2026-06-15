package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/neoscoder/aura-backend/internal/config"
)

type WhatsAppProvider struct {
	cfg     config.WhatsAppConfig
	client  *http.Client
	baseURL string
}

func NewWhatsAppProvider(cfg config.WhatsAppConfig) *WhatsAppProvider {
	return &WhatsAppProvider{
		cfg: cfg,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
		baseURL: "https://graph.facebook.com",
	}
}

func (p *WhatsAppProvider) SendOTP(ctx context.Context, to string, code string, purpose string) (string, error) {
	if !p.cfg.Enabled {
		return "", fmt.Errorf("whatsapp OTP provider is disabled")
	}
	if p.cfg.PhoneNumberID == "" || p.cfg.AccessToken == "" || p.cfg.TemplateName == "" {
		return "", fmt.Errorf("whatsapp provider credentials are incomplete")
	}
	if strings.TrimSpace(to) == "" || strings.TrimSpace(code) == "" {
		return "", fmt.Errorf("whatsapp recipient and code are required")
	}

	select {
	case <-ctx.Done():
		return "", ctx.Err()
	default:
	}

	payload := whatsAppMessageRequest{
		MessagingProduct: "whatsapp",
		To:               normalizeWhatsAppRecipient(to),
		Type:             "template",
		Template: whatsAppTemplate{
			Name: p.cfg.TemplateName,
			Language: whatsAppLanguage{
				Code: p.cfg.LanguageCode,
			},
			Components: p.templateComponents(code),
		},
	}
	if payload.Template.Language.Code == "" {
		payload.Template.Language.Code = "en_US"
	}
	log.Printf(
		"[WHATSAPP] sending phone_number_id=%s template=%s language=%s to=%s components=%d",
		p.cfg.PhoneNumberID,
		payload.Template.Name,
		payload.Template.Language.Code,
		maskWhatsAppRecipient(payload.To),
		len(payload.Template.Components),
	)

	body, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("marshal whatsapp message: %w", err)
	}

	requestURL, err := p.messagesURL()
	if err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, requestURL, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("create whatsapp request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+p.cfg.AccessToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("send whatsapp message: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return "", fmt.Errorf("read whatsapp response: %w", err)
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		log.Printf("[WHATSAPP] failed status=%d response=%s", resp.StatusCode, responseSnippet(respBody))
		return "", fmt.Errorf("whatsapp message failed with status %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}

	var result whatsAppMessageResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		log.Printf("[WHATSAPP] invalid_response status=%d response=%s", resp.StatusCode, responseSnippet(respBody))
		return "", fmt.Errorf("decode whatsapp response: %w", err)
	}
	if len(result.Messages) > 0 && result.Messages[0].ID != "" {
		log.Printf("[WHATSAPP] sent status=%d provider_message_id=%s", resp.StatusCode, result.Messages[0].ID)
		return result.Messages[0].ID, nil
	}
	messageID := "whatsapp-" + uuid.NewString()
	log.Printf("[WHATSAPP] sent status=%d provider_message_id=%s response_without_message_id=true", resp.StatusCode, messageID)
	return messageID, nil
}

func (p *WhatsAppProvider) templateComponents(code string) []whatsAppComponent {
	if strings.EqualFold(strings.TrimSpace(p.cfg.TemplateName), "hello_world") {
		return nil
	}

	return []whatsAppComponent{
		{
			Type: "body",
			Parameters: []whatsAppParameter{
				{
					Type: "text",
					Text: code,
				},
			},
		},
	}
}

func (p *WhatsAppProvider) messagesURL() (string, error) {
	baseURL := strings.TrimRight(p.baseURL, "/")
	version := strings.Trim(strings.TrimSpace(p.cfg.GraphAPIVersion), "/")
	if version == "" {
		version = "v25.0"
	}
	phoneNumberID := strings.TrimSpace(p.cfg.PhoneNumberID)
	endpoint := fmt.Sprintf("%s/%s/%s/messages", baseURL, version, url.PathEscape(phoneNumberID))
	if _, err := url.ParseRequestURI(endpoint); err != nil {
		return "", fmt.Errorf("invalid whatsapp messages URL: %w", err)
	}
	return endpoint, nil
}

func normalizeWhatsAppRecipient(to string) string {
	return strings.TrimPrefix(strings.TrimSpace(to), "+")
}

func maskWhatsAppRecipient(value string) string {
	value = strings.TrimSpace(value)
	if len(value) <= 4 {
		return "****"
	}
	if len(value) <= 8 {
		return value[:2] + "****" + value[len(value)-2:]
	}
	return value[:4] + "****" + value[len(value)-4:]
}

func responseSnippet(body []byte) string {
	value := strings.TrimSpace(string(body))
	if len(value) <= 500 {
		return value
	}
	return value[:500] + "...truncated"
}

type whatsAppMessageRequest struct {
	MessagingProduct string           `json:"messaging_product"`
	To               string           `json:"to"`
	Type             string           `json:"type"`
	Template         whatsAppTemplate `json:"template"`
}

type whatsAppTemplate struct {
	Name       string              `json:"name"`
	Language   whatsAppLanguage    `json:"language"`
	Components []whatsAppComponent `json:"components,omitempty"`
}

type whatsAppLanguage struct {
	Code string `json:"code"`
}

type whatsAppComponent struct {
	Type       string              `json:"type"`
	Parameters []whatsAppParameter `json:"parameters"`
}

type whatsAppParameter struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type whatsAppMessageResponse struct {
	Messages []struct {
		ID string `json:"id"`
	} `json:"messages"`
}

package provider

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/neoscoder/aura-backend/internal/config"
)

func TestWhatsAppProviderSendOTPSendsTemplateMessage(t *testing.T) {
	var request whatsAppMessageRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/v25.0/1183278588198365/messages" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-token" {
			t.Fatalf("authorization = %q", got)
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatalf("decode request: %v", err)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"messages":[{"id":"wamid.test"}]}`))
	}))
	defer server.Close()

	provider := NewWhatsAppProvider(config.WhatsAppConfig{
		Enabled:         true,
		PhoneNumberID:   "1183278588198365",
		AccessToken:     "test-token",
		TemplateName:    "auth_code",
		LanguageCode:    "en_US",
		GraphAPIVersion: "v25.0",
	})
	provider.baseURL = server.URL
	provider.client = server.Client()

	messageID, err := provider.SendOTP(context.Background(), "+8801712345678", "123456", "LOGIN")
	if err != nil {
		t.Fatalf("SendOTP returned error: %v", err)
	}
	if messageID != "wamid.test" {
		t.Fatalf("messageID = %q", messageID)
	}
	if request.MessagingProduct != "whatsapp" {
		t.Fatalf("messaging_product = %q", request.MessagingProduct)
	}
	if request.To != "8801712345678" {
		t.Fatalf("to = %q", request.To)
	}
	if request.Type != "template" {
		t.Fatalf("type = %q", request.Type)
	}
	if request.Template.Name != "auth_code" {
		t.Fatalf("template name = %q", request.Template.Name)
	}
	if request.Template.Language.Code != "en_US" {
		t.Fatalf("language code = %q", request.Template.Language.Code)
	}
	if len(request.Template.Components) != 1 {
		t.Fatalf("component count = %d", len(request.Template.Components))
	}
	component := request.Template.Components[0]
	if component.Type != "body" {
		t.Fatalf("component type = %q", component.Type)
	}
	if len(component.Parameters) != 1 {
		t.Fatalf("parameter count = %d", len(component.Parameters))
	}
	if component.Parameters[0].Type != "text" || component.Parameters[0].Text != "123456" {
		t.Fatalf("parameter = %#v", component.Parameters[0])
	}
}

func TestWhatsAppProviderSendOTPOmitsComponentsForHelloWorldTemplate(t *testing.T) {
	var request map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatalf("decode request: %v", err)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"messages":[{"id":"wamid.test"}]}`))
	}))
	defer server.Close()

	provider := NewWhatsAppProvider(config.WhatsAppConfig{
		Enabled:         true,
		PhoneNumberID:   "1183278588198365",
		AccessToken:     "test-token",
		TemplateName:    "hello_world",
		LanguageCode:    "en_US",
		GraphAPIVersion: "v25.0",
	})
	provider.baseURL = server.URL
	provider.client = server.Client()

	if _, err := provider.SendOTP(context.Background(), "+8801625930011", "123456", "LOGIN"); err != nil {
		t.Fatalf("SendOTP returned error: %v", err)
	}

	template, ok := request["template"].(map[string]any)
	if !ok {
		t.Fatalf("template = %#v", request["template"])
	}
	if template["name"] != "hello_world" {
		t.Fatalf("template name = %#v", template["name"])
	}
	language, ok := template["language"].(map[string]any)
	if !ok {
		t.Fatalf("language = %#v", template["language"])
	}
	if language["code"] != "en_US" {
		t.Fatalf("language code = %#v", language["code"])
	}
	if _, ok := template["components"]; ok {
		t.Fatalf("components should be omitted for hello_world template: %#v", template["components"])
	}
}

func TestWhatsAppProviderSendOTPReturnsAPIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"error":{"message":"template not found"}}`, http.StatusBadRequest)
	}))
	defer server.Close()

	provider := NewWhatsAppProvider(config.WhatsAppConfig{
		Enabled:         true,
		PhoneNumberID:   "1183278588198365",
		AccessToken:     "test-token",
		TemplateName:    "auth_code",
		GraphAPIVersion: "v25.0",
	})
	provider.baseURL = server.URL
	provider.client = server.Client()

	if _, err := provider.SendOTP(context.Background(), "+8801712345678", "123456", "LOGIN"); err == nil {
		t.Fatal("SendOTP returned nil error")
	}
}

func TestWhatsAppProviderSendOTPRequiresCredentials(t *testing.T) {
	provider := NewWhatsAppProvider(config.WhatsAppConfig{Enabled: true})

	if _, err := provider.SendOTP(context.Background(), "+8801712345678", "123456", "LOGIN"); err == nil {
		t.Fatal("SendOTP returned nil error")
	}
}

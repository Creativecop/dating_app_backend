package middleware

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func TestSafeRequestPathMasksTokenQuery(t *testing.T) {
	c := &gin.Context{
		Request: &http.Request{
			URL: &url.URL{
				Path:     "/ws",
				RawQuery: "token=secret&cursor=abc",
			},
		},
	}

	got := safeRequestPath(c)
	if got != "/ws?cursor=abc&token=%2A%2A%2A" {
		t.Fatalf("unexpected safe path: %s", got)
	}
}

func TestSanitizeBodyRedactsSensitiveRequestFields(t *testing.T) {
	got := sanitizeBody([]byte(`{
		"phone":"+8801000000000",
		"email":"user@example.com",
		"code":"123456",
		"refreshToken":"secret",
		"nested":{"latitude":23.8,"longitude":90.4,"bio":"ok"}
	}`), true)

	for _, leaked := range []string{"+8801000000000", "user@example.com", "123456", "secret", "23.8", "90.4"} {
		if strings.Contains(got, leaked) {
			t.Fatalf("sensitive value leaked in %s", got)
		}
	}
	if !strings.Contains(got, `"bio":"ok"`) {
		t.Fatalf("non-sensitive field missing: %s", got)
	}
}

func TestSanitizeBodyKeepsResponseErrorCode(t *testing.T) {
	got := sanitizeBody([]byte(`{"success":false,"error":{"code":"VALIDATION_ERROR"}}`), false)
	if !strings.Contains(got, "VALIDATION_ERROR") {
		t.Fatalf("expected response error code to be preserved: %s", got)
	}
}

func TestPrettyHTTPLogIsStructuredAndRedacted(t *testing.T) {
	userID := uint64(12)
	adminUserID := uint64(34)
	got := prettyHTTPLog(
		"req-1",
		http.MethodPost,
		"/api/v1/admin/auth/login",
		http.StatusBadRequest,
		25*time.Millisecond,
		"127.0.0.1",
		"/api/v1/admin/auth/login",
		&userID,
		&adminUserID,
		"VALIDATION_ERROR",
		sanitizeBody([]byte(`{"email":"admin@example.com","password":"secret","name":"ok"}`), true),
		`{"success":false,"error":{"code":"VALIDATION_ERROR"}}`,
	)

	var record map[string]any
	if err := json.Unmarshal([]byte(got), &record); err != nil {
		t.Fatalf("log is not JSON: %v: %s", err, got)
	}
	if record["event"] != "http_request" || record["request_id"] != "req-1" || record["error_code"] != "VALIDATION_ERROR" {
		t.Fatalf("unexpected log fields: %#v", record)
	}
	request, _ := record["request"].(string)
	if strings.Contains(request, "admin@example.com") || strings.Contains(request, "secret") {
		t.Fatalf("sensitive request values leaked: %s", request)
	}
	if !strings.Contains(request, `"name":"ok"`) {
		t.Fatalf("non-sensitive request value missing: %s", request)
	}
}

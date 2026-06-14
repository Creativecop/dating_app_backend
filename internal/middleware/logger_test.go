package middleware

import (
	"net/http"
	"net/url"
	"strings"
	"testing"

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

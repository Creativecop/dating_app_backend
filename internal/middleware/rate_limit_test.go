package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func TestRateLimiterFailClosedReturnsRateLimited(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	limiter := NewRateLimiter(nil, true)
	router.GET("/limited", limiter.Limit(RateLimitRule{
		Scope:      "test",
		Limit:      1,
		Window:     time.Minute,
		Identifier: ConstantIdentifier("subject"),
		FailClosed: true,
	}), func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/limited", nil)
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusTooManyRequests {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusTooManyRequests)
	}
	if recorder.Header().Get("Retry-After") != "60" {
		t.Fatalf("Retry-After = %q, want 60", recorder.Header().Get("Retry-After"))
	}
	var body struct {
		StatusCode       int `json:"statusCode"`
		StatusCodeCompat int `json:"status_code"`
		Error            struct {
			Code              string `json:"code"`
			RetryAfterSeconds int64  `json:"retryAfterSeconds"`
		} `json:"error"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if body.Error.Code != "RATE_LIMITED" || body.Error.RetryAfterSeconds != 60 {
		t.Fatalf("unexpected rate limit body: %#v", body.Error)
	}
	if body.StatusCode != http.StatusTooManyRequests || body.StatusCodeCompat != http.StatusTooManyRequests {
		t.Fatalf("unexpected rate limit status fields: statusCode=%d status_code=%d", body.StatusCode, body.StatusCodeCompat)
	}
}

func TestRateLimiterFailOpenContinuesOnRedisError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	limiter := NewRateLimiter(nil, true)
	router.GET("/limited", limiter.Limit(RateLimitRule{
		Scope:      "test",
		Limit:      1,
		Window:     time.Minute,
		Identifier: ConstantIdentifier("subject"),
		FailClosed: false,
	}), func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/limited", nil)
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusNoContent)
	}
}

func TestRateLimitKeyDoesNotExposeIdentifier(t *testing.T) {
	key := rateLimitKey("admin_login", "admin@example.com", 15*time.Minute)
	if key == "" {
		t.Fatal("expected key")
	}
	if key == "rate:admin_login:admin@example.com:900" {
		t.Fatal("raw identifier leaked into rate limit key")
	}
}

package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestJSONBodyLimitRejectsOversizedBody(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/body", JSONBodyLimit(8), func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/body", strings.NewReader(`{"large":true}`))
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusRequestEntityTooLarge)
	}
	var body struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if body.Error.Code != "REQUEST_BODY_TOO_LARGE" {
		t.Fatalf("code = %q, want REQUEST_BODY_TOO_LARGE", body.Error.Code)
	}
}

func TestRequireIdempotencyKey(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/payment", RequireIdempotencyKey(), func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	missing := httptest.NewRecorder()
	router.ServeHTTP(missing, httptest.NewRequest(http.MethodPost, "/payment", nil))
	if missing.Code != http.StatusBadRequest {
		t.Fatalf("missing key status = %d, want %d", missing.Code, http.StatusBadRequest)
	}

	present := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/payment", nil)
	req.Header.Set("Idempotency-Key", "payment-123")
	router.ServeHTTP(present, req)
	if present.Code != http.StatusNoContent {
		t.Fatalf("present key status = %d, want %d", present.Code, http.StatusNoContent)
	}
}

func TestValidateUUIDParamsRejectsInvalidUUID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/items/:itemUuid", ValidateUUIDParams("itemUuid"), func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/items/not-a-uuid", nil)
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusBadRequest)
	}
}

package middleware

import (
	"net/http"
	"net/url"
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

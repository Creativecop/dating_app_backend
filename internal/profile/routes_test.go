package profile

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestCatalogAliasRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	v1 := router.Group("/api/v1")
	RegisterRoutes(v1, func(c *gin.Context) {
		c.AbortWithStatus(http.StatusNoContent)
	}, &Handler{})

	paths := []string{
		"/api/v1/profile/interests",
		"/api/v1/profile/prompts",
		"/api/v1/profile/lifestyle",
	}
	for _, path := range paths {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusNoContent {
			t.Fatalf("GET %s status = %d, want %d", path, rec.Code, http.StatusNoContent)
		}
	}
}

package router

import (
	"os"
	"strings"
	"testing"
)

func TestOpenAPIDocumentsReleaseRoutes(t *testing.T) {
	raw, err := os.ReadFile("../../docs/openapi.yaml")
	if err != nil {
		t.Fatalf("read OpenAPI spec: %v", err)
	}
	spec := string(raw)
	for _, path := range []string{
		"/health:",
		"/api/v1/health:",
		"/api/v1/admin/dashboard/summary:",
		"/api/v1/admin/analytics/users:",
		"/api/v1/admin/analytics/reports:",
		"/api/v1/admin/analytics/restrictions:",
		"/api/v1/admin/analytics/trust-safety:",
		"/api/v1/admin/analytics/admin-activity:",
		"/api/v1/admin/analytics/subscription-payments:",
		"/api/v1/admin/audit-logs:",
		"/api/v1/subscriptions/manual-payment-requests:",
		"/api/v1/admin/subscriptions/payment-requests/{paymentRequestUuid}/approve:",
		"/api/v1/admin/subscriptions/payment-requests/{paymentRequestUuid}/reject:",
	} {
		if !strings.Contains(spec, path) {
			t.Fatalf("OpenAPI spec missing %s", path)
		}
	}
	for _, value := range []string{"Idempotency-Key", "actorAdminUserId", "actionType"} {
		if !strings.Contains(spec, value) {
			t.Fatalf("OpenAPI spec missing %s", value)
		}
	}
}

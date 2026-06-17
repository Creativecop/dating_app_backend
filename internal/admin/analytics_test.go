package admin

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestSubscriptionPaymentAnalyticsResponseOmitsSensitiveFields(t *testing.T) {
	response := SubscriptionPaymentAnalyticsResponse{
		Period: AnalyticsPeriod{
			From: time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC),
			To:   time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC),
		},
		Totals: SubscriptionPaymentTotals{PaymentCount: 1, TotalAmount: 100},
		StatusBreakdown: []PaymentBreakdown{
			{Key: "APPROVED", Count: 1, Amount: 100},
		},
	}
	payload, err := json.Marshal(response)
	if err != nil {
		t.Fatalf("Marshal returned error: %v", err)
	}
	raw := string(payload)
	for _, forbidden := range []string{"payerPhone", "paymentReference", "screenshot", "userUuid", "adminReviewNote"} {
		if strings.Contains(raw, forbidden) {
			t.Fatalf("subscription analytics response includes sensitive field %q: %s", forbidden, raw)
		}
	}
}

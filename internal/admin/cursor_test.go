package admin

import (
	"testing"
	"time"
)

func TestAuditLogCursorRoundTrip(t *testing.T) {
	original := auditLogCursor{CreatedAt: time.Now().UTC(), AuditLogID: 42}
	raw, err := encodeAuditLogCursor(original)
	if err != nil {
		t.Fatalf("encodeAuditLogCursor returned error: %v", err)
	}
	decoded, err := decodeAuditLogCursor(raw)
	if err != nil {
		t.Fatalf("decodeAuditLogCursor returned error: %v", err)
	}
	if decoded.AuditLogID != original.AuditLogID || !decoded.CreatedAt.Equal(original.CreatedAt) {
		t.Fatalf("decoded cursor = %#v, want %#v", decoded, original)
	}
}

func TestNormalizeAuditLogLimit(t *testing.T) {
	value, err := normalizeAuditLogLimit("")
	if err != nil {
		t.Fatalf("normalizeAuditLogLimit returned error: %v", err)
	}
	if value != defaultAuditLogLimit {
		t.Fatalf("default limit = %d", value)
	}
	value, err = normalizeAuditLogLimit("500")
	if err != nil {
		t.Fatalf("normalizeAuditLogLimit returned error: %v", err)
	}
	if value != maxAuditLogLimit {
		t.Fatalf("max limit = %d", value)
	}
	if _, err := normalizeAuditLogLimit("0"); err == nil {
		t.Fatal("expected zero limit to fail")
	}
}

func TestNormalizeAnalyticsRangeDefaultUTCWindow(t *testing.T) {
	now := time.Date(2026, 6, 17, 18, 30, 0, 0, time.UTC)
	period, err := normalizeAnalyticsRange("", "", now)
	if err != nil {
		t.Fatalf("normalizeAnalyticsRange returned error: %v", err)
	}
	wantFrom := time.Date(2026, 5, 19, 0, 0, 0, 0, time.UTC)
	wantTo := time.Date(2026, 6, 18, 0, 0, 0, 0, time.UTC)
	if !period.From.Equal(wantFrom) || !period.To.Equal(wantTo) {
		t.Fatalf("period = %s..%s, want %s..%s", period.From, period.To, wantFrom, wantTo)
	}
}

func TestNormalizeAnalyticsRangeRejectsOverMax(t *testing.T) {
	_, err := normalizeAnalyticsRange("2025-01-01", "2026-01-03", time.Date(2026, 6, 17, 0, 0, 0, 0, time.UTC))
	if err == nil {
		t.Fatal("expected over-max range to fail")
	}
	serviceErr, ok := AsServiceError(err)
	if !ok || serviceErr.Code != CodeAdminAnalyticsRangeInvalid {
		t.Fatalf("err = %#v, want %s", err, CodeAdminAnalyticsRangeInvalid)
	}
}

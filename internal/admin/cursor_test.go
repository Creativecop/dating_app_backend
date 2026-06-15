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

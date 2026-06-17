package admin

import "testing"

func TestNormalizeAuditLogListQueryAliases(t *testing.T) {
	query, err := normalizeAuditLogListQuery(AuditLogListQuery{
		ActorAdminUserID: "11111111-1111-1111-1111-111111111111",
		ActionType:       "REPORT_REVIEWED",
		From:             "2026-06-01T00:00:00Z",
		To:               "2026-06-18T00:00:00Z",
	})
	if err != nil {
		t.Fatalf("normalizeAuditLogListQuery returned error: %v", err)
	}
	if query.AdminUserUUID != "11111111-1111-1111-1111-111111111111" {
		t.Fatalf("admin user alias was not normalized: %#v", query)
	}
	if query.Action != "REPORT_REVIEWED" {
		t.Fatalf("action alias was not normalized: %#v", query)
	}
	if query.CreatedFrom != "2026-06-01T00:00:00Z" || query.CreatedTo != "2026-06-18T00:00:00Z" {
		t.Fatalf("date aliases were not normalized: %#v", query)
	}
}

func TestNormalizeAuditLogListQueryCanonicalAndAliasSameValueAllowed(t *testing.T) {
	_, err := normalizeAuditLogListQuery(AuditLogListQuery{
		Action:     "REPORT_REVIEWED",
		ActionType: "REPORT_REVIEWED",
	})
	if err != nil {
		t.Fatalf("expected same canonical and alias value to pass: %v", err)
	}
}

func TestNormalizeAuditLogListQueryConflictingAliasRejected(t *testing.T) {
	_, err := normalizeAuditLogListQuery(AuditLogListQuery{
		Action:     "REPORT_REVIEWED",
		ActionType: "ADMIN_ROLE_ASSIGNED",
	})
	if err == nil {
		t.Fatal("expected conflicting audit filters to fail")
	}
	serviceErr, ok := AsServiceError(err)
	if !ok || serviceErr.Code != CodeAdminAuditFilterConflict {
		t.Fatalf("err = %#v, want %s", err, CodeAdminAuditFilterConflict)
	}
}

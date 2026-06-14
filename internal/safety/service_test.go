package safety

import (
	"strings"
	"testing"
)

func TestSeverityForReason(t *testing.T) {
	tests := map[string]string{
		ReasonUnderage:       SeverityCritical,
		ReasonViolenceThreat: SeverityCritical,
		ReasonHateSpeech:     SeverityHigh,
		ReasonHarassment:     SeverityHigh,
		ReasonSexualContent:  SeverityHigh,
		ReasonImpersonation:  SeverityHigh,
		ReasonScamSpam:       SeverityMedium,
		ReasonFakeProfile:    SeverityMedium,
		ReasonOther:          SeverityMedium,
		"UNKNOWN":            SeverityLow,
	}

	for reason, expected := range tests {
		if got := SeverityForReason(reason); got != expected {
			t.Fatalf("SeverityForReason(%q) = %q, want %q", reason, got, expected)
		}
	}
}

func TestNormalizeReportRequest(t *testing.T) {
	note := "  abusive message  "
	result, err := normalizeReportRequest(CreateReportRequest{
		TargetType: "message",
		TargetUUID: "7f2a43b5-bbd7-48a5-a7da-5c73a2dd3474",
		ReasonCode: " harassment ",
		Note:       &note,
		BlockUser:  true,
	})
	if err != nil {
		t.Fatalf("normalize report: %v", err)
	}
	if result.TargetType != TargetMessage {
		t.Fatalf("unexpected target type: %s", result.TargetType)
	}
	if result.ReasonCode != ReasonHarassment {
		t.Fatalf("unexpected reason: %s", result.ReasonCode)
	}
	if result.Note == nil || *result.Note != "abusive message" {
		t.Fatalf("unexpected note: %#v", result.Note)
	}
	if !result.BlockUser {
		t.Fatal("expected blockUser to be preserved")
	}
}

func TestNormalizeReportRequestRejectsLongNote(t *testing.T) {
	note := strings.Repeat("a", 1001)
	_, err := normalizeReportRequest(CreateReportRequest{
		TargetType: TargetUser,
		TargetUUID: "7f2a43b5-bbd7-48a5-a7da-5c73a2dd3474",
		ReasonCode: ReasonOther,
		Note:       &note,
	})
	if err == nil {
		t.Fatal("expected long note to fail")
	}
}

func TestReasonApplies(t *testing.T) {
	raw := `["USER","PROFILE","MESSAGE"]`
	if !reasonApplies(raw, TargetMessage) {
		t.Fatal("expected reason to apply to message")
	}
	if reasonApplies(raw, TargetMedia) {
		t.Fatal("expected reason not to apply to media")
	}
}

func TestNormalizeBlockRequest(t *testing.T) {
	note := "  no contact  "
	reason := "harassment"
	result, err := normalizeBlockRequest(BlockUserRequest{ReasonCode: &reason, Note: &note}, BlockSourceReportFlow)
	if err != nil {
		t.Fatalf("normalize block: %v", err)
	}
	if result.Source != BlockSourceReportFlow {
		t.Fatalf("unexpected source: %s", result.Source)
	}
	if result.ReasonCode != ReasonHarassment {
		t.Fatalf("unexpected reason: %s", result.ReasonCode)
	}
	if result.Note == nil || *result.Note != "no contact" {
		t.Fatalf("unexpected note: %#v", result.Note)
	}
}

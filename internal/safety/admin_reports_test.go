package safety

import (
	"testing"
	"time"
)

func TestNormalizeReviewReportRequestMatrix(t *testing.T) {
	now := time.Now().UTC()
	tests := []struct {
		name    string
		req     ReviewReportRequest
		wantErr bool
	}{
		{
			name: "reviewed with no action",
			req:  ReviewReportRequest{Decision: ReportReviewed, Reason: "Checked"},
		},
		{
			name: "dismissed with none action",
			req:  ReviewReportRequest{Decision: ReportDismissed, Reason: "No violation", Action: &ReviewReportAction{Type: ReportActionNone}},
		},
		{
			name: "actioned with restrict user",
			req: ReviewReportRequest{
				Decision: ReportActioned,
				Reason:   "Confirmed abuse",
				Action:   &ReviewReportAction{Type: ReportActionRestrictUser, RestrictionType: "COMMENT_BAN"},
			},
		},
		{
			name:    "actioned without real action rejected",
			req:     ReviewReportRequest{Decision: ReportActioned, Reason: "Confirmed abuse"},
			wantErr: true,
		},
		{
			name:    "reviewed with action rejected",
			req:     ReviewReportRequest{Decision: ReportReviewed, Reason: "Checked", Action: &ReviewReportAction{Type: ReportActionRestrictUser, RestrictionType: "COMMENT_BAN"}},
			wantErr: true,
		},
		{
			name:    "hide comment disabled",
			req:     ReviewReportRequest{Decision: ReportActioned, Reason: "Confirmed abuse", Action: &ReviewReportAction{Type: ReportActionHideComment}},
			wantErr: true,
		},
		{
			name:    "force end live disabled",
			req:     ReviewReportRequest{Decision: ReportActioned, Reason: "Confirmed abuse", Action: &ReviewReportAction{Type: ReportActionForceEndLive}},
			wantErr: true,
		},
		{
			name:    "hide chat message disabled",
			req:     ReviewReportRequest{Decision: ReportActioned, Reason: "Confirmed abuse", Action: &ReviewReportAction{Type: ReportActionHideChatMessage}},
			wantErr: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := normalizeReviewReportRequest(test.req, now)
			if test.wantErr && err == nil {
				t.Fatal("expected error")
			}
			if !test.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestNormalizeReviewReportRequestRequiresReason(t *testing.T) {
	_, err := normalizeReviewReportRequest(ReviewReportRequest{Decision: ReportReviewed}, time.Now().UTC())
	if err == nil {
		t.Fatal("expected missing reason to fail")
	}
}

func TestReportStatusValidation(t *testing.T) {
	for _, status := range []string{ReportPending, ReportReviewed, ReportDismissed, ReportActioned} {
		if !validReportStatus(status) {
			t.Fatalf("%s should be valid", status)
		}
	}
	if validReportStatus("OPEN") {
		t.Fatal("OPEN should not be a Phase 13CDE report status")
	}
}

package safety

import "time"

type ReportReasonResponse struct {
	ReasonCode  string   `json:"reasonCode"`
	Title       string   `json:"title"`
	Description *string  `json:"description,omitempty"`
	AppliesTo   []string `json:"appliesTo"`
}

type ReportReasonsResponse struct {
	Items []ReportReasonResponse `json:"items"`
}

type CreateReportRequest struct {
	TargetType string  `json:"targetType" binding:"required"`
	TargetUUID string  `json:"targetUuid" binding:"required"`
	ReasonCode string  `json:"reasonCode" binding:"required"`
	Note       *string `json:"note"`
	BlockUser  bool    `json:"blockUser"`
}

type CreateReportResponse struct {
	ReportUUID string `json:"reportUuid"`
	CaseUUID   string `json:"caseUuid"`
	Status     string `json:"status"`
	Blocked    bool   `json:"blocked,omitempty"`
}

type ReportTargetSnapshot struct {
	TargetType       string         `json:"targetType"`
	TargetUUID       string         `json:"targetUuid"`
	ReportedUserID   uint64         `json:"-"`
	ReportedUserUUID string         `json:"reportedUserUuid"`
	Evidence         map[string]any `json:"-"`
}

type MyReportsResponse struct {
	Items []MyReportResponse `json:"items"`
}

type MyReportResponse struct {
	ReportUUID string    `json:"reportUuid"`
	TargetType string    `json:"targetType"`
	ReasonCode string    `json:"reasonCode"`
	Status     string    `json:"status"`
	CreatedAt  time.Time `json:"createdAt"`
}

type BlockUserRequest struct {
	ReasonCode *string `json:"reasonCode"`
	Note       *string `json:"note"`
}

type BlockedUserResponse struct {
	UserUUID    string    `json:"userUuid"`
	DisplayName *string   `json:"displayName"`
	BlockedAt   time.Time `json:"blockedAt"`
	ReasonCode  *string   `json:"reasonCode,omitempty"`
	Source      string    `json:"source"`
}

type BlockListResponse struct {
	Items []BlockedUserResponse `json:"items"`
}

type UpdateSafetySettingsRequest struct {
	AllowMessageRequests *bool `json:"allowMessageRequests"`
	AutoHideBlockedUsers *bool `json:"autoHideBlockedUsers"`
}

type SafetySettingsResponse struct {
	UUID                 string `json:"uuid"`
	AllowMessageRequests bool   `json:"allowMessageRequests"`
	AutoHideBlockedUsers bool   `json:"autoHideBlockedUsers"`
}

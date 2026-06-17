package safety

import (
	"encoding/base64"
	"encoding/json"
	"strconv"
	"strings"
	"time"
)

const (
	defaultAdminReportLimit = 50
	maxAdminReportLimit     = 100
	maxAdminReportRangeDays = 366
)

type adminReportCursor struct {
	CreatedAt time.Time `json:"createdAt"`
	ReportID  uint64    `json:"reportId"`
}

func encodeAdminReportCursor(cursor adminReportCursor) (string, error) {
	payload, err := json.Marshal(cursor)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(payload), nil
}

func decodeAdminReportCursor(raw string) (*adminReportCursor, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	payload, err := base64.RawURLEncoding.DecodeString(raw)
	if err != nil {
		return nil, validationError("Cursor is invalid", map[string]any{"field": "cursor"})
	}
	var cursor adminReportCursor
	if err := json.Unmarshal(payload, &cursor); err != nil {
		return nil, validationError("Cursor is invalid", map[string]any{"field": "cursor"})
	}
	if cursor.CreatedAt.IsZero() || cursor.ReportID == 0 {
		return nil, validationError("Cursor is invalid", map[string]any{"field": "cursor"})
	}
	return &cursor, nil
}

func normalizeAdminReportLimit(raw string) (int, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return defaultAdminReportLimit, nil
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value < 1 {
		return 0, validationError("Limit must be a positive integer", map[string]any{"field": "limit"})
	}
	if value > maxAdminReportLimit {
		return maxAdminReportLimit, nil
	}
	return value, nil
}

func validateAdminReportDateRange(from *time.Time, to *time.Time) error {
	if from == nil || to == nil {
		return nil
	}
	if !from.Before(*to) {
		return validationError("createdFrom must be before createdTo", map[string]any{"field": "createdFrom"})
	}
	if to.Sub(*from) > maxAdminReportRangeDays*24*time.Hour {
		return validationError("Date range cannot exceed 366 days", map[string]any{"field": "createdTo"})
	}
	return nil
}

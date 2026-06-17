package admin

import (
	"encoding/base64"
	"encoding/json"
	"strconv"
	"strings"
	"time"
)

const (
	defaultAuditLogLimit  = 50
	maxAuditLogLimit      = 100
	maxAnalyticsRangeDays = 366
)

type auditLogCursor struct {
	CreatedAt  time.Time `json:"createdAt"`
	AuditLogID uint64    `json:"auditLogId"`
}

type adminUserListCursor struct {
	CreatedAt time.Time `json:"createdAt"`
	UserID    uint64    `json:"userId"`
}

func encodeAuditLogCursor(cursor auditLogCursor) (string, error) {
	payload, err := json.Marshal(cursor)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(payload), nil
}

func decodeAuditLogCursor(raw string) (*auditLogCursor, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	payload, err := base64.RawURLEncoding.DecodeString(raw)
	if err != nil {
		return nil, validationError("Cursor is invalid", map[string]any{"field": "cursor"})
	}
	var cursor auditLogCursor
	if err := json.Unmarshal(payload, &cursor); err != nil {
		return nil, validationError("Cursor is invalid", map[string]any{"field": "cursor"})
	}
	if cursor.CreatedAt.IsZero() || cursor.AuditLogID == 0 {
		return nil, validationError("Cursor is invalid", map[string]any{"field": "cursor"})
	}
	return &cursor, nil
}

func normalizeAuditLogLimit(raw string) (int, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return defaultAuditLogLimit, nil
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value < 1 {
		return 0, validationError("Limit must be a positive integer", map[string]any{"field": "limit"})
	}
	if value > maxAuditLogLimit {
		return maxAuditLogLimit, nil
	}
	return value, nil
}

func normalizeAnalyticsRange(fromRaw string, toRaw string, now time.Time) (AnalyticsPeriod, error) {
	now = now.UTC()
	defaultTo := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC).AddDate(0, 0, 1)
	defaultFrom := defaultTo.AddDate(0, 0, -30)

	from := defaultFrom
	to := defaultTo
	var err error
	if strings.TrimSpace(fromRaw) != "" {
		from, err = parseAnalyticsDate(fromRaw, "from")
		if err != nil {
			return AnalyticsPeriod{}, err
		}
	}
	if strings.TrimSpace(toRaw) != "" {
		to, err = parseAnalyticsDate(toRaw, "to")
		if err != nil {
			return AnalyticsPeriod{}, err
		}
	}
	if !from.Before(to) {
		return AnalyticsPeriod{}, analyticsRangeError("from must be before to")
	}
	if to.Sub(from) > maxAnalyticsRangeDays*24*time.Hour {
		return AnalyticsPeriod{}, analyticsRangeError("Date range cannot exceed 366 days")
	}
	return AnalyticsPeriod{From: from, To: to}, nil
}

func parseAnalyticsDate(raw string, field string) (time.Time, error) {
	parsed, err := time.Parse("2006-01-02", strings.TrimSpace(raw))
	if err != nil {
		return time.Time{}, &ServiceError{
			Status:  400,
			Code:    CodeAdminAnalyticsRangeInvalid,
			Message: field + " must use YYYY-MM-DD",
			Details: map[string]any{"field": field},
		}
	}
	return parsed.UTC(), nil
}

func analyticsRangeError(message string) *ServiceError {
	return &ServiceError{Status: 400, Code: CodeAdminAnalyticsRangeInvalid, Message: message}
}

func encodeAdminUserListCursor(cursor adminUserListCursor) (string, error) {
	payload, err := json.Marshal(cursor)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(payload), nil
}

func decodeAdminUserListCursor(raw string) (*adminUserListCursor, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	payload, err := base64.RawURLEncoding.DecodeString(raw)
	if err != nil {
		return nil, validationError("Cursor is invalid", map[string]any{"field": "cursor"})
	}
	var cursor adminUserListCursor
	if err := json.Unmarshal(payload, &cursor); err != nil {
		return nil, validationError("Cursor is invalid", map[string]any{"field": "cursor"})
	}
	if cursor.CreatedAt.IsZero() || cursor.UserID == 0 {
		return nil, validationError("Cursor is invalid", map[string]any{"field": "cursor"})
	}
	return &cursor, nil
}

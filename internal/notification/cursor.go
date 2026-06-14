package notification

import (
	"encoding/base64"
	"encoding/json"
	"time"
)

type notificationCursor struct {
	CreatedAt      time.Time `json:"createdAt"`
	NotificationID uint64    `json:"notificationId"`
}

func encodeCursor(cursor notificationCursor) (string, error) {
	payload, err := json.Marshal(cursor)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(payload), nil
}

func decodeCursor(raw string) (*notificationCursor, error) {
	if raw == "" {
		return nil, nil
	}
	payload, err := base64.RawURLEncoding.DecodeString(raw)
	if err != nil {
		return nil, validationError("Cursor is invalid", map[string]any{"field": "cursor"})
	}
	var cursor notificationCursor
	if err := json.Unmarshal(payload, &cursor); err != nil {
		return nil, validationError("Cursor is invalid", map[string]any{"field": "cursor"})
	}
	if cursor.NotificationID == 0 || cursor.CreatedAt.IsZero() {
		return nil, validationError("Cursor is invalid", map[string]any{"field": "cursor"})
	}
	return &cursor, nil
}

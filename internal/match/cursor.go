package match

import (
	"encoding/base64"
	"encoding/json"
	"time"
)

const (
	defaultListLimit = 20
	maxListLimit     = 50
)

type listCursor struct {
	MatchedAt time.Time `json:"matchedAt"`
	MatchID   uint64    `json:"matchId"`
}

func encodeListCursor(cursor listCursor) (string, error) {
	payload, err := json.Marshal(cursor)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(payload), nil
}

func decodeListCursor(raw string) (*listCursor, error) {
	if raw == "" {
		return nil, nil
	}
	payload, err := base64.RawURLEncoding.DecodeString(raw)
	if err != nil {
		return nil, validationError("Cursor is invalid", map[string]any{"field": "cursor"})
	}
	var cursor listCursor
	if err := json.Unmarshal(payload, &cursor); err != nil {
		return nil, validationError("Cursor is invalid", map[string]any{"field": "cursor"})
	}
	if cursor.MatchedAt.IsZero() || cursor.MatchID == 0 {
		return nil, validationError("Cursor is invalid", map[string]any{"field": "cursor"})
	}
	return &cursor, nil
}

func normalizeListLimit(raw string) (int, error) {
	if raw == "" {
		return defaultListLimit, nil
	}
	value := 0
	for _, ch := range raw {
		if ch < '0' || ch > '9' {
			return 0, validationError("Limit must be a positive integer", map[string]any{"field": "limit"})
		}
		value = value*10 + int(ch-'0')
	}
	if value < 1 {
		return 0, validationError("Limit must be a positive integer", map[string]any{"field": "limit"})
	}
	if value > maxListLimit {
		value = maxListLimit
	}
	return value, nil
}

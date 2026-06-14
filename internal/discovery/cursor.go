package discovery

import (
	"encoding/base64"
	"encoding/json"
	"time"
)

type feedCursor struct {
	DistanceMeters  int       `json:"distanceMeters"`
	CompletedAt     time.Time `json:"completedAt"`
	CandidateUserID uint64    `json:"candidateUserId"`
}

func encodeFeedCursor(cursor feedCursor) (string, error) {
	payload, err := json.Marshal(cursor)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(payload), nil
}

func decodeFeedCursor(raw string) (*feedCursor, error) {
	if raw == "" {
		return nil, nil
	}
	payload, err := base64.RawURLEncoding.DecodeString(raw)
	if err != nil {
		return nil, validationError("Cursor is invalid", map[string]any{"field": "cursor"})
	}
	var cursor feedCursor
	if err := json.Unmarshal(payload, &cursor); err != nil {
		return nil, validationError("Cursor is invalid", map[string]any{"field": "cursor"})
	}
	if cursor.CompletedAt.IsZero() || cursor.CandidateUserID == 0 {
		return nil, validationError("Cursor is invalid", map[string]any{"field": "cursor"})
	}
	return &cursor, nil
}

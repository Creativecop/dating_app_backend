package chat

import (
	"encoding/base64"
	"encoding/json"
	"strings"
	"time"
)

const (
	defaultChatListLimit = 20
	maxChatListLimit     = 50
	defaultMessageLimit  = 30
	maxMessageLimit      = 100
)

type chatListCursor struct {
	LastMessageAt  *time.Time `json:"lastMessageAt"`
	ConversationID uint64     `json:"conversationId"`
}

type messageCursor struct {
	CreatedAt time.Time `json:"createdAt"`
	MessageID uint64    `json:"messageId"`
}

func encodeChatListCursor(cursor chatListCursor) (string, error) {
	return encodeCursor(cursor)
}

func decodeChatListCursor(raw string) (*chatListCursor, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, nil
	}
	var cursor chatListCursor
	if err := decodeCursor(raw, &cursor); err != nil {
		return nil, validationError("Cursor is invalid", map[string]any{"field": "cursor"})
	}
	if cursor.ConversationID == 0 {
		return nil, validationError("Cursor is invalid", map[string]any{"field": "cursor"})
	}
	return &cursor, nil
}

func encodeMessageCursor(cursor messageCursor) (string, error) {
	return encodeCursor(cursor)
}

func decodeMessageCursor(raw string) (*messageCursor, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, nil
	}
	var cursor messageCursor
	if err := decodeCursor(raw, &cursor); err != nil {
		return nil, validationError("Cursor is invalid", map[string]any{"field": "cursor"})
	}
	if cursor.CreatedAt.IsZero() || cursor.MessageID == 0 {
		return nil, validationError("Cursor is invalid", map[string]any{"field": "cursor"})
	}
	return &cursor, nil
}

func normalizeLimit(raw string, defaultValue int, maxValue int) (int, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return defaultValue, nil
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
	if value > maxValue {
		return maxValue, nil
	}
	return value, nil
}

func encodeCursor(value any) (string, error) {
	payload, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(payload), nil
}

func decodeCursor(raw string, target any) error {
	payload, err := base64.RawURLEncoding.DecodeString(raw)
	if err != nil {
		return err
	}
	return json.Unmarshal(payload, target)
}

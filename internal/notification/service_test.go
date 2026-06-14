package notification

import (
	"encoding/json"
	"testing"
	"time"
)

func TestNormalizeSettingsRequiresFullReplacement(t *testing.T) {
	var req UpdateSettingsRequest
	err := json.Unmarshal([]byte(`{
		"pushEnabled": true,
		"newMatchEnabled": true,
		"chatMessageEnabled": true,
		"superLikeEnabled": true,
		"quietHoursEnabled": false
	}`), &req)
	if err != nil {
		t.Fatalf("unmarshal request: %v", err)
	}

	_, err = normalizeSettingsRequest(req, "Asia/Dhaka")
	if err == nil {
		t.Fatal("expected missing timezone to fail")
	}
}

func TestNormalizeSettingsQuietHours(t *testing.T) {
	var req UpdateSettingsRequest
	err := json.Unmarshal([]byte(`{
		"pushEnabled": true,
		"newMatchEnabled": true,
		"chatMessageEnabled": true,
		"superLikeEnabled": true,
		"quietHoursEnabled": true,
		"quietHoursStart": "22:30",
		"quietHoursEnd": "07:15",
		"timezone": "Asia/Dhaka"
	}`), &req)
	if err != nil {
		t.Fatalf("unmarshal request: %v", err)
	}

	normalized, err := normalizeSettingsRequest(req, "Asia/Dhaka")
	if err != nil {
		t.Fatalf("normalize settings: %v", err)
	}
	if normalized.QuietHoursStart == nil || *normalized.QuietHoursStart != "22:30:00" {
		t.Fatalf("unexpected quiet hours start: %#v", normalized.QuietHoursStart)
	}
	if normalized.QuietHoursEnd == nil || *normalized.QuietHoursEnd != "07:15:00" {
		t.Fatalf("unexpected quiet hours end: %#v", normalized.QuietHoursEnd)
	}
}

func TestInQuietHoursOvernight(t *testing.T) {
	start := "22:00:00"
	end := "07:00:00"
	settings := settingsRecord{
		QuietHoursEnabled: true,
		QuietHoursStart:   &start,
		QuietHoursEnd:     &end,
		Timezone:          "UTC",
	}

	if !inQuietHours(settings, time.Date(2026, 6, 14, 23, 0, 0, 0, time.UTC)) {
		t.Fatal("expected 23:00 UTC to be inside quiet hours")
	}
	if !inQuietHours(settings, time.Date(2026, 6, 14, 6, 30, 0, 0, time.UTC)) {
		t.Fatal("expected 06:30 UTC to be inside quiet hours")
	}
	if inQuietHours(settings, time.Date(2026, 6, 14, 12, 0, 0, 0, time.UTC)) {
		t.Fatal("expected 12:00 UTC to be outside quiet hours")
	}
}

func TestSafeDataFiltersPrivateFields(t *testing.T) {
	raw := []byte(`{
		"type": "CHAT_MESSAGE",
		"conversationUuid": "conversation-uuid",
		"matchUuid": "match-uuid",
		"messageUuid": "message-uuid",
		"senderName": "Ayesha",
		"body": "private text",
		"internalUserId": 42
	}`)

	data := safeData(raw)
	if data["type"] != TypeChatMessage {
		t.Fatalf("expected type to be preserved: %#v", data)
	}
	if _, ok := data["conversationUuid"]; !ok {
		t.Fatalf("expected conversationUuid to be preserved: %#v", data)
	}
	if _, ok := data["messageUuid"]; ok {
		t.Fatalf("messageUuid should not be exposed: %#v", data)
	}
	if _, ok := data["body"]; ok {
		t.Fatalf("body should not be exposed: %#v", data)
	}
}

func TestChatMessageUUIDFromDedupe(t *testing.T) {
	dedupe := "chat_message:7f2a43b5-bbd7-48a5-a7da-5c73a2dd3474:user-uuid"
	if got := chatMessageUUIDFromDedupe(&dedupe); got != "7f2a43b5-bbd7-48a5-a7da-5c73a2dd3474" {
		t.Fatalf("unexpected message uuid: %s", got)
	}

	invalid := "new_match:match:user"
	if got := chatMessageUUIDFromDedupe(&invalid); got != "" {
		t.Fatalf("expected invalid dedupe to be ignored, got %s", got)
	}
}

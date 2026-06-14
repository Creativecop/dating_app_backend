package chat

import (
	"context"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestNormalizeSendMessageRequest(t *testing.T) {
	result, err := normalizeSendMessageRequest("7f2a43b5-bbd7-48a5-a7da-5c73a2dd3474", SendMessageRequest{
		ClientMessageID: "7bec6f61-7420-46f7-a03f-392ed65f73ad",
		MessageType:     "text",
		Body:            "  Hey there  ",
	})
	if err != nil {
		t.Fatalf("expected valid message request: %v", err)
	}
	if result.MessageType != MessageTypeText || result.Body != "Hey there" {
		t.Fatalf("unexpected normalization: %#v", result)
	}

	if _, err := normalizeSendMessageRequest("7f2a43b5-bbd7-48a5-a7da-5c73a2dd3474", SendMessageRequest{
		ClientMessageID: "7bec6f61-7420-46f7-a03f-392ed65f73ad",
		MessageType:     "TEXT",
		Body:            strings.Repeat("x", 2001),
	}); err == nil {
		t.Fatal("expected oversized body to fail")
	}
}

func TestChatListCursorRoundTripWithNullLastMessage(t *testing.T) {
	original := chatListCursor{ConversationID: 42}
	encoded, err := encodeChatListCursor(original)
	if err != nil {
		t.Fatalf("encode cursor: %v", err)
	}
	decoded, err := decodeChatListCursor(encoded)
	if err != nil {
		t.Fatalf("decode cursor: %v", err)
	}
	if decoded == nil || !reflect.DeepEqual(*decoded, original) {
		t.Fatalf("unexpected decoded cursor: %#v", decoded)
	}
}

func TestMessageCursorRoundTrip(t *testing.T) {
	original := messageCursor{
		CreatedAt: time.Date(2026, 6, 14, 10, 30, 0, 0, time.UTC),
		MessageID: 99,
	}
	encoded, err := encodeMessageCursor(original)
	if err != nil {
		t.Fatalf("encode cursor: %v", err)
	}
	decoded, err := decodeMessageCursor(encoded)
	if err != nil {
		t.Fatalf("decode cursor: %v", err)
	}
	if decoded == nil || *decoded != original {
		t.Fatalf("unexpected decoded cursor: %#v", decoded)
	}
}

func TestDeliveryStateUsesReceiptsOnly(t *testing.T) {
	now := time.Now().UTC()
	if deliveryState(nil, nil) != DeliveryStateSent {
		t.Fatal("expected sent state")
	}
	if deliveryState(&now, nil) != DeliveryStateDelivered {
		t.Fatal("expected delivered state")
	}
	if deliveryState(&now, &now) != DeliveryStateSeen {
		t.Fatal("expected seen state")
	}
}

func TestVisibleBodyHidesDeletedMessageBody(t *testing.T) {
	body := "secret"
	if visibleBody(&body, MessageStatusDeleted) != nil {
		t.Fatal("expected deleted body to be hidden")
	}
	if visibleBody(&body, MessageStatusActive) == nil {
		t.Fatal("expected active body to be visible")
	}
}

func TestHubTypingThrottle(t *testing.T) {
	hub := NewHub()
	if !hub.AllowTypingEvent(1, 2, EventTypingStarted) {
		t.Fatal("expected first typing event to pass")
	}
	if hub.AllowTypingEvent(1, 2, EventTypingStarted) {
		t.Fatal("expected repeated typing event to be throttled")
	}
	if !hub.AllowTypingEvent(1, 2, EventTypingStopped) {
		t.Fatal("expected separate typing event type to pass")
	}
}

func TestHubSendToUserFanout(t *testing.T) {
	hub := NewHub()
	client := &Client{UserID: 10, send: make(chan Event, 1)}
	hub.Register(client)
	defer hub.Unregister(client)

	delivered := hub.SendToUser(10, Event{Event: EventConversation})
	if delivered != 1 {
		t.Fatalf("unexpected delivered count: %d", delivered)
	}
	select {
	case event := <-client.send:
		if event.Event != EventConversation {
			t.Fatalf("unexpected event: %#v", event)
		}
	default:
		t.Fatal("expected event in client channel")
	}
}

func TestErrorEventForServiceError(t *testing.T) {
	event := errorEventFor(idempotencyConflictError())
	if event.Event != EventError {
		t.Fatalf("unexpected event type: %s", event.Event)
	}
	data, ok := event.Data.(ErrorEventData)
	if !ok || data.Code != CodeIdempotencyKeyConflict {
		t.Fatalf("unexpected error event data: %#v", event.Data)
	}
}

func TestClientWritePumpStopsOnClosedChannel(t *testing.T) {
	client := &Client{UserID: 1, send: make(chan Event)}
	close(client.send)
	client.WritePump(context.Background())
}

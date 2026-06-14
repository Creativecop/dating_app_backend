package chat

import (
	"time"

	"github.com/google/uuid"
)

const (
	ConversationActive   = "ACTIVE"
	ConversationReadOnly = "READ_ONLY"
	ConversationClosed   = "CLOSED"

	MessageTypeText   = "TEXT"
	MessageTypeSystem = "SYSTEM"

	MessageStatusActive  = "ACTIVE"
	MessageStatusDeleted = "DELETED"

	DeliveryStateSent      = "SENT"
	DeliveryStateDelivered = "DELIVERED"
	DeliveryStateSeen      = "SEEN"

	EventMessageReceived  = "chat:message_received"
	EventMessageDelivered = "chat:message_delivered"
	EventMessageSeen      = "chat:message_seen"
	EventTypingStarted    = "chat:typing_started"
	EventTypingStopped    = "chat:typing_stopped"
	EventConversation     = "chat:conversation_updated"
	EventError            = "chat:error"

	ClientEventDeliveredAck = "chat:message_delivered_ack"
	ClientEventTypingStart  = "chat:typing_start"
	ClientEventTypingStop   = "chat:typing_stop"
	ClientEventMarkRead     = "chat:mark_read"
	ClientEventSendMessage  = "chat:send_message"
)

type Conversation struct {
	ID            uint64    `gorm:"primaryKey"`
	UUID          uuid.UUID `gorm:"type:uuid;uniqueIndex"`
	MatchID       uint64
	Status        string
	LastMessageID *uint64
	LastMessageAt *time.Time
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

func (Conversation) TableName() string {
	return "conversations"
}

type ConversationParticipant struct {
	ID                uint64    `gorm:"primaryKey"`
	UUID              uuid.UUID `gorm:"type:uuid;uniqueIndex"`
	ConversationID    uint64
	UserID            uint64
	LastReadMessageID *uint64
	LastReadAt        *time.Time
	MutedUntil        *time.Time
	HiddenAt          *time.Time
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

func (ConversationParticipant) TableName() string {
	return "conversation_participants"
}

type Message struct {
	ID              uint64    `gorm:"primaryKey"`
	UUID            uuid.UUID `gorm:"type:uuid;uniqueIndex"`
	ConversationID  uint64
	MatchID         uint64
	SenderUserID    uint64
	MessageType     string
	Body            *string
	ClientMessageID uuid.UUID `gorm:"type:uuid"`
	Status          string
	DeletedAt       *time.Time
	DeletedByUserID *uint64
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

func (Message) TableName() string {
	return "messages"
}

type MessageReceipt struct {
	ID          uint64    `gorm:"primaryKey"`
	UUID        uuid.UUID `gorm:"type:uuid;uniqueIndex"`
	MessageID   uint64
	UserID      uint64
	DeliveredAt *time.Time
	SeenAt      *time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func (MessageReceipt) TableName() string {
	return "message_receipts"
}

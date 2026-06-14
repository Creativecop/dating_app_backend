package chat

import "time"

type ChatListResponse struct {
	Items      []ChatListItem `json:"items"`
	NextCursor *string        `json:"nextCursor"`
}

type ChatListItem struct {
	ConversationUUID string           `json:"conversationUuid"`
	MatchUUID        string           `json:"matchUuid"`
	User             ChatUserPreview  `json:"user"`
	LastMessage      *MessageResponse `json:"lastMessage"`
	UnreadCount      int              `json:"unreadCount"`
}

type ChatUserPreview struct {
	UserUUID     string     `json:"userUuid"`
	DisplayName  *string    `json:"displayName"`
	PrimaryPhoto *ChatPhoto `json:"primaryPhoto"`
}

type ChatPhoto struct {
	MediaUUID    string `json:"mediaUuid"`
	ThumbnailURL string `json:"thumbnailUrl"`
}

type MessageListResponse struct {
	Items      []MessageResponse `json:"items"`
	NextCursor *string           `json:"nextCursor"`
}

type MessageResponse struct {
	MessageUUID     string     `json:"messageUuid"`
	ClientMessageID string     `json:"clientMessageId,omitempty"`
	SenderUserUUID  string     `json:"senderUserUuid"`
	MessageType     string     `json:"messageType"`
	Body            *string    `json:"body"`
	MessageStatus   string     `json:"messageStatus"`
	DeliveryState   string     `json:"deliveryState"`
	CreatedAt       time.Time  `json:"createdAt"`
	Deleted         bool       `json:"deleted"`
	DeletedAt       *time.Time `json:"deletedAt,omitempty"`
}

type SendMessageRequest struct {
	ClientMessageID string `json:"clientMessageId" binding:"required"`
	MessageType     string `json:"messageType" binding:"required"`
	Body            string `json:"body" binding:"required"`
}

type SendMessageResponse struct {
	MessageUUID     string    `json:"messageUuid"`
	ClientMessageID string    `json:"clientMessageId"`
	CreatedAt       time.Time `json:"createdAt"`
}

type MarkReadRequest struct {
	LastReadMessageUUID string `json:"lastReadMessageUuid" binding:"required"`
}

type MarkReadResponse struct {
	ConversationUUID    string    `json:"conversationUuid"`
	LastReadMessageUUID string    `json:"lastReadMessageUuid"`
	LastReadAt          time.Time `json:"lastReadAt"`
}

type DeleteMessageResponse struct {
	MessageUUID string `json:"messageUuid"`
	Status      string `json:"status"`
}

type DeliveredAckRequest struct {
	ConversationUUID string `json:"conversationUuid"`
	MessageUUID      string `json:"messageUuid"`
}

type TypingRequest struct {
	ConversationUUID string `json:"conversationUuid"`
}

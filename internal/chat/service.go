package chat

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Service struct {
	db            *gorm.DB
	repo          *Repository
	hub           *Hub
	notifications ChatNotificationDispatcher
}

type ChatNotificationDispatcher interface {
	NotifyChatMessage(ctx context.Context, messageUUID string) error
}

type conversationAccess struct {
	ConversationID   uint64
	ConversationUUID string
	MatchID          uint64
	MatchUUID        string
	UserUUID         string
	OtherUserID      uint64
	OtherUserUUID    string
}

type chatListRow struct {
	ConversationID      uint64
	ConversationUUID    string
	MatchUUID           string
	OtherUserUUID       string
	DisplayName         *string
	PrimaryMediaUUID    *string
	LastMessageID       *uint64
	LastMessageUUID     *string
	LastClientMessageID *string
	LastSenderUserUUID  *string
	LastMessageType     *string
	LastBody            *string
	LastMessageStatus   *string
	LastCreatedAt       *time.Time
	LastDeletedAt       *time.Time
	LastDeliveredAt     *time.Time
	LastSeenAt          *time.Time
	LastMessageAt       *time.Time
	UnreadCount         int
}

type messageRow struct {
	MessageID       uint64
	MessageUUID     string
	ClientMessageID string
	SenderUserID    uint64
	SenderUserUUID  string
	MessageType     string
	Body            *string
	MessageStatus   string
	CreatedAt       time.Time
	DeletedAt       *time.Time
	DeliveredAt     *time.Time
	SeenAt          *time.Time
}

type existingMessage struct {
	MessageID        uint64
	MessageUUID      string
	ConversationUUID string
	MessageType      string
	Body             *string
	MessageStatus    string
	ClientMessageID  string
	CreatedAt        time.Time
}

type normalizedSendMessage struct {
	ConversationUUID uuid.UUID
	ClientMessageID  uuid.UUID
	MessageType      string
	Body             string
}

func NewService(db *gorm.DB, hub *Hub) *Service {
	return &Service{db: db, repo: NewRepository(db), hub: hub}
}

func (s *Service) SetNotificationDispatcher(dispatcher ChatNotificationDispatcher) {
	s.notifications = dispatcher
}

func (s *Service) List(ctx context.Context, userID uint64, rawLimit string, rawCursor string) (*ChatListResponse, error) {
	limit, err := normalizeLimit(rawLimit, defaultChatListLimit, maxChatListLimit)
	if err != nil {
		return nil, err
	}
	cursor, err := decodeChatListCursor(rawCursor)
	if err != nil {
		return nil, err
	}

	rows, err := s.chatListRows(ctx, userID, limit+1, cursor)
	if err != nil {
		return nil, err
	}
	response := &ChatListResponse{Items: make([]ChatListItem, 0, minInt(limit, len(rows)))}
	visibleRows := rows
	if len(rows) > limit {
		visibleRows = rows[:limit]
		last := visibleRows[len(visibleRows)-1]
		next, err := encodeChatListCursor(chatListCursor{
			LastMessageAt:  last.LastMessageAt,
			ConversationID: last.ConversationID,
		})
		if err != nil {
			return nil, err
		}
		response.NextCursor = &next
	}
	for _, row := range visibleRows {
		response.Items = append(response.Items, row.toResponse())
	}
	return response, nil
}

func (s *Service) Messages(ctx context.Context, userID uint64, rawConversationUUID string, rawLimit string, rawCursor string) (*MessageListResponse, error) {
	conversationUUID, err := parseUUID(rawConversationUUID, "conversationUuid")
	if err != nil {
		return nil, err
	}
	access, err := s.activeConversationByUUID(ctx, userID, conversationUUID)
	if err != nil {
		return nil, err
	}
	if access == nil {
		return nil, notFoundError("Chat not found")
	}

	limit, err := normalizeLimit(rawLimit, defaultMessageLimit, maxMessageLimit)
	if err != nil {
		return nil, err
	}
	cursor, err := decodeMessageCursor(rawCursor)
	if err != nil {
		return nil, err
	}
	rows, err := s.messageRows(ctx, access.ConversationID, limit+1, cursor)
	if err != nil {
		return nil, err
	}
	response := &MessageListResponse{Items: make([]MessageResponse, 0, minInt(limit, len(rows)))}
	visibleRows := rows
	if len(rows) > limit {
		visibleRows = rows[:limit]
		last := visibleRows[len(visibleRows)-1]
		next, err := encodeMessageCursor(messageCursor{CreatedAt: last.CreatedAt, MessageID: last.MessageID})
		if err != nil {
			return nil, err
		}
		response.NextCursor = &next
	}
	for _, row := range visibleRows {
		response.Items = append(response.Items, row.toResponse())
	}
	return response, nil
}

func (s *Service) SendMessage(ctx context.Context, senderUserID uint64, rawConversationUUID string, req SendMessageRequest) (*SendMessageResponse, bool, error) {
	normalized, err := normalizeSendMessageRequest(rawConversationUUID, req)
	if err != nil {
		return nil, false, err
	}

	existing, err := s.messageByClientID(ctx, senderUserID, normalized.ClientMessageID)
	if err != nil {
		return nil, false, err
	}
	if existing != nil {
		if existing.ConversationUUID != normalized.ConversationUUID.String() ||
			existing.MessageType != normalized.MessageType ||
			derefString(existing.Body) != normalized.Body {
			return nil, false, idempotencyConflictError()
		}
		return &SendMessageResponse{
			MessageUUID:     existing.MessageUUID,
			ClientMessageID: existing.ClientMessageID,
			CreatedAt:       existing.CreatedAt,
		}, false, nil
	}

	var inserted messageRow
	var access *conversationAccess
	created := false
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var err error
		access, err = s.activeConversationByUUIDTx(ctx, tx, senderUserID, normalized.ConversationUUID, true)
		if err != nil {
			return err
		}
		if access == nil {
			return notFoundError("Chat not found")
		}

		if existing, err := s.messageByClientIDTx(ctx, tx, senderUserID, normalized.ClientMessageID); err != nil {
			return err
		} else if existing != nil {
			if existing.ConversationUUID != normalized.ConversationUUID.String() ||
				existing.MessageType != normalized.MessageType ||
				derefString(existing.Body) != normalized.Body {
				return idempotencyConflictError()
			}
			inserted = messageRow{
				MessageID:       existing.MessageID,
				MessageUUID:     existing.MessageUUID,
				ClientMessageID: existing.ClientMessageID,
				SenderUserID:    senderUserID,
				SenderUserUUID:  access.UserUUID,
				MessageType:     existing.MessageType,
				Body:            existing.Body,
				MessageStatus:   existing.MessageStatus,
				CreatedAt:       existing.CreatedAt,
			}
			return nil
		}

		now := time.Now().UTC()
		messageUUID := uuid.New()
		if err := tx.WithContext(ctx).Raw(`
			INSERT INTO messages (
			  uuid,
			  conversation_id,
			  match_id,
			  sender_user_id,
			  message_type,
			  body,
			  client_message_id,
			  status,
			  created_at,
			  updated_at
			)
			VALUES (?, ?, ?, ?, ?, ?, ?, 'ACTIVE', ?, ?)
			RETURNING id AS message_id, uuid::text AS message_uuid, client_message_id::text AS client_message_id, created_at
		`, messageUUID, access.ConversationID, access.MatchID, senderUserID, normalized.MessageType, normalized.Body, normalized.ClientMessageID, now, now).
			Scan(&inserted).Error; err != nil {
			return err
		}
		inserted.SenderUserID = senderUserID
		inserted.SenderUserUUID = access.UserUUID
		inserted.MessageType = normalized.MessageType
		inserted.Body = &normalized.Body
		inserted.MessageStatus = MessageStatusActive
		created = true

		if err := tx.WithContext(ctx).Exec(`
			INSERT INTO message_receipts (message_id, user_id, created_at, updated_at)
			VALUES (?, ?, ?, ?)
			ON CONFLICT (message_id, user_id) DO NOTHING
		`, inserted.MessageID, access.OtherUserID, now, now).Error; err != nil {
			return err
		}
		return tx.WithContext(ctx).Exec(`
			UPDATE conversations
			SET last_message_id = ?,
			    last_message_at = ?,
			    updated_at = ?
			WHERE id = ?
		`, inserted.MessageID, inserted.CreatedAt, now, access.ConversationID).Error
	})
	if err != nil {
		return nil, false, err
	}

	response := &SendMessageResponse{
		MessageUUID:     inserted.MessageUUID,
		ClientMessageID: inserted.ClientMessageID,
		CreatedAt:       inserted.CreatedAt,
	}
	if access != nil && inserted.MessageID != 0 {
		s.emitMessageCreated(context.Background(), *access, inserted)
		if created && s.notifications != nil {
			if err := s.notifications.NotifyChatMessage(context.Background(), inserted.MessageUUID); err != nil {
				log.Printf("notification_event=chat_message message_uuid=%s error=%v", inserted.MessageUUID, err)
			}
		}
	}
	return response, created, nil
}

func (s *Service) MarkRead(ctx context.Context, userID uint64, rawConversationUUID string, req MarkReadRequest) (*MarkReadResponse, error) {
	conversationUUID, err := parseUUID(rawConversationUUID, "conversationUuid")
	if err != nil {
		return nil, err
	}
	messageUUID, err := parseUUID(req.LastReadMessageUUID, "lastReadMessageUuid")
	if err != nil {
		return nil, err
	}

	var access *conversationAccess
	var target messageRow
	var readAt time.Time
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var err error
		access, err = s.activeConversationByUUIDTx(ctx, tx, userID, conversationUUID, false)
		if err != nil {
			return err
		}
		if access == nil {
			return notFoundError("Chat not found")
		}
		if err := tx.WithContext(ctx).Raw(`
			SELECT id AS message_id, uuid::text AS message_uuid, sender_user_id, created_at
			FROM messages
			WHERE uuid = ?
			  AND conversation_id = ?
		`, messageUUID, access.ConversationID).Scan(&target).Error; err != nil {
			return err
		}
		if target.MessageID == 0 {
			return notFoundError("Message not found")
		}

		var currentLastReadID sql.NullInt64
		if err := tx.WithContext(ctx).Raw(`
			SELECT last_read_message_id
			FROM conversation_participants
			WHERE conversation_id = ?
			  AND user_id = ?
			FOR UPDATE
		`, access.ConversationID, userID).Scan(&currentLastReadID).Error; err != nil {
			return err
		}

		readAt = time.Now().UTC()
		if !currentLastReadID.Valid || target.MessageID > uint64(currentLastReadID.Int64) {
			if err := tx.WithContext(ctx).Exec(`
				UPDATE conversation_participants
				SET last_read_message_id = ?,
				    last_read_at = ?,
				    updated_at = ?
				WHERE conversation_id = ?
				  AND user_id = ?
			`, target.MessageID, readAt, readAt, access.ConversationID, userID).Error; err != nil {
				return err
			}
		}
		return tx.WithContext(ctx).Exec(`
			UPDATE message_receipts mr
			SET seen_at = COALESCE(mr.seen_at, ?),
			    delivered_at = COALESCE(mr.delivered_at, ?),
			    updated_at = ?
			FROM messages m
			WHERE mr.message_id = m.id
			  AND mr.user_id = ?
			  AND m.conversation_id = ?
			  AND m.sender_user_id <> ?
			  AND m.id <= ?
		`, readAt, readAt, readAt, userID, access.ConversationID, userID, target.MessageID).Error
	})
	if err != nil {
		return nil, err
	}
	if access != nil {
		s.emitSeen(context.Background(), *access, target.MessageUUID)
	}
	return &MarkReadResponse{
		ConversationUUID:    conversationUUID.String(),
		LastReadMessageUUID: target.MessageUUID,
		LastReadAt:          readAt,
	}, nil
}

func (s *Service) DeleteMessage(ctx context.Context, userID uint64, rawMessageUUID string) (*DeleteMessageResponse, error) {
	messageUUID, err := parseUUID(rawMessageUUID, "messageUuid")
	if err != nil {
		return nil, err
	}

	var row messageRow
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.WithContext(ctx).Raw(`
			SELECT id AS message_id, uuid::text AS message_uuid, sender_user_id, status AS message_status, conversation_id
			FROM messages
			WHERE uuid = ?
		`, messageUUID).Scan(&row).Error; err != nil {
			return err
		}
		if row.MessageID == 0 {
			return notFoundError("Message not found")
		}
		if row.SenderUserID != userID {
			return forbiddenError("Cannot delete another user's message")
		}
		if row.MessageStatus == MessageStatusDeleted {
			return nil
		}
		now := time.Now().UTC()
		return tx.WithContext(ctx).Exec(`
			UPDATE messages
			SET status = 'DELETED',
			    body = NULL,
			    deleted_at = ?,
			    deleted_by_user_id = ?,
			    updated_at = ?
			WHERE id = ?
		`, now, userID, now, row.MessageID).Error
	})
	if err != nil {
		return nil, err
	}
	return &DeleteMessageResponse{MessageUUID: row.MessageUUID, Status: MessageStatusDeleted}, nil
}

func (s *Service) MarkDelivered(ctx context.Context, userID uint64, req DeliveredAckRequest) error {
	conversationUUID, err := parseUUID(req.ConversationUUID, "conversationUuid")
	if err != nil {
		return err
	}
	messageUUID, err := parseUUID(req.MessageUUID, "messageUuid")
	if err != nil {
		return err
	}

	var access *conversationAccess
	var message messageRow
	now := time.Now().UTC()
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var err error
		access, err = s.activeConversationByUUIDTx(ctx, tx, userID, conversationUUID, false)
		if err != nil {
			return err
		}
		if access == nil {
			return notFoundError("Chat not found")
		}
		if err := tx.WithContext(ctx).Raw(`
			SELECT m.id AS message_id, m.uuid::text AS message_uuid, m.sender_user_id, su.uuid::text AS sender_user_uuid, m.created_at
			FROM messages m
			JOIN users su ON su.id = m.sender_user_id
			WHERE m.uuid = ?
			  AND m.conversation_id = ?
			  AND m.status = 'ACTIVE'
		`, messageUUID, access.ConversationID).Scan(&message).Error; err != nil {
			return err
		}
		if message.MessageID == 0 {
			return notFoundError("Message not found")
		}
		if message.SenderUserID == userID {
			return validationError("Cannot acknowledge your own message", map[string]any{"field": "messageUuid"})
		}
		return tx.WithContext(ctx).Exec(`
			UPDATE message_receipts
			SET delivered_at = COALESCE(delivered_at, ?),
			    updated_at = ?
			WHERE message_id = ?
			  AND user_id = ?
		`, now, now, message.MessageID, userID).Error
	})
	if err != nil {
		return err
	}
	if access != nil && message.SenderUserID != 0 {
		s.emitToUser(message.SenderUserID, EventMessageDelivered, map[string]any{
			"conversationUuid": access.ConversationUUID,
			"messageUuid":      message.MessageUUID,
			"deliveredAt":      now,
		})
	}
	return nil
}

func (s *Service) Typing(ctx context.Context, userID uint64, rawConversationUUID string, started bool) error {
	conversationUUID, err := parseUUID(rawConversationUUID, "conversationUuid")
	if err != nil {
		return err
	}
	access, err := s.activeConversationByUUID(ctx, userID, conversationUUID)
	if err != nil {
		return err
	}
	if access == nil {
		return notFoundError("Chat not found")
	}
	event := EventTypingStopped
	if started {
		event = EventTypingStarted
	}
	if s.hub != nil && !s.hub.AllowTypingEvent(userID, access.ConversationID, event) {
		return nil
	}
	s.emitToUser(access.OtherUserID, event, map[string]any{
		"conversationUuid": access.ConversationUUID,
		"userUuid":         access.UserUUID,
	})
	return nil
}

func (s *Service) activeConversationByUUID(ctx context.Context, userID uint64, conversationUUID uuid.UUID) (*conversationAccess, error) {
	return s.activeConversationByUUIDTx(ctx, s.db.WithContext(ctx), userID, conversationUUID, false)
}

func (s *Service) activeConversationByUUIDTx(ctx context.Context, tx *gorm.DB, userID uint64, conversationUUID uuid.UUID, lock bool) (*conversationAccess, error) {
	lockClause := ""
	if lock {
		lockClause = " FOR UPDATE OF c"
	}
	var row conversationAccess
	err := tx.WithContext(ctx).Raw(`
SELECT
  c.id AS conversation_id,
  c.uuid::text AS conversation_uuid,
  m.id AS match_id,
  m.uuid::text AS match_uuid,
  current_u.uuid::text AS user_uuid,
  other_u.id AS other_user_id,
  other_u.uuid::text AS other_user_uuid
FROM conversations c
JOIN matches m ON m.id = c.match_id
JOIN conversation_participants cp
  ON cp.conversation_id = c.id
 AND cp.user_id = ?
 AND cp.hidden_at IS NULL
JOIN users current_u ON current_u.id = ?
JOIN users other_u
  ON other_u.id = CASE WHEN m.user_low_id = ? THEN m.user_high_id ELSE m.user_low_id END
JOIN conversation_participants other_cp
  ON other_cp.conversation_id = c.id
 AND other_cp.user_id = other_u.id
WHERE c.uuid = ?
  AND c.status = 'ACTIVE'
  AND m.status = 'ACTIVE'
  AND (m.user_low_id = ? OR m.user_high_id = ?)
  AND current_u.status = 'ACTIVE'
  AND current_u.deleted_at IS NULL
  AND other_u.status = 'ACTIVE'
  AND other_u.deleted_at IS NULL
  AND NOT EXISTS (
    SELECT 1
    FROM user_blocks ub
    WHERE (ub.blocker_user_id = ? AND ub.blocked_user_id = other_u.id)
       OR (ub.blocked_user_id = ? AND ub.blocker_user_id = other_u.id)
  )
`+lockClause, userID, userID, userID, conversationUUID, userID, userID, userID, userID).Scan(&row).Error
	if err != nil {
		return nil, err
	}
	if row.ConversationID == 0 {
		return nil, nil
	}
	return &row, nil
}

func (s *Service) messageByClientID(ctx context.Context, senderUserID uint64, clientMessageID uuid.UUID) (*existingMessage, error) {
	return s.messageByClientIDTx(ctx, s.db.WithContext(ctx), senderUserID, clientMessageID)
}

func (s *Service) messageByClientIDTx(ctx context.Context, tx *gorm.DB, senderUserID uint64, clientMessageID uuid.UUID) (*existingMessage, error) {
	var row existingMessage
	err := tx.WithContext(ctx).Raw(`
		SELECT
		  m.id AS message_id,
		  m.uuid::text AS message_uuid,
		  c.uuid::text AS conversation_uuid,
		  m.message_type,
		  m.body,
		  m.status AS message_status,
		  m.client_message_id::text AS client_message_id,
		  m.created_at
		FROM messages m
		JOIN conversations c ON c.id = m.conversation_id
		WHERE m.sender_user_id = ?
		  AND m.client_message_id = ?
	`, senderUserID, clientMessageID).Scan(&row).Error
	if err != nil {
		return nil, err
	}
	if row.MessageID == 0 {
		return nil, nil
	}
	return &row, nil
}

func (s *Service) chatListRows(ctx context.Context, userID uint64, limit int, cursor *chatListCursor) ([]chatListRow, error) {
	hasCursor := cursor != nil
	cursorHasTime := false
	cursorTime := time.Time{}
	cursorConversationID := uint64(0)
	if cursor != nil {
		cursorHasTime = cursor.LastMessageAt != nil
		if cursor.LastMessageAt != nil {
			cursorTime = *cursor.LastMessageAt
		}
		cursorConversationID = cursor.ConversationID
	}
	var rows []chatListRow
	err := s.db.WithContext(ctx).Raw(`
SELECT
  c.id AS conversation_id,
  c.uuid::text AS conversation_uuid,
  m.uuid::text AS match_uuid,
  other_u.uuid::text AS other_user_uuid,
  p.display_name,
  pm.uuid::text AS primary_media_uuid,
  lm.id AS last_message_id,
  lm.uuid::text AS last_message_uuid,
  lm.client_message_id::text AS last_client_message_id,
  sender_u.uuid::text AS last_sender_user_uuid,
  lm.message_type AS last_message_type,
  lm.body AS last_body,
  lm.status AS last_message_status,
  lm.created_at AS last_created_at,
  lm.deleted_at AS last_deleted_at,
  lmr.delivered_at AS last_delivered_at,
  lmr.seen_at AS last_seen_at,
  c.last_message_at,
  (
    SELECT COUNT(*)
    FROM messages unread
    WHERE unread.conversation_id = c.id
      AND unread.sender_user_id <> ?
      AND unread.status = 'ACTIVE'
      AND (cp.last_read_message_id IS NULL OR unread.id > cp.last_read_message_id)
  ) AS unread_count
FROM conversation_participants cp
JOIN conversations c ON c.id = cp.conversation_id
JOIN matches m ON m.id = c.match_id
JOIN users other_u
  ON other_u.id = CASE WHEN m.user_low_id = ? THEN m.user_high_id ELSE m.user_low_id END
LEFT JOIN profiles p ON p.user_id = other_u.id
LEFT JOIN user_media pm ON pm.user_id = other_u.id
  AND pm.media_purpose = 'PROFILE_PHOTO'
  AND pm.processing_status = 'READY'
  AND pm.moderation_status = 'APPROVED'
  AND pm.is_primary = TRUE
  AND pm.deleted_at IS NULL
  AND EXISTS (
    SELECT 1
    FROM user_media_variants thumbnail_variant
    WHERE thumbnail_variant.media_id = pm.id
      AND thumbnail_variant.variant_type = 'THUMBNAIL'
  )
LEFT JOIN messages lm ON lm.id = c.last_message_id
LEFT JOIN users sender_u ON sender_u.id = lm.sender_user_id
LEFT JOIN message_receipts lmr ON lmr.message_id = lm.id AND lmr.user_id <> lm.sender_user_id
WHERE cp.user_id = ?
  AND cp.hidden_at IS NULL
  AND c.status = 'ACTIVE'
  AND m.status = 'ACTIVE'
  AND (m.user_low_id = ? OR m.user_high_id = ?)
  AND other_u.status = 'ACTIVE'
  AND other_u.deleted_at IS NULL
  AND NOT EXISTS (
    SELECT 1
    FROM user_blocks ub
    WHERE (ub.blocker_user_id = ? AND ub.blocked_user_id = other_u.id)
       OR (ub.blocked_user_id = ? AND ub.blocker_user_id = other_u.id)
  )
  AND (
    ? = FALSE
    OR (
      ? = TRUE
      AND (
        c.last_message_at < ?
        OR (c.last_message_at = ? AND c.id < ?)
        OR c.last_message_at IS NULL
      )
    )
    OR (
      ? = FALSE
      AND c.last_message_at IS NULL
      AND c.id < ?
    )
  )
ORDER BY c.last_message_at DESC NULLS LAST, c.id DESC
LIMIT ?
`, userID, userID, userID, userID, userID, userID, userID, hasCursor, cursorHasTime, cursorTime, cursorTime, cursorConversationID, cursorHasTime, cursorConversationID, limit).Scan(&rows).Error
	return rows, err
}

func (s *Service) messageRows(ctx context.Context, conversationID uint64, limit int, cursor *messageCursor) ([]messageRow, error) {
	hasCursor := cursor != nil
	cursorCreatedAt := time.Time{}
	cursorMessageID := uint64(0)
	if cursor != nil {
		cursorCreatedAt = cursor.CreatedAt
		cursorMessageID = cursor.MessageID
	}
	var rows []messageRow
	err := s.db.WithContext(ctx).Raw(`
SELECT
  m.id AS message_id,
  m.uuid::text AS message_uuid,
  m.client_message_id::text AS client_message_id,
  m.sender_user_id,
  sender_u.uuid::text AS sender_user_uuid,
  m.message_type,
  m.body,
  m.status AS message_status,
  m.created_at,
  m.deleted_at,
  mr.delivered_at,
  mr.seen_at
FROM messages m
JOIN users sender_u ON sender_u.id = m.sender_user_id
LEFT JOIN message_receipts mr ON mr.message_id = m.id AND mr.user_id <> m.sender_user_id
WHERE m.conversation_id = ?
  AND (
    ? = FALSE
    OR m.created_at < ?
    OR (m.created_at = ? AND m.id < ?)
  )
ORDER BY m.created_at DESC, m.id DESC
LIMIT ?
`, conversationID, hasCursor, cursorCreatedAt, cursorCreatedAt, cursorMessageID, limit).Scan(&rows).Error
	return rows, err
}

func normalizeSendMessageRequest(rawConversationUUID string, req SendMessageRequest) (normalizedSendMessage, error) {
	conversationUUID, err := parseUUID(rawConversationUUID, "conversationUuid")
	if err != nil {
		return normalizedSendMessage{}, err
	}
	clientID, err := parseUUID(req.ClientMessageID, "clientMessageId")
	if err != nil {
		return normalizedSendMessage{}, err
	}
	messageType := strings.ToUpper(strings.TrimSpace(req.MessageType))
	if messageType != MessageTypeText {
		return normalizedSendMessage{}, validationError("Only text messages are supported", map[string]any{"field": "messageType"})
	}
	body := strings.TrimSpace(req.Body)
	if body == "" || len(body) > 2000 {
		return normalizedSendMessage{}, validationError("Message body must be 1 to 2000 characters", map[string]any{"field": "body"})
	}
	return normalizedSendMessage{
		ConversationUUID: conversationUUID,
		ClientMessageID:  clientID,
		MessageType:      messageType,
		Body:             body,
	}, nil
}

func parseUUID(raw string, field string) (uuid.UUID, error) {
	parsed, err := uuid.Parse(strings.TrimSpace(raw))
	if err != nil {
		return uuid.UUID{}, validationError(field+" is invalid", map[string]any{"field": field})
	}
	return parsed, nil
}

func (row chatListRow) toResponse() ChatListItem {
	var photo *ChatPhoto
	if row.PrimaryMediaUUID != nil && strings.TrimSpace(*row.PrimaryMediaUUID) != "" {
		photo = &ChatPhoto{
			MediaUUID:    *row.PrimaryMediaUUID,
			ThumbnailURL: "/api/v1/media/" + *row.PrimaryMediaUUID + "/thumbnail",
		}
	}
	var lastMessage *MessageResponse
	if row.LastMessageID != nil && row.LastMessageUUID != nil && row.LastSenderUserUUID != nil && row.LastMessageType != nil && row.LastMessageStatus != nil && row.LastCreatedAt != nil {
		lastMessage = &MessageResponse{
			MessageUUID:     *row.LastMessageUUID,
			ClientMessageID: derefStringPtr(row.LastClientMessageID),
			SenderUserUUID:  *row.LastSenderUserUUID,
			MessageType:     *row.LastMessageType,
			Body:            visibleBody(row.LastBody, *row.LastMessageStatus),
			MessageStatus:   *row.LastMessageStatus,
			DeliveryState:   deliveryState(row.LastDeliveredAt, row.LastSeenAt),
			CreatedAt:       *row.LastCreatedAt,
			Deleted:         *row.LastMessageStatus == MessageStatusDeleted,
			DeletedAt:       row.LastDeletedAt,
		}
	}
	return ChatListItem{
		ConversationUUID: row.ConversationUUID,
		MatchUUID:        row.MatchUUID,
		User: ChatUserPreview{
			UserUUID:     row.OtherUserUUID,
			DisplayName:  row.DisplayName,
			PrimaryPhoto: photo,
		},
		LastMessage: lastMessage,
		UnreadCount: row.UnreadCount,
	}
}

func (row messageRow) toResponse() MessageResponse {
	return MessageResponse{
		MessageUUID:     row.MessageUUID,
		ClientMessageID: row.ClientMessageID,
		SenderUserUUID:  row.SenderUserUUID,
		MessageType:     row.MessageType,
		Body:            visibleBody(row.Body, row.MessageStatus),
		MessageStatus:   row.MessageStatus,
		DeliveryState:   deliveryState(row.DeliveredAt, row.SeenAt),
		CreatedAt:       row.CreatedAt,
		Deleted:         row.MessageStatus == MessageStatusDeleted,
		DeletedAt:       row.DeletedAt,
	}
}

func visibleBody(body *string, status string) *string {
	if status == MessageStatusDeleted {
		return nil
	}
	return body
}

func deliveryState(deliveredAt *time.Time, seenAt *time.Time) string {
	if seenAt != nil {
		return DeliveryStateSeen
	}
	if deliveredAt != nil {
		return DeliveryStateDelivered
	}
	return DeliveryStateSent
}

func derefString(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func derefStringPtr(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func minInt(a int, b int) int {
	if a < b {
		return a
	}
	return b
}

func (s *Service) emitMessageCreated(ctx context.Context, access conversationAccess, message messageRow) {
	s.emitToUser(access.OtherUserID, EventMessageReceived, map[string]any{
		"conversationUuid": access.ConversationUUID,
		"message":          message.toResponse(),
	})
	s.emitToUser(access.OtherUserID, EventConversation, map[string]any{
		"conversationUuid": access.ConversationUUID,
	})
	s.emitToUser(message.SenderUserID, EventConversation, map[string]any{
		"conversationUuid": access.ConversationUUID,
	})
	_ = ctx
}

func (s *Service) emitSeen(ctx context.Context, access conversationAccess, messageUUID string) {
	s.emitToUser(access.OtherUserID, EventMessageSeen, map[string]any{
		"conversationUuid":    access.ConversationUUID,
		"lastReadMessageUuid": messageUUID,
		"seenByUserUuid":      access.UserUUID,
	})
	_ = ctx
}

func (s *Service) emitToUser(userID uint64, event string, data any) {
	if s.hub == nil {
		return
	}
	s.hub.SendToUser(userID, Event{Event: event, Data: data})
}

func IsNotFound(err error) bool {
	serviceErr, ok := AsServiceError(err)
	return ok && serviceErr.Code == CodeNotFound
}

func IsForbidden(err error) bool {
	serviceErr, ok := AsServiceError(err)
	return ok && serviceErr.Code == CodeForbidden
}

func IsValidation(err error) bool {
	serviceErr, ok := AsServiceError(err)
	return ok && serviceErr.Code == CodeValidation
}

func IsDuplicateKey(err error) bool {
	return errors.Is(err, gorm.ErrDuplicatedKey)
}

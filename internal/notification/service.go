package notification

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	defaultNotificationLimit = 20
	maxNotificationLimit     = 50
)

type Service struct {
	db          *gorm.DB
	queueClient *asynq.Client
	cfg         NotificationConfig
	provider    Provider
}

type NotificationConfig interface {
	GetPushEnabled() bool
	GetProvider() string
	GetPushMaxRetry() int
	GetPushTimeout() time.Duration
	GetPushGrace() time.Duration
	GetDefaultTimezone() string
}

type configAdapter struct {
	pushEnabled     bool
	provider        string
	pushMaxRetry    int
	pushTimeout     time.Duration
	pushGrace       time.Duration
	defaultTimezone string
}

func (c configAdapter) GetPushEnabled() bool {
	return c.pushEnabled
}

func (c configAdapter) GetProvider() string {
	return c.provider
}

func (c configAdapter) GetPushMaxRetry() int {
	return c.pushMaxRetry
}

func (c configAdapter) GetPushTimeout() time.Duration {
	return c.pushTimeout
}

func (c configAdapter) GetPushGrace() time.Duration {
	return c.pushGrace
}

func (c configAdapter) GetDefaultTimezone() string {
	return c.defaultTimezone
}

type createNotificationInput struct {
	UserID           uint64
	NotificationType string
	Title            string
	Body             string
	Data             map[string]any
	DedupeKey        string
}

type notificationRecord struct {
	ID               uint64
	UUID             string
	UserID           uint64
	NotificationType string
	Title            string
	Body             string
	DataRaw          []byte `gorm:"column:data"`
	DedupeKey        *string
	ReadAt           *time.Time
	ClickedAt        *time.Time
	CreatedAt        time.Time
}

type settingsRecord struct {
	ID                 uint64
	UUID               string
	UserID             uint64
	PushEnabled        bool
	NewMatchEnabled    bool
	ChatMessageEnabled bool
	SuperLikeEnabled   bool
	QuietHoursEnabled  bool
	QuietHoursStart    *string
	QuietHoursEnd      *string
	Timezone           string
}

type deviceRecord struct {
	ID       uint64
	UserID   uint64
	DeviceID string
	FCMToken string
}

func NewService(db *gorm.DB, queueClient *asynq.Client, cfg configAdapter) *Service {
	return &Service{
		db:          db,
		queueClient: queueClient,
		cfg:         cfg,
		provider:    NewNoopProvider(),
	}
}

func NewConfigAdapter(pushEnabled bool, provider string, maxRetry int, timeoutSeconds int, graceSeconds int, defaultTimezone string) configAdapter {
	return configAdapter{
		pushEnabled:     pushEnabled,
		provider:        strings.ToLower(strings.TrimSpace(provider)),
		pushMaxRetry:    maxRetry,
		pushTimeout:     time.Duration(timeoutSeconds) * time.Second,
		pushGrace:       time.Duration(graceSeconds) * time.Second,
		defaultTimezone: strings.TrimSpace(defaultTimezone),
	}
}

func (s *Service) SetProvider(provider Provider) {
	if provider != nil {
		s.provider = provider
	}
}

func (s *Service) EnsureNotificationSettings(ctx context.Context, userID uint64) error {
	now := time.Now().UTC()
	settings := NotificationSettings{
		UUID:               uuid.New(),
		UserID:             userID,
		PushEnabled:        true,
		NewMatchEnabled:    true,
		ChatMessageEnabled: true,
		SuperLikeEnabled:   true,
		QuietHoursEnabled:  false,
		Timezone:           s.defaultTimezone(),
		CreatedAt:          now,
		UpdatedAt:          now,
	}
	return s.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_id"}},
		DoNothing: true,
	}).Create(&settings).Error
}

func (s *Service) UpsertFCMToken(ctx context.Context, userID uint64, req UpsertFCMTokenRequest) (*DeviceTokenResponse, error) {
	normalized, err := normalizeTokenRequest(req)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := clearTokenFromOtherDevices(ctx, tx, userID, normalized.DeviceID, normalized.FCMToken, now); err != nil {
			return err
		}
		device := Device{
			UUID:              uuid.New(),
			UserID:            userID,
			DeviceID:          normalized.DeviceID,
			DeviceName:        normalized.DeviceName,
			Platform:          normalized.Platform,
			FCMToken:          &normalized.FCMToken,
			PushEnabled:       true,
			FCMTokenUpdatedAt: &now,
			AppVersion:        normalized.AppVersion,
			OSVersion:         normalized.OSVersion,
			LastActiveAt:      &now,
			CreatedAt:         now,
			UpdatedAt:         now,
		}
		return tx.WithContext(ctx).Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "user_id"}, {Name: "device_id"}},
			DoUpdates: clause.Assignments(map[string]any{
				"device_name":          device.DeviceName,
				"platform":             device.Platform,
				"fcm_token":            device.FCMToken,
				"push_enabled":         true,
				"fcm_token_updated_at": now,
				"push_failure_count":   0,
				"app_version":          device.AppVersion,
				"os_version":           device.OSVersion,
				"last_active_at":       now,
				"updated_at":           now,
			}),
		}).Create(&device).Error
	})
	if err != nil {
		return nil, err
	}
	return &DeviceTokenResponse{DeviceID: normalized.DeviceID, PushEnabled: true, FCMTokenUpdatedAt: &now}, nil
}

func (s *Service) DeleteFCMToken(ctx context.Context, userID uint64, req DeleteFCMTokenRequest) (*DeviceTokenResponse, error) {
	deviceID := strings.TrimSpace(req.DeviceID)
	if deviceID == "" {
		return nil, validationError("deviceId is required", map[string]any{"field": "deviceId"})
	}
	now := time.Now().UTC()
	if err := s.db.WithContext(ctx).Model(&Device{}).
		Where("user_id = ? AND device_id = ?", userID, deviceID).
		Updates(map[string]any{
			"fcm_token":    nil,
			"push_enabled": false,
			"updated_at":   now,
		}).Error; err != nil {
		return nil, err
	}
	return &DeviceTokenResponse{DeviceID: deviceID, PushEnabled: false}, nil
}

func (s *Service) GetSettings(ctx context.Context, userID uint64) (*SettingsResponse, error) {
	if err := s.EnsureNotificationSettings(ctx, userID); err != nil {
		return nil, err
	}
	row, err := s.settingsByUser(ctx, s.db.WithContext(ctx), userID)
	if err != nil {
		return nil, err
	}
	response := settingsResponse(*row)
	return &response, nil
}

func (s *Service) UpdateSettings(ctx context.Context, userID uint64, req UpdateSettingsRequest) (*SettingsResponse, error) {
	normalized, err := normalizeSettingsRequest(req, s.defaultTimezone())
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	if err := s.EnsureNotificationSettings(ctx, userID); err != nil {
		return nil, err
	}
	if err := s.db.WithContext(ctx).Model(&NotificationSettings{}).
		Where("user_id = ?", userID).
		Updates(map[string]any{
			"push_enabled":         normalized.PushEnabled,
			"new_match_enabled":    normalized.NewMatchEnabled,
			"chat_message_enabled": normalized.ChatMessageEnabled,
			"super_like_enabled":   normalized.SuperLikeEnabled,
			"quiet_hours_enabled":  normalized.QuietHoursEnabled,
			"quiet_hours_start":    normalized.QuietHoursStart,
			"quiet_hours_end":      normalized.QuietHoursEnd,
			"timezone":             normalized.Timezone,
			"updated_at":           now,
		}).Error; err != nil {
		return nil, err
	}
	return s.GetSettings(ctx, userID)
}

func (s *Service) ListNotifications(ctx context.Context, userID uint64, limit int, rawCursor string) (*NotificationListResponse, error) {
	if limit <= 0 {
		limit = defaultNotificationLimit
	}
	if limit > maxNotificationLimit {
		limit = maxNotificationLimit
	}
	cursor, err := decodeCursor(rawCursor)
	if err != nil {
		return nil, err
	}
	args := []any{userID}
	cursorSQL := ""
	if cursor != nil {
		cursorSQL = "AND (created_at < ? OR (created_at = ? AND id < ?))"
		args = append(args, cursor.CreatedAt, cursor.CreatedAt, cursor.NotificationID)
	}
	args = append(args, limit+1)

	var rows []notificationRecord
	if err := s.db.WithContext(ctx).Raw(`
		SELECT id, uuid::text AS uuid, user_id, notification_type, title, body, data, read_at, clicked_at, dedupe_key, created_at
		FROM notifications
		WHERE user_id = ?
		`+cursorSQL+`
		ORDER BY created_at DESC, id DESC
		LIMIT ?
	`, args...).Scan(&rows).Error; err != nil {
		return nil, err
	}

	response := &NotificationListResponse{}
	visible := rows
	if len(rows) > limit {
		visible = rows[:limit]
		last := visible[len(visible)-1]
		next, err := encodeCursor(notificationCursor{CreatedAt: last.CreatedAt, NotificationID: last.ID})
		if err != nil {
			return nil, err
		}
		response.NextCursor = &next
	}
	for _, row := range visible {
		response.Items = append(response.Items, notificationResponse(row))
	}
	return response, nil
}

func (s *Service) MarkRead(ctx context.Context, userID uint64, rawNotificationUUID string) (*MarkReadResponse, error) {
	notificationUUID, err := uuid.Parse(strings.TrimSpace(rawNotificationUUID))
	if err != nil {
		return nil, validationError("notificationUuid is invalid", map[string]any{"field": "notificationUuid"})
	}
	now := time.Now().UTC()
	var row notificationRecord
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.WithContext(ctx).Raw(`
			SELECT id, uuid::text AS uuid, read_at
			FROM notifications
			WHERE uuid = ? AND user_id = ?
			FOR UPDATE
		`, notificationUUID, userID).Scan(&row).Error; err != nil {
			return err
		}
		if row.ID == 0 {
			return notFoundError("Notification not found")
		}
		if row.ReadAt != nil {
			return nil
		}
		return tx.WithContext(ctx).Model(&Notification{}).
			Where("id = ?", row.ID).
			Update("read_at", now).Error
	})
	if err != nil {
		return nil, err
	}
	if row.ReadAt == nil {
		row.ReadAt = &now
	}
	return &MarkReadResponse{NotificationUUID: row.UUID, ReadAt: row.ReadAt}, nil
}

func (s *Service) MarkAllRead(ctx context.Context, userID uint64) (*MarkAllReadResponse, error) {
	now := time.Now().UTC()
	result := s.db.WithContext(ctx).Model(&Notification{}).
		Where("user_id = ? AND read_at IS NULL", userID).
		Update("read_at", now)
	if result.Error != nil {
		return nil, result.Error
	}
	return &MarkAllReadResponse{Updated: result.RowsAffected, ReadAt: now}, nil
}

func (s *Service) NotifyNewMatch(ctx context.Context, matchUUID string) error {
	type row struct {
		MatchUUID string
		UserID    uint64
		UserUUID  string
	}
	var rows []row
	if err := s.db.WithContext(ctx).Raw(`
		SELECT m.uuid::text AS match_uuid, u.id AS user_id, u.uuid::text AS user_uuid
		FROM matches m
		JOIN users u ON u.id IN (m.user_low_id, m.user_high_id)
		WHERE m.uuid = ?
	`, matchUUID).Scan(&rows).Error; err != nil {
		return err
	}
	for _, item := range rows {
		notification, err := s.createOrGetNotification(ctx, createNotificationInput{
			UserID:           item.UserID,
			NotificationType: TypeNewMatch,
			Title:            "It's a match!",
			Body:             "You have a new match.",
			Data: map[string]any{
				"type":      TypeNewMatch,
				"matchUuid": item.MatchUUID,
			},
			DedupeKey: fmt.Sprintf("new_match:%s:%s", item.MatchUUID, item.UserUUID),
		})
		if err != nil {
			return err
		}
		if err := s.enqueuePush(ctx, notification, "new_match", 0); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) NotifyChatMessage(ctx context.Context, messageUUID string) error {
	type row struct {
		MessageUUID      string
		ConversationUUID string
		ReceiverUserID   uint64
		ReceiverUserUUID string
	}
	var item row
	if err := s.db.WithContext(ctx).Raw(`
		SELECT
		  m.uuid::text AS message_uuid,
		  c.uuid::text AS conversation_uuid,
		  mr.user_id AS receiver_user_id,
		  u.uuid::text AS receiver_user_uuid
		FROM messages m
		JOIN conversations c ON c.id = m.conversation_id
		JOIN message_receipts mr ON mr.message_id = m.id
		JOIN users u ON u.id = mr.user_id
		WHERE m.uuid = ?
		  AND m.status = 'ACTIVE'
	`, messageUUID).Scan(&item).Error; err != nil {
		return err
	}
	if item.ReceiverUserID == 0 {
		return nil
	}
	notification, err := s.createOrGetNotification(ctx, createNotificationInput{
		UserID:           item.ReceiverUserID,
		NotificationType: TypeChatMessage,
		Title:            "New message",
		Body:             "You received a new message.",
		Data: map[string]any{
			"type":             TypeChatMessage,
			"conversationUuid": item.ConversationUUID,
		},
		DedupeKey: fmt.Sprintf("chat_message:%s:%s", item.MessageUUID, item.ReceiverUserUUID),
	})
	if err != nil {
		return err
	}
	return s.enqueuePush(ctx, notification, "chat_message", s.cfg.GetPushGrace())
}

func (s *Service) ProcessSendPush(ctx context.Context, payload SendPushPayload) error {
	notification, err := s.notificationByUUID(ctx, payload.NotificationUUID, payload.UserID)
	if err != nil {
		return err
	}
	if notification == nil {
		return nil
	}
	providerName := s.provider.Name()
	if providerName == "" {
		providerName = strings.ToUpper(s.cfg.GetProvider())
	}

	if !s.cfg.GetPushEnabled() {
		return s.logSkipped(ctx, notification, nil, providerName, "PUSH_DISABLED", "Push delivery is disabled.")
	}

	settings, err := s.settingsByUser(ctx, s.db.WithContext(ctx), notification.UserID)
	if errors.Is(err, gorm.ErrRecordNotFound) || settings == nil {
		if err := s.EnsureNotificationSettings(ctx, notification.UserID); err != nil {
			return err
		}
		settings, err = s.settingsByUser(ctx, s.db.WithContext(ctx), notification.UserID)
	}
	if err != nil {
		return err
	}
	if skipCode := pushSkipCode(*settings, notification.NotificationType); skipCode != "" {
		return s.logSkipped(ctx, notification, nil, providerName, skipCode, "Push skipped by notification settings.")
	}
	if inQuietHours(*settings, time.Now().UTC()) {
		return s.logSkipped(ctx, notification, nil, providerName, "QUIET_HOURS", "Push skipped during quiet hours.")
	}
	if notification.NotificationType == TypeChatMessage {
		delivered, err := s.chatMessageDeliveredOrSeen(ctx, notification)
		if err != nil {
			return err
		}
		if delivered {
			return s.logSkipped(ctx, notification, nil, providerName, "CHAT_ALREADY_DELIVERED", "Chat message was already delivered or seen.")
		}
	}

	devices, err := s.activeDevices(ctx, notification.UserID)
	if err != nil {
		return err
	}
	if len(devices) == 0 {
		return s.logSkipped(ctx, notification, nil, providerName, "NO_DEVICES", "No active push devices.")
	}

	var retryErr error
	for _, device := range devices {
		alreadySent, err := s.sentLogExists(ctx, notification.ID, device.ID, providerName)
		if err != nil {
			return err
		}
		if alreadySent {
			continue
		}
		result, err := s.provider.Send(ctx, PushMessage{
			Token: device.FCMToken,
			Title: notification.Title,
			Body:  notification.Body,
			Data:  pushPayloadData(notification),
		})
		if err == nil {
			messageID := ""
			if result != nil {
				messageID = result.MessageID
			}
			if err := s.logSent(ctx, notification, device, providerName, messageID); err != nil {
				return err
			}
			continue
		}

		providerErr := providerError(err)
		if providerErr.InvalidToken {
			if updateErr := s.markDeviceInvalid(ctx, device.ID, providerErr); updateErr != nil {
				return updateErr
			}
		} else if providerErr.Temporary && retryErr == nil {
			retryErr = err
		}
		if logErr := s.logFailed(ctx, notification, device, providerName, providerErr); logErr != nil {
			return logErr
		}
	}
	return retryErr
}

func (s *Service) createOrGetNotification(ctx context.Context, input createNotificationInput) (*notificationRecord, error) {
	data, err := json.Marshal(input.Data)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	var row notificationRecord
	if strings.TrimSpace(input.DedupeKey) != "" {
		err = s.db.WithContext(ctx).Raw(`
			INSERT INTO notifications (
			  uuid,
			  user_id,
			  notification_type,
			  title,
			  body,
			  data,
			  dedupe_key,
			  created_at
			)
			VALUES (?, ?, ?, ?, ?, ?::jsonb, ?, ?)
			ON CONFLICT (user_id, dedupe_key) WHERE dedupe_key IS NOT NULL
			DO UPDATE SET dedupe_key = notifications.dedupe_key
			RETURNING id, uuid::text AS uuid, user_id, notification_type, title, body, data, dedupe_key, read_at, clicked_at, created_at
		`, uuid.New(), input.UserID, input.NotificationType, input.Title, input.Body, string(data), input.DedupeKey, now).Scan(&row).Error
	} else {
		err = s.db.WithContext(ctx).Raw(`
			INSERT INTO notifications (
			  uuid,
			  user_id,
			  notification_type,
			  title,
			  body,
			  data,
			  created_at
			)
			VALUES (?, ?, ?, ?, ?, ?::jsonb, ?)
			RETURNING id, uuid::text AS uuid, user_id, notification_type, title, body, data, dedupe_key, read_at, clicked_at, created_at
		`, uuid.New(), input.UserID, input.NotificationType, input.Title, input.Body, string(data), now).Scan(&row).Error
	}
	if err != nil {
		return nil, err
	}
	if row.ID == 0 {
		return nil, nil
	}
	return &row, nil
}

func (s *Service) enqueuePush(ctx context.Context, notification *notificationRecord, kind string, delay time.Duration) error {
	if notification == nil || s.queueClient == nil {
		return nil
	}
	task, err := NewSendPushTask(SendPushPayload{NotificationUUID: notification.UUID, UserID: notification.UserID})
	if err != nil {
		return err
	}
	_, err = s.queueClient.EnqueueContext(
		ctx,
		task,
		PushTaskOptions(PushTaskID(kind, notification.UUID), s.cfg.GetPushMaxRetry(), s.cfg.GetPushTimeout(), delay)...,
	)
	if errors.Is(err, asynq.ErrTaskIDConflict) || errors.Is(err, asynq.ErrDuplicateTask) {
		return nil
	}
	return err
}

func (s *Service) notificationByUUID(ctx context.Context, notificationUUID string, userID uint64) (*notificationRecord, error) {
	var row notificationRecord
	err := s.db.WithContext(ctx).Raw(`
		SELECT id, uuid::text AS uuid, user_id, notification_type, title, body, data, dedupe_key, read_at, clicked_at, created_at
		FROM notifications
		WHERE uuid = ?
		  AND user_id = ?
	`, notificationUUID, userID).Scan(&row).Error
	if err != nil {
		return nil, err
	}
	if row.ID == 0 {
		return nil, nil
	}
	return &row, nil
}

func (s *Service) settingsByUser(ctx context.Context, tx *gorm.DB, userID uint64) (*settingsRecord, error) {
	var row settingsRecord
	err := tx.WithContext(ctx).Raw(`
		SELECT
		  id,
		  uuid::text AS uuid,
		  user_id,
		  push_enabled,
		  new_match_enabled,
		  chat_message_enabled,
		  super_like_enabled,
		  quiet_hours_enabled,
		  quiet_hours_start::text AS quiet_hours_start,
		  quiet_hours_end::text AS quiet_hours_end,
		  timezone
		FROM notification_settings
		WHERE user_id = ?
	`, userID).Scan(&row).Error
	if err != nil {
		return nil, err
	}
	if row.ID == 0 {
		return nil, gorm.ErrRecordNotFound
	}
	return &row, nil
}

func (s *Service) activeDevices(ctx context.Context, userID uint64) ([]deviceRecord, error) {
	var devices []deviceRecord
	err := s.db.WithContext(ctx).Raw(`
		SELECT id, user_id, device_id, fcm_token
		FROM devices
		WHERE user_id = ?
		  AND push_enabled = TRUE
		  AND fcm_token IS NOT NULL
	`, userID).Scan(&devices).Error
	return devices, err
}

func (s *Service) sentLogExists(ctx context.Context, notificationID uint64, deviceID uint64, provider string) (bool, error) {
	var exists bool
	err := s.db.WithContext(ctx).Raw(`
		SELECT EXISTS (
		  SELECT 1
		  FROM push_delivery_logs
		  WHERE notification_id = ?
		    AND device_id = ?
		    AND provider = ?
		    AND status = 'SENT'
		)
	`, notificationID, deviceID, provider).Scan(&exists).Error
	return exists, err
}

func (s *Service) logSent(ctx context.Context, notification *notificationRecord, device deviceRecord, provider string, messageID string) error {
	now := time.Now().UTC()
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec(`
			INSERT INTO push_delivery_logs (
			  uuid,
			  notification_id,
			  user_id,
			  device_id,
			  provider,
			  status,
			  provider_message_id,
			  attempted_at,
			  created_at
			)
			VALUES (?, ?, ?, ?, ?, 'SENT', ?, ?, ?)
			ON CONFLICT (notification_id, device_id, provider) WHERE status = 'SENT'
			DO NOTHING
		`, uuid.New(), notification.ID, notification.UserID, device.ID, provider, optionalString(messageID), now, now).Error; err != nil {
			return err
		}
		return tx.Model(&Device{}).
			Where("id = ?", device.ID).
			Updates(map[string]any{
				"last_push_success_at": now,
				"push_failure_count":   0,
				"updated_at":           now,
			}).Error
	})
	return err
}

func (s *Service) logFailed(ctx context.Context, notification *notificationRecord, device deviceRecord, provider string, providerErr *ProviderError) error {
	now := time.Now().UTC()
	return s.db.WithContext(ctx).Create(&PushDeliveryLog{
		UUID:           uuid.New(),
		NotificationID: &notification.ID,
		UserID:         notification.UserID,
		DeviceID:       &device.ID,
		Provider:       provider,
		Status:         DeliveryStatusFailed,
		ErrorCode:      optionalString(providerErr.Code),
		ErrorMessage:   optionalString(providerErr.Error()),
		AttemptedAt:    now,
		CreatedAt:      now,
	}).Error
}

func (s *Service) logSkipped(ctx context.Context, notification *notificationRecord, device *deviceRecord, provider string, code string, message string) error {
	now := time.Now().UTC()
	var deviceID *uint64
	if device != nil {
		deviceID = &device.ID
	}
	return s.db.WithContext(ctx).Create(&PushDeliveryLog{
		UUID:           uuid.New(),
		NotificationID: &notification.ID,
		UserID:         notification.UserID,
		DeviceID:       deviceID,
		Provider:       provider,
		Status:         DeliveryStatusSkipped,
		ErrorCode:      optionalString(code),
		ErrorMessage:   optionalString(message),
		AttemptedAt:    now,
		CreatedAt:      now,
	}).Error
}

func (s *Service) markDeviceInvalid(ctx context.Context, deviceID uint64, providerErr *ProviderError) error {
	now := time.Now().UTC()
	return s.db.WithContext(ctx).Model(&Device{}).
		Where("id = ?", deviceID).
		Updates(map[string]any{
			"fcm_token":            nil,
			"push_enabled":         false,
			"last_push_failure_at": now,
			"push_failure_count":   gorm.Expr("push_failure_count + 1"),
			"updated_at":           now,
		}).Error
}

func (s *Service) chatMessageDeliveredOrSeen(ctx context.Context, notification *notificationRecord) (bool, error) {
	messageUUID := chatMessageUUIDFromDedupe(notification.DedupeKey)
	if messageUUID == "" {
		return false, nil
	}
	var delivered bool
	err := s.db.WithContext(ctx).Raw(`
		SELECT EXISTS (
		  SELECT 1
		  FROM messages m
		  JOIN message_receipts mr ON mr.message_id = m.id
		  WHERE m.uuid = ?
		    AND mr.user_id = ?
		    AND (mr.delivered_at IS NOT NULL OR mr.seen_at IS NOT NULL)
		)
	`, messageUUID, notification.UserID).Scan(&delivered).Error
	return delivered, err
}

func chatMessageUUIDFromDedupe(dedupe *string) string {
	if dedupe == nil {
		return ""
	}
	parts := strings.Split(*dedupe, ":")
	if len(parts) != 3 || parts[0] != "chat_message" {
		return ""
	}
	return parts[1]
}

func pushSkipCode(settings settingsRecord, notificationType string) string {
	if !settings.PushEnabled {
		return "USER_PUSH_DISABLED"
	}
	switch notificationType {
	case TypeNewMatch:
		if !settings.NewMatchEnabled {
			return "NEW_MATCH_DISABLED"
		}
	case TypeChatMessage:
		if !settings.ChatMessageEnabled {
			return "CHAT_MESSAGE_DISABLED"
		}
	case TypeSuperLike:
		if !settings.SuperLikeEnabled {
			return "SUPER_LIKE_DISABLED"
		}
	}
	return ""
}

func inQuietHours(settings settingsRecord, now time.Time) bool {
	if !settings.QuietHoursEnabled || settings.QuietHoursStart == nil || settings.QuietHoursEnd == nil {
		return false
	}
	loc, err := time.LoadLocation(settings.Timezone)
	if err != nil {
		loc = time.UTC
	}
	local := now.In(loc)
	current := local.Hour()*60 + local.Minute()
	start := minutesFromClock(*settings.QuietHoursStart)
	end := minutesFromClock(*settings.QuietHoursEnd)
	if start == end {
		return true
	}
	if start < end {
		return current >= start && current < end
	}
	return current >= start || current < end
}

func minutesFromClock(value string) int {
	parsed, err := time.Parse("15:04:05", normalizeClockText(value))
	if err != nil {
		return 0
	}
	return parsed.Hour()*60 + parsed.Minute()
}

func pushPayloadData(row *notificationRecord) map[string]string {
	data := safeData(row.DataRaw)
	payload := map[string]string{
		"type":             row.NotificationType,
		"notificationUuid": row.UUID,
	}
	if value, ok := data["conversationUuid"].(string); ok && value != "" {
		payload["conversationUuid"] = value
	}
	if value, ok := data["matchUuid"].(string); ok && value != "" {
		payload["matchUuid"] = value
	}
	return payload
}

func notificationResponse(row notificationRecord) NotificationResponse {
	return NotificationResponse{
		NotificationUUID: row.UUID,
		Type:             row.NotificationType,
		Title:            row.Title,
		Body:             row.Body,
		Data:             safeData(row.DataRaw),
		ReadAt:           row.ReadAt,
		ClickedAt:        row.ClickedAt,
		CreatedAt:        row.CreatedAt,
	}
}

func safeData(raw []byte) map[string]any {
	if len(raw) == 0 {
		return map[string]any{}
	}
	var parsed map[string]any
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return map[string]any{}
	}
	safe := map[string]any{}
	for _, key := range []string{"type", "conversationUuid", "matchUuid"} {
		if value, ok := parsed[key]; ok {
			safe[key] = value
		}
	}
	return safe
}

func settingsResponse(row settingsRecord) SettingsResponse {
	return SettingsResponse{
		UUID:               row.UUID,
		PushEnabled:        row.PushEnabled,
		NewMatchEnabled:    row.NewMatchEnabled,
		ChatMessageEnabled: row.ChatMessageEnabled,
		SuperLikeEnabled:   row.SuperLikeEnabled,
		QuietHoursEnabled:  row.QuietHoursEnabled,
		QuietHoursStart:    shortClock(row.QuietHoursStart),
		QuietHoursEnd:      shortClock(row.QuietHoursEnd),
		Timezone:           row.Timezone,
	}
}

func normalizeTokenRequest(req UpsertFCMTokenRequest) (UpsertFCMTokenRequest, error) {
	req.DeviceID = strings.TrimSpace(req.DeviceID)
	req.FCMToken = strings.TrimSpace(req.FCMToken)
	if req.DeviceID == "" {
		return req, validationError("deviceId is required", map[string]any{"field": "deviceId"})
	}
	if req.FCMToken == "" {
		return req, validationError("fcmToken is required", map[string]any{"field": "fcmToken"})
	}
	req.DeviceName = normalizeOptional(req.DeviceName, 150)
	req.Platform = normalizeOptional(req.Platform, 50)
	req.AppVersion = normalizeOptional(req.AppVersion, 50)
	req.OSVersion = normalizeOptional(req.OSVersion, 100)
	return req, nil
}

type normalizedSettings struct {
	PushEnabled        bool
	NewMatchEnabled    bool
	ChatMessageEnabled bool
	SuperLikeEnabled   bool
	QuietHoursEnabled  bool
	QuietHoursStart    *string
	QuietHoursEnd      *string
	Timezone           string
}

func normalizeSettingsRequest(req UpdateSettingsRequest, defaultTimezone string) (normalizedSettings, error) {
	required := []string{"pushEnabled", "newMatchEnabled", "chatMessageEnabled", "superLikeEnabled", "quietHoursEnabled", "timezone"}
	for _, field := range required {
		if !req.Has(field) || req.IsNull(field) {
			return normalizedSettings{}, validationError(field+" is required", map[string]any{"field": field})
		}
	}
	timezone := strings.TrimSpace(derefString(req.Timezone))
	if timezone == "" {
		timezone = defaultTimezone
	}
	if _, err := time.LoadLocation(timezone); err != nil {
		return normalizedSettings{}, validationError("timezone is invalid", map[string]any{"field": "timezone"})
	}

	var start, end *string
	if boolValue(req.QuietHoursEnabled) {
		if !req.Has("quietHoursStart") || req.IsNull("quietHoursStart") {
			return normalizedSettings{}, validationError("quietHoursStart is required when quiet hours are enabled", map[string]any{"field": "quietHoursStart"})
		}
		if !req.Has("quietHoursEnd") || req.IsNull("quietHoursEnd") {
			return normalizedSettings{}, validationError("quietHoursEnd is required when quiet hours are enabled", map[string]any{"field": "quietHoursEnd"})
		}
		normalizedStart, err := normalizeClock(derefString(req.QuietHoursStart), "quietHoursStart")
		if err != nil {
			return normalizedSettings{}, err
		}
		normalizedEnd, err := normalizeClock(derefString(req.QuietHoursEnd), "quietHoursEnd")
		if err != nil {
			return normalizedSettings{}, err
		}
		start = &normalizedStart
		end = &normalizedEnd
	}

	return normalizedSettings{
		PushEnabled:        boolValue(req.PushEnabled),
		NewMatchEnabled:    boolValue(req.NewMatchEnabled),
		ChatMessageEnabled: boolValue(req.ChatMessageEnabled),
		SuperLikeEnabled:   boolValue(req.SuperLikeEnabled),
		QuietHoursEnabled:  boolValue(req.QuietHoursEnabled),
		QuietHoursStart:    start,
		QuietHoursEnd:      end,
		Timezone:           timezone,
	}, nil
}

func normalizeClock(value string, field string) (string, error) {
	parsed, err := time.Parse("15:04", strings.TrimSpace(value))
	if err != nil {
		parsed, err = time.Parse("15:04:05", strings.TrimSpace(value))
	}
	if err != nil {
		return "", validationError(field+" must use HH:MM format", map[string]any{"field": field})
	}
	return parsed.Format("15:04:05"), nil
}

func normalizeClockText(value string) string {
	value = strings.TrimSpace(value)
	if len(value) == len("15:04") {
		value += ":00"
	}
	return value
}

func shortClock(value *string) *string {
	if value == nil {
		return nil
	}
	normalized := normalizeClockText(*value)
	parsed, err := time.Parse("15:04:05", normalized)
	if err != nil {
		trimmed := strings.TrimSpace(*value)
		return &trimmed
	}
	result := parsed.Format("15:04")
	return &result
}

func clearTokenFromOtherDevices(ctx context.Context, tx *gorm.DB, userID uint64, deviceID string, token string, now time.Time) error {
	if token == "" {
		return nil
	}
	return tx.WithContext(ctx).Model(&Device{}).
		Where("fcm_token = ? AND NOT (user_id = ? AND device_id = ?)", token, userID, deviceID).
		Updates(map[string]any{
			"fcm_token":    nil,
			"push_enabled": false,
			"updated_at":   now,
		}).Error
}

func providerError(err error) *ProviderError {
	var providerErr *ProviderError
	if errors.As(err, &providerErr) {
		return providerErr
	}
	return &ProviderError{Code: "PROVIDER_ERROR", Message: err.Error(), Temporary: true, Err: err}
}

func optionalString(value string) *string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return &value
}

func normalizeOptional(value *string, max int) *string {
	if value == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil
	}
	if len(trimmed) > max {
		trimmed = trimmed[:max]
	}
	return &trimmed
}

func derefString(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func boolValue(value *bool) bool {
	return value != nil && *value
}

func (s *Service) defaultTimezone() string {
	value := strings.TrimSpace(s.cfg.GetDefaultTimezone())
	if value == "" {
		return "Asia/Dhaka"
	}
	return value
}

func logNotificationError(context string, err error) {
	if err != nil {
		log.Printf("notification_context=%s error=%v", context, err)
	}
}

func nullUint64(value *uint64) sql.NullInt64 {
	if value == nil {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: int64(*value), Valid: true}
}

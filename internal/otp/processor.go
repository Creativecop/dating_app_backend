package otp

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"gorm.io/gorm"

	"github.com/neoscoder/aura-backend/internal/otp/provider"
)

type Processor struct {
	db        *gorm.DB
	providers map[string]provider.OTPProvider
}

func NewProcessor(db *gorm.DB, providers map[string]provider.OTPProvider) *Processor {
	return &Processor{db: db, providers: providers}
}

func (p *Processor) Register(mux *asynq.ServeMux) {
	mux.HandleFunc(TaskDeliverOTP, p.ProcessDeliverOTP)
}

func (p *Processor) ProcessDeliverOTP(ctx context.Context, task *asynq.Task) error {
	var payload DeliveryPayload
	if err := json.Unmarshal(task.Payload(), &payload); err != nil {
		return fmt.Errorf("unmarshal otp delivery payload: %w", err)
	}
	log.Printf("[OTP] delivery_task_received channel=%s purpose=%s otp_id=%s to=%s", payload.Channel, payload.Purpose, payload.OTPID, maskIdentifier(payload.Identifier))

	otpUUID, err := uuid.Parse(payload.OTPID)
	if err != nil {
		return fmt.Errorf("parse otp id: %w", err)
	}

	var code OTPCode
	if err := p.db.WithContext(ctx).Where("uuid = ?", otpUUID).First(&code).Error; err != nil {
		return fmt.Errorf("find otp code: %w", err)
	}

	now := time.Now().UTC()
	if now.After(code.ExpiresAt) {
		log.Printf("[OTP] delivery_skipped reason=expired channel=%s otp_id=%s", payload.Channel, payload.OTPID)
		return p.db.WithContext(ctx).Model(&OTPCode{}).
			Where("id = ?", code.ID).
			Updates(map[string]any{"delivery_status": DeliveryExpired, "updated_at": now}).Error
	}
	if code.ConsumedAt != nil {
		log.Printf("[OTP] delivery_skipped reason=consumed channel=%s otp_id=%s", payload.Channel, payload.OTPID)
		return nil
	}

	sender, ok := p.providers[payload.Channel]
	if !ok {
		log.Printf("[OTP] delivery_failed reason=missing_provider channel=%s otp_id=%s", payload.Channel, payload.OTPID)
		return fmt.Errorf("missing otp provider for channel %s", payload.Channel)
	}

	providerName := payload.Channel
	log.Printf("[OTP] delivery_sending channel=%s provider=%s otp_id=%s to=%s", payload.Channel, providerName, payload.OTPID, maskIdentifier(payload.Identifier))
	if err := p.db.WithContext(ctx).Model(&OTPCode{}).
		Where("id = ?", code.ID).
		Updates(map[string]any{
			"delivery_status":          DeliverySending,
			"delivery_provider":        providerName,
			"delivery_attempts":        gorm.Expr("delivery_attempts + 1"),
			"last_delivery_attempt_at": now,
			"updated_at":               now,
		}).Error; err != nil {
		return fmt.Errorf("mark otp sending: %w", err)
	}

	messageID, sendErr := sender.SendOTP(ctx, payload.Identifier, payload.Code, payload.Purpose)
	if sendErr != nil {
		errMessage := sendErr.Error()
		log.Printf("[OTP] delivery_failed channel=%s provider=%s otp_id=%s error=%v", payload.Channel, providerName, payload.OTPID, sendErr)
		_ = p.createDeliveryLog(ctx, code.ID, payload.Channel, providerName, payload.Identifier, DeliveryFailed, "", errMessage)
		_ = p.db.WithContext(ctx).Model(&OTPCode{}).
			Where("id = ?", code.ID).
			Updates(map[string]any{
				"delivery_status": DeliveryFailed,
				"failed_at":       time.Now().UTC(),
				"delivery_error":  errMessage,
				"updated_at":      time.Now().UTC(),
			}).Error
		return sendErr
	}

	if err := p.createDeliveryLog(ctx, code.ID, payload.Channel, providerName, payload.Identifier, DeliverySent, messageID, ""); err != nil {
		return err
	}
	log.Printf("[OTP] delivery_sent channel=%s provider=%s otp_id=%s provider_message_id=%s", payload.Channel, providerName, payload.OTPID, messageID)

	now = time.Now().UTC()
	return p.db.WithContext(ctx).Model(&OTPCode{}).
		Where("id = ?", code.ID).
		Updates(map[string]any{
			"delivery_status":     DeliverySent,
			"provider_message_id": messageID,
			"sent_at":             now,
			"failed_at":           nil,
			"delivery_error":      nil,
			"updated_at":          now,
		}).Error
}

func (p *Processor) createDeliveryLog(ctx context.Context, otpID uint64, channel, providerName, identifier, status, messageID, errMessage string) error {
	deliveryLog := &OTPDeliveryLog{
		UUID:        uuid.New(),
		OTPCodeID:   otpID,
		Channel:     channel,
		Provider:    stringPtr(providerName),
		Identifier:  identifier,
		Status:      status,
		AttemptedAt: time.Now().UTC(),
		CreatedAt:   time.Now().UTC(),
	}
	if messageID != "" {
		deliveryLog.ProviderMessageID = stringPtr(messageID)
	}
	if errMessage != "" {
		deliveryLog.ErrorMessage = stringPtr(errMessage)
	}
	return p.db.WithContext(ctx).Create(deliveryLog).Error
}

func maskIdentifier(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if strings.Contains(value, "@") {
		parts := strings.SplitN(value, "@", 2)
		return maskMiddle(parts[0]) + "@" + parts[1]
	}
	return maskMiddle(value)
}

func maskMiddle(value string) string {
	if len(value) <= 4 {
		return "****"
	}
	if len(value) <= 8 {
		return value[:2] + "****" + value[len(value)-2:]
	}
	return value[:4] + "****" + value[len(value)-4:]
}

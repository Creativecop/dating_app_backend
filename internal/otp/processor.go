package otp

import (
	"context"
	"encoding/json"
	"fmt"
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
		return p.db.WithContext(ctx).Model(&OTPCode{}).
			Where("id = ?", code.ID).
			Updates(map[string]any{"delivery_status": DeliveryExpired, "updated_at": now}).Error
	}
	if code.ConsumedAt != nil {
		return nil
	}

	sender, ok := p.providers[payload.Channel]
	if !ok {
		return fmt.Errorf("missing otp provider for channel %s", payload.Channel)
	}

	providerName := payload.Channel
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
	log := &OTPDeliveryLog{
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
		log.ProviderMessageID = stringPtr(messageID)
	}
	if errMessage != "" {
		log.ErrorMessage = stringPtr(errMessage)
	}
	return p.db.WithContext(ctx).Create(log).Error
}

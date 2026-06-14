package otp

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	goredis "github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"github.com/neoscoder/aura-backend/internal/config"
)

var (
	ErrInvalidChannel    = errors.New("invalid otp channel")
	ErrInvalidPurpose    = errors.New("invalid otp purpose")
	ErrInvalidIdentifier = errors.New("invalid otp identifier")
	ErrInvalidCode       = errors.New("invalid otp code")
	ErrOTPExpired        = errors.New("otp expired")
	ErrOTPMaxAttempts    = errors.New("otp max attempts exceeded")
	ErrRateLimited       = errors.New("otp rate limit exceeded")
	ErrQueueUnavailable  = errors.New("otp queue unavailable")
)

type Service struct {
	db    *gorm.DB
	redis *goredis.Client
	queue *asynq.Client
	cfg   config.OTPConfig
}

type RequestInput struct {
	Channel   string
	Phone     string
	Email     string
	Purpose   string
	IPAddress string
	UserAgent string
	DeviceID  string
}

type VerifyInput struct {
	Channel string
	Phone   string
	Email   string
	Purpose string
	Code    string
}

func NewService(db *gorm.DB, redis *goredis.Client, queue *asynq.Client, cfg config.OTPConfig) *Service {
	return &Service{db: db, redis: redis, queue: queue, cfg: cfg}
}

func (s *Service) Request(ctx context.Context, input RequestInput) (*OTPCode, error) {
	channel, err := NormalizeChannel(input.Channel)
	if err != nil {
		return nil, err
	}
	purpose, err := NormalizePurpose(input.Purpose)
	if err != nil {
		return nil, err
	}
	identifier, err := NormalizeIdentifier(channel, input.Phone, input.Email)
	if err != nil {
		return nil, err
	}

	identifierHash := HashIdentifier(s.cfg.Secret, identifier)
	if err := s.enforceRateLimits(ctx, identifierHash, input.IPAddress); err != nil {
		return nil, err
	}

	code, err := GenerateNumericCode(s.cfg.Length)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	otpCode := &OTPCode{
		UUID:             uuid.New(),
		Channel:          channel,
		Purpose:          purpose,
		IdentifierHash:   identifierHash,
		OTPHash:          HashCode(s.cfg.Secret, code, identifier, purpose),
		ExpiresAt:        now.Add(s.cfg.ExpiryTTL()),
		MaxAttempts:      s.cfg.MaxAttempts,
		IPAddress:        input.IPAddress,
		UserAgent:        input.UserAgent,
		DeliveryStatus:   DeliveryPending,
		DeliveryAttempts: 0,
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	if channel == ChannelWhatsApp {
		otpCode.Phone = stringPtr(identifier)
	} else {
		otpCode.Email = stringPtr(identifier)
	}
	if input.DeviceID != "" {
		otpCode.DeviceID = stringPtr(input.DeviceID)
	}

	if err := s.db.WithContext(ctx).Create(otpCode).Error; err != nil {
		return nil, fmt.Errorf("create otp code: %w", err)
	}

	if s.queue == nil {
		_ = s.markDeliveryFailed(ctx, otpCode.ID, "queue client is not configured")
		return nil, ErrQueueUnavailable
	}

	task, err := NewDeliveryTask(DeliveryPayload{
		OTPID:      otpCode.UUID.String(),
		Channel:    channel,
		Identifier: identifier,
		Code:       code,
		Purpose:    purpose,
	})
	if err != nil {
		_ = s.markDeliveryFailed(ctx, otpCode.ID, err.Error())
		return nil, err
	}

	if _, err := s.queue.EnqueueContext(ctx, task, DeliveryTaskOptions()...); err != nil {
		_ = s.markDeliveryFailed(ctx, otpCode.ID, err.Error())
		return nil, fmt.Errorf("enqueue otp delivery: %w", err)
	}

	if err := s.db.WithContext(ctx).Model(&OTPCode{}).
		Where("id = ?", otpCode.ID).
		Updates(map[string]any{
			"delivery_status": DeliveryQueued,
			"updated_at":      time.Now().UTC(),
		}).Error; err != nil {
		return nil, fmt.Errorf("mark otp queued: %w", err)
	}
	otpCode.DeliveryStatus = DeliveryQueued

	return otpCode, nil
}

func (s *Service) Verify(ctx context.Context, input VerifyInput) (*OTPCode, error) {
	channel, err := NormalizeChannel(input.Channel)
	if err != nil {
		return nil, err
	}
	purpose, err := NormalizePurpose(input.Purpose)
	if err != nil {
		return nil, err
	}
	identifier, err := NormalizeIdentifier(channel, input.Phone, input.Email)
	if err != nil {
		return nil, err
	}
	if input.Code == "" {
		return nil, ErrInvalidCode
	}

	var code OTPCode
	err = s.db.WithContext(ctx).
		Where("identifier_hash = ? AND channel = ? AND purpose = ? AND consumed_at IS NULL", HashIdentifier(s.cfg.Secret, identifier), channel, purpose).
		Order("created_at DESC").
		First(&code).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrInvalidCode
	}
	if err != nil {
		return nil, fmt.Errorf("find otp code: %w", err)
	}

	now := time.Now().UTC()
	if now.After(code.ExpiresAt) {
		_ = s.db.WithContext(ctx).Model(&OTPCode{}).Where("id = ?", code.ID).Update("delivery_status", DeliveryExpired).Error
		return nil, ErrOTPExpired
	}
	if code.AttemptCount >= code.MaxAttempts {
		return nil, ErrOTPMaxAttempts
	}

	if !CompareCodeHash(code.OTPHash, s.cfg.Secret, input.Code, identifier, purpose) {
		nextAttempts := code.AttemptCount + 1
		_ = s.db.WithContext(ctx).Model(&OTPCode{}).
			Where("id = ?", code.ID).
			Updates(map[string]any{"attempt_count": nextAttempts, "updated_at": now}).Error
		if nextAttempts >= code.MaxAttempts {
			return nil, ErrOTPMaxAttempts
		}
		return nil, ErrInvalidCode
	}

	if err := s.db.WithContext(ctx).Model(&OTPCode{}).
		Where("id = ? AND consumed_at IS NULL", code.ID).
		Updates(map[string]any{"consumed_at": now, "updated_at": now}).Error; err != nil {
		return nil, fmt.Errorf("consume otp code: %w", err)
	}
	code.ConsumedAt = &now

	return &code, nil
}

func (s *Service) enforceRateLimits(ctx context.Context, identifierHash, ip string) error {
	if s.redis == nil {
		return nil
	}

	cooldownKey := "rl:otp:cooldown:" + identifierHash
	ok, err := s.redis.SetNX(ctx, cooldownKey, "1", s.cfg.ResendCooldownTTL()).Result()
	if err != nil {
		return fmt.Errorf("otp cooldown rate limit: %w", err)
	}
	if !ok {
		return ErrRateLimited
	}

	identifierHourKey := "rl:otp:identifier:hour:" + identifierHash
	if err := incrementWindow(ctx, s.redis, identifierHourKey, time.Hour, s.cfg.MaxPerIdentifierHour); err != nil {
		return err
	}

	if ip != "" {
		ipHourKey := "rl:otp:ip:hour:" + ip
		if err := incrementWindow(ctx, s.redis, ipHourKey, time.Hour, s.cfg.MaxPerIPHour); err != nil {
			return err
		}
	}

	return nil
}

func incrementWindow(ctx context.Context, client *goredis.Client, key string, ttl time.Duration, max int) error {
	count, err := client.Incr(ctx, key).Result()
	if err != nil {
		return fmt.Errorf("increment rate limit: %w", err)
	}
	if count == 1 {
		_ = client.Expire(ctx, key, ttl).Err()
	}
	if max > 0 && count > int64(max) {
		return ErrRateLimited
	}
	return nil
}

func (s *Service) markDeliveryFailed(ctx context.Context, id uint64, message string) error {
	now := time.Now().UTC()
	return s.db.WithContext(ctx).Model(&OTPCode{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"delivery_status": DeliveryFailed,
			"failed_at":       now,
			"delivery_error":  message,
			"updated_at":      now,
		}).Error
}

func stringPtr(value string) *string {
	return &value
}

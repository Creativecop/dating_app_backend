package notification

import (
	"context"

	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/messaging"
	"google.golang.org/api/option"

	"github.com/neoscoder/aura-backend/internal/config"
)

type FCMProvider struct {
	client *messaging.Client
}

func NewFCMProvider(ctx context.Context, cfg config.NotificationConfig) (*FCMProvider, error) {
	options := []option.ClientOption{}
	if cfg.FirebaseCredentialsFile != "" {
		options = append(options, option.WithCredentialsFile(cfg.FirebaseCredentialsFile))
	}
	app, err := firebase.NewApp(ctx, &firebase.Config{ProjectID: cfg.FirebaseProjectID}, options...)
	if err != nil {
		return nil, err
	}
	client, err := app.Messaging(ctx)
	if err != nil {
		return nil, err
	}
	return &FCMProvider{client: client}, nil
}

func (p *FCMProvider) Name() string {
	return ProviderFCM
}

func (p *FCMProvider) Send(ctx context.Context, message PushMessage) (*PushResult, error) {
	id, err := p.client.Send(ctx, &messaging.Message{
		Token: message.Token,
		Notification: &messaging.Notification{
			Title: message.Title,
			Body:  message.Body,
		},
		Data: message.Data,
	})
	if err != nil {
		return nil, classifyFCMError(err)
	}
	return &PushResult{MessageID: id}, nil
}

func classifyFCMError(err error) *ProviderError {
	switch {
	case messaging.IsUnregistered(err):
		return &ProviderError{Code: "UNREGISTERED", Message: err.Error(), InvalidToken: true, Err: err}
	case messaging.IsInvalidArgument(err):
		return &ProviderError{Code: "INVALID_ARGUMENT", Message: err.Error(), InvalidToken: true, Err: err}
	case messaging.IsSenderIDMismatch(err):
		return &ProviderError{Code: "SENDER_ID_MISMATCH", Message: err.Error(), InvalidToken: true, Err: err}
	case messaging.IsUnavailable(err), messaging.IsInternal(err), messaging.IsQuotaExceeded(err):
		return &ProviderError{Code: "TEMPORARY_PROVIDER_ERROR", Message: err.Error(), Temporary: true, Err: err}
	default:
		return &ProviderError{Code: "FCM_ERROR", Message: err.Error(), Temporary: true, Err: err}
	}
}

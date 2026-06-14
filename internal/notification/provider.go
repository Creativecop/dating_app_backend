package notification

import (
	"context"
	"fmt"
)

type PushMessage struct {
	Token string
	Title string
	Body  string
	Data  map[string]string
}

type PushResult struct {
	MessageID string
}

type ProviderError struct {
	Code         string
	Message      string
	InvalidToken bool
	Temporary    bool
	Err          error
}

func (e *ProviderError) Error() string {
	if e.Message != "" {
		return e.Message
	}
	if e.Err != nil {
		return e.Err.Error()
	}
	return e.Code
}

func (e *ProviderError) Unwrap() error {
	return e.Err
}

type Provider interface {
	Send(ctx context.Context, message PushMessage) (*PushResult, error)
	Name() string
}

type NoopProvider struct{}

func NewNoopProvider() *NoopProvider {
	return &NoopProvider{}
}

func (p *NoopProvider) Name() string {
	return ProviderNoop
}

func (p *NoopProvider) Send(ctx context.Context, message PushMessage) (*PushResult, error) {
	_ = ctx
	if message.Token == "" {
		return nil, &ProviderError{Code: "INVALID_TOKEN", Message: "missing push token", InvalidToken: true}
	}
	return &PushResult{MessageID: fmt.Sprintf("noop:%s", message.Token)}, nil
}

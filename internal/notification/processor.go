package notification

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hibiken/asynq"
)

type Processor struct {
	service *Service
}

func NewProcessor(service *Service) *Processor {
	return &Processor{service: service}
}

func (p *Processor) Register(mux *asynq.ServeMux) {
	mux.HandleFunc(TaskSendPush, p.ProcessSendPush)
}

func (p *Processor) ProcessSendPush(ctx context.Context, task *asynq.Task) error {
	var payload SendPushPayload
	if err := json.Unmarshal(task.Payload(), &payload); err != nil {
		return fmt.Errorf("unmarshal push payload: %w", err)
	}
	if payload.NotificationUUID == "" || payload.UserID == 0 {
		return nil
	}
	return p.service.ProcessSendPush(ctx, payload)
}

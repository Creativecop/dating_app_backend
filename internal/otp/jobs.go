package otp

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/hibiken/asynq"
)

const TaskDeliverOTP = "otp:deliver"

type DeliveryPayload struct {
	OTPID      string `json:"otpId"`
	Channel    string `json:"channel"`
	Identifier string `json:"identifier"`
	Code       string `json:"code"`
	Purpose    string `json:"purpose"`
}

func NewDeliveryTask(payload DeliveryPayload) (*asynq.Task, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal otp delivery payload: %w", err)
	}
	return asynq.NewTask(TaskDeliverOTP, body), nil
}

func DeliveryTaskOptions() []asynq.Option {
	return []asynq.Option{
		asynq.Queue("critical"),
		asynq.MaxRetry(3),
		asynq.Timeout(15 * time.Second),
		asynq.Unique(60 * time.Second),
	}
}

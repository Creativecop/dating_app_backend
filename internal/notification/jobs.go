package notification

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/hibiken/asynq"
)

const TaskSendPush = "notification:send_push"

type SendPushPayload struct {
	NotificationUUID string `json:"notificationUuid"`
	UserID           uint64 `json:"userId"`
}

func NewSendPushTask(payload SendPushPayload) (*asynq.Task, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	return asynq.NewTask(TaskSendPush, data), nil
}

func PushTaskID(kind string, notificationUUID string) string {
	return fmt.Sprintf("push:%s:%s", kind, notificationUUID)
}

func PushTaskOptions(taskID string, maxRetry int, timeout time.Duration, delay time.Duration) []asynq.Option {
	options := []asynq.Option{
		asynq.Queue("default"),
		asynq.TaskID(taskID),
		asynq.MaxRetry(maxRetry),
		asynq.Timeout(timeout),
	}
	if delay > 0 {
		options = append(options, asynq.ProcessIn(delay))
	}
	return options
}

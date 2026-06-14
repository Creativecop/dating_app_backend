package media

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/hibiken/asynq"
)

const (
	TaskProcessPhoto      = "media:process_photo"
	TaskProcessIntroVideo = "media:process_intro_video"
)

type ProcessMediaPayload struct {
	MediaUUID string `json:"mediaUuid"`
}

func NewProcessPhotoTask(mediaUUID string) (*asynq.Task, error) {
	return newProcessMediaTask(TaskProcessPhoto, mediaUUID)
}

func NewProcessIntroVideoTask(mediaUUID string) (*asynq.Task, error) {
	return newProcessMediaTask(TaskProcessIntroVideo, mediaUUID)
}

func newProcessMediaTask(taskType, mediaUUID string) (*asynq.Task, error) {
	body, err := json.Marshal(ProcessMediaPayload{MediaUUID: mediaUUID})
	if err != nil {
		return nil, fmt.Errorf("marshal media task: %w", err)
	}
	return asynq.NewTask(taskType, body), nil
}

func ProcessTaskOptions() []asynq.Option {
	return []asynq.Option{
		asynq.Queue("default"),
		asynq.MaxRetry(3),
		asynq.Timeout(2 * time.Minute),
		asynq.Unique(30 * time.Second),
	}
}

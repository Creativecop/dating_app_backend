package queue

import (
	"github.com/hibiken/asynq"

	"github.com/neoscoder/aura-backend/internal/config"
)

func NewServer(cfg config.RedisConfig) *asynq.Server {
	return asynq.NewServer(
		RedisOpt(cfg),
		asynq.Config{
			Concurrency: 10,
			Queues: map[string]int{
				"critical": 6,
				"default":  3,
				"low":      1,
			},
		},
	)
}

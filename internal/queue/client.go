package queue

import (
	"github.com/hibiken/asynq"

	"github.com/neoscoder/aura-backend/internal/config"
)

func RedisOpt(cfg config.RedisConfig) asynq.RedisClientOpt {
	return asynq.RedisClientOpt{
		Addr:     cfg.Addr(),
		Password: cfg.Password,
		DB:       cfg.DB,
	}
}

func NewClient(cfg config.RedisConfig) *asynq.Client {
	return asynq.NewClient(RedisOpt(cfg))
}

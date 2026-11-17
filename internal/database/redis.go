package database

import (
	"fmt"

	"github.com/gabipuzon/horus/internal/queue"
	"github.com/redis/go-redis/v9"
)

type RedisConfig struct {
	Host string
	Port string
}

func NewRedis(config RedisConfig) *queue.Redis {
	client := redis.NewClient(&redis.Options{
		Addr:                  fmt.Sprintf("%s:%s", config.Host, config.Port),
		ContextTimeoutEnabled: true,
	})
	return queue.NewRedisClient(client)
}

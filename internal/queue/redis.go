package queue

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
)

type Config struct {
	Host string
	Port string
}

type Redis struct {
	client *redis.Client
}

func NewRedis(config Config) *Redis {
	return &Redis{
		client: redis.NewClient(&redis.Options{
			Addr: fmt.Sprintf("%s:%s", config.Host, config.Port),
		}),
	}
}

func (r *Redis) Ping(ctx context.Context) error {
	return r.client.Ping(ctx).Err()
}

func (r *Redis) Close() error {
	return r.client.Close()
}

package queue

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

const CheckQueue = "horus:checks"

const dequeueWait = time.Second

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

func (r *Redis) EnqueueCheck(
	ctx context.Context,
	job CheckJob,
) error {
	data, err := json.Marshal(job)
	if err != nil {
		return err
	}

	return r.client.LPush(
		ctx,
		CheckQueue,
		data,
	).Err()
}

func (r *Redis) DequeueCheck(
	ctx context.Context,
) (CheckJob, error) {
	for {
		if err := ctx.Err(); err != nil {
			return CheckJob{}, err
		}

		data, err := r.client.BRPop(ctx, dequeueWait, CheckQueue).Result()
		if errors.Is(err, redis.Nil) {
			continue
		}
		if err != nil {
			return CheckJob{}, err
		}

		var job CheckJob

		if err := json.Unmarshal(
			[]byte(data[1]),
			&job,
		); err != nil {
			return CheckJob{}, err
		}

		return job, nil
	}
}

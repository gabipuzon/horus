package queue

import (
	"context"
	"testing"
	"time"
)

func TestRedisPing(t *testing.T) {
	redis := NewRedis(Config{
		Host: "localhost",
		Port: "6379",
	})
	defer redis.Close()

	if err := redis.Ping(context.Background()); err != nil {
		t.Fatalf("expected redis ping to succeed: %v", err)
	}
}

func TestRedisCheckQueue(t *testing.T) {
	ctx := context.Background()

	redis := NewRedis(Config{
		Host: "localhost",
		Port: "6379",
	})
	defer redis.Close()

	if err := redis.client.Del(ctx, CheckQueue).Err(); err != nil {
		t.Fatalf("failed to clear check queue: %v", err)
	}

	expected := CheckJob{
		MonitorID: "monitor-123",
	}

	if err := redis.EnqueueCheck(ctx, expected); err != nil {
		t.Fatalf("failed to enqueue check: %v", err)
	}

	dequeueCtx, cancel := context.WithTimeout(
		ctx,
		time.Second,
	)
	defer cancel()

	actual, err := redis.DequeueCheck(dequeueCtx)
	if err != nil {
		t.Fatalf("failed to dequeue check: %v", err)
	}

	if actual.MonitorID != expected.MonitorID {
		t.Fatalf(
			"expected monitor ID %s, got %s",
			expected.MonitorID,
			actual.MonitorID,
		)
	}
}

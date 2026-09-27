package queue

import (
	"context"
	"testing"
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

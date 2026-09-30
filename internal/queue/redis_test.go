package queue

import (
	"context"
	"errors"
	"net"
	"testing"
	"time"

	goredis "github.com/redis/go-redis/v9"
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

func TestRedisPingRespectsContextDeadline(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	closed := make(chan struct{})
	defer close(closed)
	go func() {
		connection, err := listener.Accept()
		if err == nil {
			defer connection.Close()
			<-closed
		}
	}()
	host, port, err := net.SplitHostPort(listener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	r := NewRedis(Config{Host: host, Port: port})
	defer r.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	start := time.Now()
	if err := r.Ping(ctx); err == nil {
		t.Fatal("expected stalled Redis ping to fail")
	}
	if elapsed := time.Since(start); elapsed > time.Second {
		t.Fatalf("Redis ping ignored context deadline, taking %s", elapsed)
	}
}

func TestRedisDequeueStopsAfterCancellation(t *testing.T) {
	r := &Redis{client: goredis.NewClient(&goredis.Options{Addr: "localhost:6379", DB: 15})}
	defer r.Close()
	if err := r.client.Del(context.Background(), CheckQueue).Err(); err != nil {
		t.Fatalf("failed to clear isolated test queue: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		_, err := r.DequeueCheck(ctx)
		done <- err
	}()
	time.Sleep(100 * time.Millisecond)
	cancel()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("expected context cancellation, got %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("blocking Redis dequeue did not stop after cancellation")
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

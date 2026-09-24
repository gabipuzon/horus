package database

import (
	"context"
	"testing"
)

func TestNewPool(t *testing.T) {
	config := Config{
		Host:     "localhost",
		Port:     "5432",
		User:     "horus",
		Password: "horus",
		Name:     "horus",
	}

	ctx := context.Background()

	pool, err := NewPool(ctx, config)
	if err != nil {
		t.Fatalf("failed to connect to database: %v", err)
	}

	defer pool.Close()
}

package database

import (
	"context"
	"testing"

	"github.com/gabipuzon/horus/internal/testutil"
)

func TestNewPool(t *testing.T) {
	testutil.RequireIntegration(t)
	config := Config{
		Host:     testutil.Env("HORUS_DB_HOST", "localhost"),
		Port:     testutil.Env("HORUS_DB_PORT", "5432"),
		User:     testutil.Env("HORUS_DB_USER", "horus"),
		Password: testutil.Env("HORUS_DB_PASSWORD", "horus"),
		Name:     testutil.Env("HORUS_DB_NAME", "horus"),
	}

	ctx := context.Background()

	pool, err := NewPool(ctx, config)
	if err != nil {
		t.Fatalf("failed to connect to database: %v", err)
	}

	defer pool.Close()
}

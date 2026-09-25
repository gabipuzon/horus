package monitor

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestCheckRepositoryCreate(t *testing.T) {
	ctx := context.Background()

	db, err := pgxpool.New(
		ctx,
		"postgres://horus:horus@localhost:5432/horus",
	)
	if err != nil {
		t.Fatalf("failed to connect to database: %v", err)
	}
	defer db.Close()

	repository := NewCheckRepository(db)

	monitorID := "00000000-0000-0000-0000-000000000001"

	_, err = db.Exec(
		ctx,
		`
		INSERT INTO monitors (
			id,
			name,
			url,
			interval_seconds,
			timeout_seconds,
			expected_status,
			enabled,
			created_at,
			updated_at
		)
		VALUES (
			$1,
			'Repository Test Monitor',
			'https://example.com',
			60,
			5,
			200,
			true,
			NOW(),
			NOW()
		)
		ON CONFLICT (id) DO NOTHING
		`,
		monitorID,
	)
	if err != nil {
		t.Fatalf("failed to create test monitor: %v", err)
	}

	result := CheckResult{
		StatusCode:  200,
		Latency:     150 * time.Millisecond,
		Success:     true,
		FailureType: FailureNone,
	}

	err = repository.Create(ctx, monitorID, result)
	if err != nil {
		t.Fatalf("failed to create check: %v", err)
	}

	var (
		statusCode  int
		latencyMs   int64
		success     bool
		failureType string
	)

	err = db.QueryRow(
		ctx,
		`
		SELECT
			status_code,
			latency_ms,
			success,
			failure_type
		FROM checks
		WHERE monitor_id = $1
		ORDER BY checked_at DESC
		LIMIT 1
		`,
		monitorID,
	).Scan(
		&statusCode,
		&latencyMs,
		&success,
		&failureType,
	)
	if err != nil {
		t.Fatalf("failed to query created check: %v", err)
	}

	if statusCode != 200 {
		t.Fatalf("expected status code 200, got %d", statusCode)
	}

	if latencyMs != 150 {
		t.Fatalf("expected latency 150ms, got %dms", latencyMs)
	}

	if !success {
		t.Fatal("expected check to be successful")
	}

	if failureType != "" {
		t.Fatalf("expected no failure type, got %q", failureType)
	}
}

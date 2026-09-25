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

func TestCheckRepositoryListByMonitor(t *testing.T) {
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

	monitorID := "00000000-0000-0000-0000-000000000002"

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
			'List Test Monitor',
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

	_, err = db.Exec(
		ctx,
		`
		DELETE FROM checks
		WHERE monitor_id = $1
		`,
		monitorID,
	)
	if err != nil {
		t.Fatalf("failed to clean up checks: %v", err)
	}

	firstResult := CheckResult{
		StatusCode:  200,
		Latency:     100 * time.Millisecond,
		Success:     true,
		FailureType: FailureNone,
	}

	secondResult := CheckResult{
		StatusCode:  500,
		Latency:     250 * time.Millisecond,
		Success:     false,
		FailureType: FailureHTTP,
	}

	if err := repository.Create(ctx, monitorID, firstResult); err != nil {
		t.Fatalf("failed to create first check: %v", err)
	}

	time.Sleep(10 * time.Millisecond)

	if err := repository.Create(ctx, monitorID, secondResult); err != nil {
		t.Fatalf("failed to create second check: %v", err)
	}

	checks, err := repository.ListByMonitor(ctx, monitorID, 50, 0)
	if err != nil {
		t.Fatalf("failed to list checks: %v", err)
	}

	if len(checks) != 2 {
		t.Fatalf(
			"expected 2 checks, got %d",
			len(checks),
		)
	}

	if checks[0].StatusCode != 500 {
		t.Fatalf(
			"expected newest check to have status 500, got %d",
			checks[0].StatusCode,
		)
	}

	if checks[0].Latency != 250*time.Millisecond {
		t.Fatalf(
			"expected latency 250ms, got %s",
			checks[0].Latency,
		)
	}

	if checks[0].Success {
		t.Fatal("expected newest check to be unsuccessful")
	}

	if checks[0].FailureType != FailureHTTP {
		t.Fatalf(
			"expected failure type %q, got %q",
			FailureHTTP,
			checks[0].FailureType,
		)
	}

	if checks[1].StatusCode != 200 {
		t.Fatalf(
			"expected oldest check to have status 200, got %d",
			checks[1].StatusCode,
		)
	}
}

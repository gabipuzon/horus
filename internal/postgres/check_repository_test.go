package postgres

import (
	"context"
	"testing"
	"time"

	"github.com/gabipuzon/horus/internal/check"
	"github.com/gabipuzon/horus/internal/incident"
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

	result := check.Result{
		StatusCode:  200,
		Latency:     150 * time.Millisecond,
		Success:     true,
		FailureType: check.FailureNone,
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

func TestCheckRepositoryIncidentLifecycle(t *testing.T) {
	ctx := context.Background()
	db, err := pgxpool.New(ctx, "postgres://horus:horus@localhost:5432/horus")
	if err != nil {
		t.Fatalf("failed to connect to database: %v", err)
	}
	defer db.Close()

	monitorID := "00000000-0000-0000-0000-000000000004"
	_, err = db.Exec(ctx, `
		INSERT INTO monitors (id, name, url, interval_seconds, timeout_seconds, expected_status, enabled, created_at, updated_at)
		VALUES ($1, 'Incident Test', 'https://example.com', 60, 5, 200, true, NOW(), NOW())
		ON CONFLICT (id) DO NOTHING
	`, monitorID)
	if err != nil {
		t.Fatalf("failed to create test monitor: %v", err)
	}
	_, err = db.Exec(ctx, `DELETE FROM incidents WHERE monitor_id = $1`, monitorID)
	if err != nil {
		t.Fatalf("failed to clean test incidents: %v", err)
	}

	repository := NewCheckRepository(db)
	first := incident.Incident{
		MonitorID:      monitorID,
		StartedAt:      time.Now().Add(-time.Minute).Truncate(time.Microsecond),
		FailureType:    "http",
		StatusCode:     503,
		FailureMessage: "unexpected status",
	}
	opened, err := repository.OpenIncident(ctx, first)
	if err != nil || opened == nil {
		t.Fatalf("failed to open incident: %v", err)
	}
	first.FailureType = "network"
	first.StatusCode = 0
	first.FailureMessage = "later failure"
	opened, err = repository.OpenIncident(ctx, first)
	if err != nil || opened != nil {
		t.Fatalf("failed to keep incident open: %v", err)
	}

	var id, failureType, failureMessage string
	var statusCode int
	var startedAt time.Time
	var resolvedAt *time.Time
	err = db.QueryRow(ctx, `
		SELECT id, started_at, resolved_at, failure_type, status_code, failure_message
		FROM incidents WHERE monitor_id = $1 AND resolved_at IS NULL
	`, monitorID).Scan(&id, &startedAt, &resolvedAt, &failureType, &statusCode, &failureMessage)
	if err != nil {
		t.Fatalf("failed to query open incident: %v", err)
	}
	if resolvedAt != nil || failureType != "http" || statusCode != 503 || failureMessage != "unexpected status" {
		t.Fatalf("expected original open incident context to remain, got resolved=%v type=%q status=%d message=%q", resolvedAt, failureType, statusCode, failureMessage)
	}
	if !startedAt.Equal(first.StartedAt) {
		t.Fatalf("expected incident start %s, got %s", first.StartedAt, startedAt)
	}
	var count int
	if err := db.QueryRow(ctx, `SELECT COUNT(*) FROM incidents WHERE monitor_id = $1`, monitorID).Scan(&count); err != nil {
		t.Fatalf("failed to count incidents: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected repeated failure to retain one incident, got %d", count)
	}

	resolved, err := repository.ResolveIncident(ctx, monitorID)
	if err != nil || resolved == nil {
		t.Fatalf("failed to resolve incident: %v", err)
	}
	if resolved.ID != id || resolved.ResolvedAt == nil {
		t.Fatalf("expected resolved transition for incident %q, got %+v", id, resolved)
	}
	if unchanged, err := repository.ResolveIncident(ctx, monitorID); err != nil || unchanged != nil {
		t.Fatalf("expected no second resolution transition, got %+v, %v", unchanged, err)
	}
	var resolvedID string
	if err := db.QueryRow(ctx, `SELECT id FROM incidents WHERE monitor_id = $1 AND resolved_at IS NOT NULL`, monitorID).Scan(&resolvedID); err != nil {
		t.Fatalf("failed to query resolved incident: %v", err)
	}
	if resolvedID != id {
		t.Fatalf("expected incident %q to resolve, got %q", id, resolvedID)
	}

	first.StartedAt = time.Now()
	opened, err = repository.OpenIncident(ctx, first)
	if err != nil || opened == nil {
		t.Fatalf("failed to open subsequent incident: %v", err)
	}
	if err := db.QueryRow(ctx, `SELECT COUNT(*) FROM incidents WHERE monitor_id = $1`, monitorID).Scan(&count); err != nil {
		t.Fatalf("failed to count incidents after reopening: %v", err)
	}
	if count != 2 {
		t.Fatalf("expected a new incident after resolution, got %d incidents", count)
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

	firstResult := check.Result{
		StatusCode:  200,
		Latency:     100 * time.Millisecond,
		Success:     true,
		FailureType: check.FailureNone,
	}

	secondResult := check.Result{
		StatusCode:  500,
		Latency:     250 * time.Millisecond,
		Success:     false,
		FailureType: check.FailureHTTP,
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

	if checks[0].FailureType != check.FailureHTTP {
		t.Fatalf(
			"expected failure type %q, got %q",
			check.FailureHTTP,
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

func TestCheckRepositoryGetSummary(t *testing.T) {
	ctx := context.Background()

	db, err := pgxpool.New(
		ctx,
		"postgres://horus:horus@localhost:5432/horus",
	)
	if err != nil {
		t.Fatalf("failed to create database pool: %v", err)
	}
	defer db.Close()

	monitorID := "00000000-0000-0000-0000-000000000003"

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
			$2,
			$3,
			$4,
			$5,
			$6,
			$7,
			NOW(),
			NOW()
		)
		ON CONFLICT (id) DO NOTHING
		`,
		monitorID,
		"Summary Test",
		"https://example.com",
		60,
		5,
		200,
		true,
	)
	if err != nil {
		t.Fatalf("failed to create test monitor: %v", err)
	}

	_, err = db.Exec(
		ctx,
		`DELETE FROM checks WHERE monitor_id = $1`,
		monitorID,
	)
	if err != nil {
		t.Fatalf("failed to clean test checks: %v", err)
	}

	repository := NewCheckRepository(db)

	err = repository.Create(
		ctx,
		monitorID,
		check.Result{
			StatusCode:  200,
			Latency:     100 * time.Millisecond,
			Success:     true,
			FailureType: check.FailureNone,
		},
	)
	if err != nil {
		t.Fatalf("failed to create successful check: %v", err)
	}

	err = repository.Create(
		ctx,
		monitorID,
		check.Result{
			StatusCode:  500,
			Latency:     300 * time.Millisecond,
			Success:     false,
			FailureType: check.FailureHTTP,
		},
	)
	if err != nil {
		t.Fatalf("failed to create failed check: %v", err)
	}

	summary, err := repository.GetSummary(ctx, monitorID)
	if err != nil {
		t.Fatalf("failed to get summary: %v", err)
	}

	if summary.TotalChecks != 2 {
		t.Fatalf(
			"expected 2 total checks, got %d",
			summary.TotalChecks,
		)
	}

	if summary.SuccessfulChecks != 1 {
		t.Fatalf(
			"expected 1 successful check, got %d",
			summary.SuccessfulChecks,
		)
	}

	if summary.FailedChecks != 1 {
		t.Fatalf(
			"expected 1 failed check, got %d",
			summary.FailedChecks,
		)
	}

	if summary.AverageLatency != 200*time.Millisecond {
		t.Fatalf(
			"expected average latency of 200ms, got %s",
			summary.AverageLatency,
		)
	}

	if summary.LatestStatus != 500 {
		t.Fatalf(
			"expected latest status 500, got %d",
			summary.LatestStatus,
		)
	}
}

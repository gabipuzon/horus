package check_test

import (
	"context"
	"testing"
	"time"

	"github.com/gabipuzon/horus/internal/check"
	"github.com/gabipuzon/horus/internal/monitor"
	"github.com/gabipuzon/horus/internal/testutil"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestCheckRepositoryCreate(t *testing.T) {
	testutil.RequireIntegration(t)
	ctx := context.Background()

	db, err := pgxpool.New(
		ctx,
		testutil.DatabaseURL(),
	)
	if err != nil {
		t.Fatalf("failed to connect to database: %v", err)
	}
	defer db.Close()

	repository := check.NewRepository(db)

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

	startedAt := time.Now().UTC().Truncate(time.Microsecond)
	scheduledAt := startedAt.Add(-2 * time.Second)
	result := check.Result{
		StartedAt:   startedAt,
		ScheduledAt: scheduledAt,
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
		statusCode        int
		latencyMs         int64
		success           bool
		failureType       string
		storedStartedAt   time.Time
		storedScheduledAt time.Time
	)

	err = db.QueryRow(
		ctx,
		`
		SELECT
			status_code,
			latency_ms,
			success,
			failure_type,
			started_at,
			scheduled_at
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
		&storedStartedAt,
		&storedScheduledAt,
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
	if !storedStartedAt.Equal(startedAt) || !storedScheduledAt.Equal(scheduledAt) {
		t.Fatalf("stored timing changed: started=%s scheduled=%s", storedStartedAt, storedScheduledAt)
	}
}

func TestCheckRepositoryListByMonitor(t *testing.T) {
	testutil.RequireIntegration(t)
	ctx := context.Background()

	db, err := pgxpool.New(
		ctx,
		testutil.DatabaseURL(),
	)
	if err != nil {
		t.Fatalf("failed to connect to database: %v", err)
	}
	defer db.Close()

	repository := check.NewRepository(db)

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
	testutil.RequireIntegration(t)
	ctx := context.Background()

	db, err := pgxpool.New(
		ctx,
		testutil.DatabaseURL(),
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

	repository := check.NewRepository(db)

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

func TestCheckRepositoryHistoryAndSummarySemantics(t *testing.T) {
	testutil.RequireIntegration(t)
	ctx := context.Background()
	db, err := pgxpool.New(ctx, testutil.DatabaseURL())
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	monitorID := uuid.NewString()
	_, err = db.Exec(ctx, `
		INSERT INTO monitors (id, name, url, interval_seconds, timeout_seconds, expected_status, enabled, created_at, updated_at)
		VALUES ($1, 'Check Semantics Test', 'https://example.com', 60, 5, 200, false, NOW(), NOW())
	`, monitorID)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Exec(ctx, `DELETE FROM monitors WHERE id = $1`, monitorID)

	repository := check.NewRepository(db)
	monitors := monitor.NewRepository(db)
	for _, test := range []struct {
		id   string
		want bool
	}{
		{id: monitorID, want: true},
		{id: uuid.NewString()},
		{id: "not-a-uuid"},
	} {
		exists, err := monitors.Exists(ctx, test.id)
		if err != nil || exists != test.want {
			t.Fatalf("monitor existence for %q: want %t, got %t, err %v", test.id, test.want, exists, err)
		}
	}
	assertSummary := func(want check.Summary, wantUptime *float64) {
		t.Helper()
		got, err := repository.GetSummary(ctx, monitorID)
		if err != nil || got != want {
			t.Fatalf("summary: want %+v, got %+v, err %v", want, got, err)
		}
		percentage := got.UptimePercentage()
		if (percentage == nil) != (wantUptime == nil) || (percentage != nil && *percentage != *wantUptime) {
			t.Fatalf("uptime: want %v, got %v", wantUptime, percentage)
		}
	}

	assertSummary(check.Summary{}, nil)
	empty, err := repository.ListByMonitor(ctx, monitorID, 50, 0)
	if err != nil || len(empty) != 0 {
		t.Fatalf("expected empty history, got %+v, %v", empty, err)
	}

	insert := func(id string, status any, latencyMs int, success bool, failureType string, failureMessage any, checkedAt time.Time) {
		t.Helper()
		_, err := db.Exec(ctx, `
			INSERT INTO checks (id, monitor_id, status_code, latency_ms, success, failure_type, error, checked_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		`, id, monitorID, status, latencyMs, success, failureType, failureMessage, checkedAt)
		if err != nil {
			t.Fatal(err)
		}
	}

	olderID := uuid.NewString()
	checkedAt := time.Date(2026, 9, 25, 13, 0, 0, 0, time.UTC)
	insert(olderID, 200, 100, true, "", nil, checkedAt)
	allSuccess := float64(100)
	assertSummary(check.Summary{TotalChecks: 1, SuccessfulChecks: 1, AverageLatency: 100 * time.Millisecond, LatestStatus: 200}, &allSuccess)

	lowID, highID := uuid.NewString(), uuid.NewString()
	if lowID > highID {
		lowID, highID = highID, lowID
	}
	newerAt := checkedAt.Add(time.Minute)
	insert(lowID, 500, 101, false, "http", "unexpected status", newerAt)
	mixed := float64(50)
	// AVG(100, 101) is 100.5 ms; the API's integer milliseconds truncate it.
	assertSummary(check.Summary{TotalChecks: 2, SuccessfulChecks: 1, FailedChecks: 1, AverageLatency: 100 * time.Millisecond, LatestStatus: 500}, &mixed)

	insert(highID, nil, 301, false, "network", nil, newerAt)
	mixed = 33.33
	assertSummary(check.Summary{TotalChecks: 3, SuccessfulChecks: 1, FailedChecks: 2, AverageLatency: 167 * time.Millisecond, LatestStatus: 0}, &mixed)
	history, err := repository.ListByMonitor(ctx, monitorID, 50, 0)
	if err != nil || len(history) != 3 {
		t.Fatalf("expected three checks, got %+v, %v", history, err)
	}
	if history[0].ID != highID || history[1].ID != lowID || history[2].ID != olderID {
		t.Fatalf("expected timestamp and ID descending order, got %+v", history)
	}
	if history[0].StatusCode != 0 || history[0].Error != "" || history[1].Error != "unexpected status" {
		t.Fatalf("expected safe nullable failure fields, got %+v", history)
	}
	page, err := repository.ListByMonitor(ctx, monitorID, 1, 1)
	if err != nil || len(page) != 1 || page[0].ID != lowID {
		t.Fatalf("expected the second check on page two, got %+v, %v", page, err)
	}

	if _, err := db.Exec(ctx, `DELETE FROM checks WHERE id = $1`, olderID); err != nil {
		t.Fatal(err)
	}
	allFailed := float64(0)
	assertSummary(check.Summary{TotalChecks: 2, FailedChecks: 2, AverageLatency: 201 * time.Millisecond, LatestStatus: 0}, &allFailed)
}

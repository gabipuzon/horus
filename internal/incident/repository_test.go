package incident_test

import (
	"context"
	"testing"
	"time"

	"github.com/gabipuzon/horus/internal/incident"
	"github.com/gabipuzon/horus/internal/monitor"
	"github.com/gabipuzon/horus/internal/testutil"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestIncidentRepositoryHistoryAndCurrent(t *testing.T) {
	testutil.RequireIntegration(t)
	ctx := context.Background()
	db, err := pgxpool.New(ctx, testutil.DatabaseURL())
	if err != nil {
		t.Fatalf("failed to connect to database: %v", err)
	}
	defer db.Close()

	monitorID := uuid.NewString()
	_, err = db.Exec(ctx, `
		INSERT INTO monitors (id, name, url, interval_seconds, timeout_seconds, expected_status, enabled, created_at, updated_at)
		VALUES ($1, 'Incident Read Test', 'https://example.com', 60, 5, 200, true, NOW(), NOW())
	`, monitorID)
	if err != nil {
		t.Fatalf("failed to create test monitor: %v", err)
	}
	defer db.Exec(ctx, `DELETE FROM monitors WHERE id = $1`, monitorID)

	monitors := monitor.NewRepository(db)
	for _, test := range []struct {
		id   string
		want bool
	}{
		{id: monitorID, want: true},
		{id: uuid.NewString(), want: false},
		{id: "invalid-id", want: false},
	} {
		exists, err := monitors.Exists(ctx, test.id)
		if err != nil || exists != test.want {
			t.Fatalf("Exists(%q): expected %t, got %t, %v", test.id, test.want, exists, err)
		}
	}

	repository := incident.NewRepository(db)
	values, err := repository.ListByMonitor(ctx, monitorID, 50, 0)
	if err != nil || len(values) != 0 {
		t.Fatalf("expected empty incident history, got %v, %v", values, err)
	}
	current, err := repository.GetOpenByMonitor(ctx, monitorID)
	if err != nil || current != nil {
		t.Fatalf("expected no current incident, got %v, %v", current, err)
	}

	checks := incident.NewRepository(db)
	started := time.Now().Add(-2 * time.Minute).Truncate(time.Microsecond)
	if opened, err := checks.OpenIncident(ctx, incident.Incident{
		MonitorID: monitorID, StartedAt: started, FailureType: "http", StatusCode: 503,
	}); err != nil || opened == nil {
		t.Fatalf("failed to open first incident: %v", err)
	}
	if resolved, err := checks.ResolveIncident(ctx, monitorID); err != nil || resolved == nil {
		t.Fatalf("failed to resolve first incident: %v", err)
	}
	if opened, err := checks.OpenIncident(ctx, incident.Incident{
		MonitorID: monitorID, StartedAt: started.Add(time.Minute), FailureType: "network", FailureMessage: "connection refused",
	}); err != nil || opened == nil {
		t.Fatalf("failed to open second incident: %v", err)
	}

	current, err = repository.GetOpenByMonitor(ctx, monitorID)
	if err != nil || current == nil {
		t.Fatalf("expected current incident, got %v, %v", current, err)
	}
	if current.ResolvedAt != nil || current.FailureType != "network" || current.FailureMessage != "connection refused" || !current.StartedAt.Equal(started.Add(time.Minute)) {
		t.Fatalf("unexpected current incident: %+v", current)
	}
	secondID := current.ID
	values, err = repository.ListByMonitor(ctx, monitorID, 1, 0)
	if err != nil || len(values) != 1 || values[0].ID != current.ID {
		t.Fatalf("expected newest incident on first page, got %v, %v", values, err)
	}
	values, err = repository.ListByMonitor(ctx, monitorID, 1, 1)
	if err != nil || len(values) != 1 || values[0].ResolvedAt == nil || values[0].FailureType != "http" || values[0].StatusCode != 503 {
		t.Fatalf("expected resolved incident on second page, got %v, %v", values, err)
	}
	if resolved, err := checks.ResolveIncident(ctx, monitorID); err != nil || resolved == nil {
		t.Fatalf("failed to resolve current incident: %v", err)
	}
	current, err = repository.GetOpenByMonitor(ctx, monitorID)
	if err != nil || current != nil {
		t.Fatalf("expected no current incident after resolution, got %v, %v", current, err)
	}

	// A later opening may share a persisted timestamp with a resolved incident.
	// History must use the same stable UUID tie-breaker on every read.
	tiedAt := started.Add(time.Minute)
	third, err := checks.OpenIncident(ctx, incident.Incident{
		MonitorID: monitorID, StartedAt: tiedAt, FailureType: "network",
	})
	if err != nil || third == nil {
		t.Fatalf("failed to open tied incident: %+v, %v", third, err)
	}
	if duplicate, err := checks.OpenIncident(ctx, incident.Incident{
		MonitorID: monitorID, StartedAt: tiedAt.Add(time.Second), FailureType: "http", StatusCode: 500, FailureMessage: "later failure",
	}); err != nil || duplicate != nil {
		t.Fatalf("expected repeated failure to preserve the open incident, got %+v, %v", duplicate, err)
	}
	current, err = repository.GetOpenByMonitor(ctx, monitorID)
	if err != nil || current == nil || current.ID != third.ID || current.ResolvedAt != nil || current.StatusCode != 0 || current.FailureMessage != "" || current.FailureType != "network" {
		t.Fatalf("expected current open incident with initial empty failure context, got %+v, %v", current, err)
	}
	values, err = repository.ListByMonitor(ctx, monitorID, 50, 0)
	if err != nil || len(values) != 3 {
		t.Fatalf("expected three incidents, got %+v, %v", values, err)
	}
	wantFirst, wantSecond := third.ID, secondID
	if wantFirst < wantSecond {
		wantFirst, wantSecond = wantSecond, wantFirst
	}
	if values[0].ID != wantFirst || values[1].ID != wantSecond || !values[0].StartedAt.Equal(tiedAt) || !values[1].StartedAt.Equal(tiedAt) {
		t.Fatalf("expected tied incidents in descending ID order, got %+v", values)
	}
	page, err := repository.ListByMonitor(ctx, monitorID, 1, 1)
	if err != nil || len(page) != 1 || page[0].ID != wantSecond {
		t.Fatalf("expected stable second page for tied incidents, got %+v, %v", page, err)
	}
}

func TestIncidentRepositoryLifecycle(t *testing.T) {
	testutil.RequireIntegration(t)
	ctx := context.Background()
	db, err := pgxpool.New(ctx, testutil.DatabaseURL())
	if err != nil {
		t.Fatalf("failed to connect to database: %v", err)
	}
	defer db.Close()

	repository := incident.NewRepository(db)
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

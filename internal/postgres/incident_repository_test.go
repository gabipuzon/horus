package postgres

import (
	"context"
	"testing"
	"time"

	"github.com/gabipuzon/horus/internal/incident"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestIncidentRepositoryHistoryAndCurrent(t *testing.T) {
	ctx := context.Background()
	db, err := pgxpool.New(ctx, "postgres://horus:horus@localhost:5432/horus")
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

	monitors := NewMonitorRepository(db)
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

	repository := NewIncidentRepository(db)
	values, err := repository.ListByMonitor(ctx, monitorID, 50, 0)
	if err != nil || len(values) != 0 {
		t.Fatalf("expected empty incident history, got %v, %v", values, err)
	}
	current, err := repository.GetOpenByMonitor(ctx, monitorID)
	if err != nil || current != nil {
		t.Fatalf("expected no current incident, got %v, %v", current, err)
	}

	checks := NewCheckRepository(db)
	started := time.Now().Add(-2 * time.Minute).Truncate(time.Microsecond)
	if err := checks.OpenIncident(ctx, incident.Incident{
		MonitorID: monitorID, StartedAt: started, FailureType: "http", StatusCode: 503,
	}); err != nil {
		t.Fatalf("failed to open first incident: %v", err)
	}
	if err := checks.ResolveIncident(ctx, monitorID); err != nil {
		t.Fatalf("failed to resolve first incident: %v", err)
	}
	if err := checks.OpenIncident(ctx, incident.Incident{
		MonitorID: monitorID, StartedAt: started.Add(time.Minute), FailureType: "network", FailureMessage: "connection refused",
	}); err != nil {
		t.Fatalf("failed to open second incident: %v", err)
	}

	current, err = repository.GetOpenByMonitor(ctx, monitorID)
	if err != nil || current == nil {
		t.Fatalf("expected current incident, got %v, %v", current, err)
	}
	if current.ResolvedAt != nil || current.FailureType != "network" || current.FailureMessage != "connection refused" || !current.StartedAt.Equal(started.Add(time.Minute)) {
		t.Fatalf("unexpected current incident: %+v", current)
	}
	values, err = repository.ListByMonitor(ctx, monitorID, 1, 0)
	if err != nil || len(values) != 1 || values[0].ID != current.ID {
		t.Fatalf("expected newest incident on first page, got %v, %v", values, err)
	}
	values, err = repository.ListByMonitor(ctx, monitorID, 1, 1)
	if err != nil || len(values) != 1 || values[0].ResolvedAt == nil || values[0].FailureType != "http" || values[0].StatusCode != 503 {
		t.Fatalf("expected resolved incident on second page, got %v, %v", values, err)
	}
	if err := checks.ResolveIncident(ctx, monitorID); err != nil {
		t.Fatalf("failed to resolve current incident: %v", err)
	}
	current, err = repository.GetOpenByMonitor(ctx, monitorID)
	if err != nil || current != nil {
		t.Fatalf("expected no current incident after resolution, got %v, %v", current, err)
	}
}

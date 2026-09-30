package postgres

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/gabipuzon/horus/internal/monitor"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestMonitorRepositoryMissingAndAffectedRows(t *testing.T) {
	ctx := context.Background()
	db, err := pgxpool.New(ctx, "postgres://horus:horus@localhost:5432/horus")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	repository := NewMonitorRepository(db)

	m, err := monitor.New("Repository Test", "https://example.com", time.Minute, 5*time.Second, 200)
	if err != nil {
		t.Fatal(err)
	}
	m.Enabled = false
	m.NextCheckAt = time.Now().Add(time.Hour)
	if err := repository.Create(ctx, m); err != nil {
		t.Fatal(err)
	}
	defer db.Exec(ctx, `DELETE FROM monitors WHERE id = $1`, m.ID)

	got, err := repository.GetByID(ctx, m.ID)
	if err != nil || got.ID != m.ID || got.Enabled {
		t.Fatalf("expected stored disabled monitor, got %+v, %v", got, err)
	}
	if err := repository.SetEnabled(ctx, m.ID, true); err != nil {
		t.Fatal(err)
	}
	if err := repository.SetEnabled(ctx, m.ID, true); err != nil {
		t.Fatalf("setting an existing monitor to its current state should succeed: %v", err)
	}
	got, err = repository.GetByID(ctx, m.ID)
	if err != nil || !got.Enabled {
		t.Fatalf("expected enabled monitor, got %+v, %v", got, err)
	}

	missingID := uuid.NewString()
	for _, id := range []string{missingID, "not-a-uuid"} {
		if got, err := repository.GetByID(ctx, id); got != nil || !errors.Is(err, monitor.ErrNotFound) {
			t.Fatalf("GetByID(%q): expected not found, got %+v, %v", id, got, err)
		}
		if err := repository.Delete(ctx, id); !errors.Is(err, monitor.ErrNotFound) {
			t.Fatalf("Delete(%q): expected not found, got %v", id, err)
		}
		if err := repository.SetEnabled(ctx, id, false); !errors.Is(err, monitor.ErrNotFound) {
			t.Fatalf("SetEnabled(%q): expected not found, got %v", id, err)
		}
	}

	canceled, cancel := context.WithCancel(ctx)
	cancel()
	if got, err := repository.GetByID(canceled, m.ID); got != nil || err == nil || errors.Is(err, monitor.ErrNotFound) {
		t.Fatalf("expected database failure from GetByID, got %+v, %v", got, err)
	}
	if err := repository.Delete(canceled, m.ID); err == nil || errors.Is(err, monitor.ErrNotFound) {
		t.Fatalf("expected database failure from Delete, got %v", err)
	}
	if err := repository.SetEnabled(canceled, m.ID, false); err == nil || errors.Is(err, monitor.ErrNotFound) {
		t.Fatalf("expected database failure from SetEnabled, got %v", err)
	}

	if err := repository.Delete(ctx, m.ID); err != nil {
		t.Fatal(err)
	}
	if err := repository.Delete(ctx, m.ID); !errors.Is(err, monitor.ErrNotFound) {
		t.Fatalf("expected second delete to report not found, got %v", err)
	}
}

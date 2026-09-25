package monitor

import (
	"context"
	"testing"
	"time"
)

type fakeMonitorSchedulerRepository struct {
	monitors []*Monitor
}

func (f *fakeMonitorSchedulerRepository) List(
	ctx context.Context,
) ([]*Monitor, error) {
	return f.monitors, nil
}

func TestSchedulerSchedule(t *testing.T) {
	now := time.Now()

	dueMonitor, err := New(
		"Due",
		"https://example.com",
		60*time.Second,
		5*time.Second,
		200,
	)
	if err != nil {
		t.Fatalf("failed to create due monitor: %v", err)
	}

	dueMonitor.NextCheckAt = now.Add(-2 * time.Minute)
	notDueMonitor, err := New(
		"Not Due",
		"https://example.com",
		60*time.Second,
		5*time.Second,
		200,
	)
	if err != nil {
		t.Fatalf("failed to create not-due monitor: %v", err)
	}

	notDueMonitor.NextCheckAt = now.Add(10 * time.Second)
	disabledMonitor, err := New(
		"Disabled",
		"https://example.com",
		60*time.Second,
		5*time.Second,
		200,
	)
	if err != nil {
		t.Fatalf("failed to create disabled monitor: %v", err)
	}

	disabledMonitor.NextCheckAt = now.Add(-2 * time.Minute)
	disabledMonitor.Enabled = false

	repository := &fakeMonitorSchedulerRepository{
		monitors: []*Monitor{
			dueMonitor,
			notDueMonitor,
			disabledMonitor,
		},
	}

	scheduler := NewScheduler(repository)

	due, err := scheduler.schedule(context.Background())
	if err != nil {
		t.Fatalf("schedule failed: %v", err)
	}

	if len(due) != 1 {
		t.Fatalf("expected 1 due monitor, got %d", len(due))
	}

	if due[0].ID != dueMonitor.ID {
		t.Fatalf(
			"expected due monitor %q, got %q",
			dueMonitor.ID,
			due[0].ID,
		)
	}
}

func (f *fakeMonitorSchedulerRepository) SetNextCheckAt(
	ctx context.Context,
	id string,
	nextCheckAt time.Time,
) error {
	for _, m := range f.monitors {
		if m.ID == id {
			m.NextCheckAt = nextCheckAt
			return nil
		}
	}

	return nil
}

func TestSchedulerAdvancesNextCheckAt(t *testing.T) {
	now := time.Now()

	monitor, err := New(
		"Example",
		"https://example.com",
		60*time.Second,
		5*time.Second,
		200,
	)
	if err != nil {
		t.Fatalf("failed to create monitor: %v", err)
	}

	monitor.NextCheckAt = now.Add(-10 * time.Second)

	repository := &fakeMonitorSchedulerRepository{
		monitors: []*Monitor{monitor},
	}

	scheduler := NewScheduler(repository)

	_, err = scheduler.schedule(context.Background())
	if err != nil {
		t.Fatalf("schedule failed: %v", err)
	}

	expected := now.Add(50 * time.Second)

	if monitor.NextCheckAt.Before(expected.Add(-time.Second)) ||
		monitor.NextCheckAt.After(expected.Add(time.Second)) {
		t.Fatalf(
			"expected next check around %s, got %s",
			expected,
			monitor.NextCheckAt,
		)
	}
}

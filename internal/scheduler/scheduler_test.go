package scheduler

import (
	"context"
	"testing"
	"time"

	"github.com/gabipuzon/horus/internal/monitor"
	"github.com/gabipuzon/horus/internal/queue"
)

type fakeMonitorSchedulerRepository struct {
	monitors []*monitor.Monitor
}

func (f *fakeMonitorSchedulerRepository) List(
	ctx context.Context,
) ([]*monitor.Monitor, error) {
	return f.monitors, nil
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

type fakeCheckQueue struct {
	jobs []queue.CheckJob
}

func (f *fakeCheckQueue) EnqueueCheck(
	ctx context.Context,
	job queue.CheckJob,
) error {
	f.jobs = append(f.jobs, job)
	return nil
}

func TestSchedulerSchedule(t *testing.T) {
	now := time.Now()

	dueMonitor, err := monitor.New(
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

	notDueMonitor, err := monitor.New(
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

	disabledMonitor, err := monitor.New(
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
		monitors: []*monitor.Monitor{
			dueMonitor,
			notDueMonitor,
			disabledMonitor,
		},
	}

	checkQueue := &fakeCheckQueue{}

	scheduler := New(
		repository,
		checkQueue,
	)

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

	if len(checkQueue.jobs) != 1 {
		t.Fatalf(
			"expected 1 queued job, got %d",
			len(checkQueue.jobs),
		)
	}

	if checkQueue.jobs[0].MonitorID != dueMonitor.ID {
		t.Fatalf(
			"expected queued monitor %q, got %q",
			dueMonitor.ID,
			checkQueue.jobs[0].MonitorID,
		)
	}
}

func TestSchedulerAdvancesNextCheckAt(t *testing.T) {
	now := time.Now()

	mon, err := monitor.New(
		"Example",
		"https://example.com",
		60*time.Second,
		5*time.Second,
		200,
	)
	if err != nil {
		t.Fatalf("failed to create monitor: %v", err)
	}

	mon.NextCheckAt = now.Add(-10 * time.Second)

	repository := &fakeMonitorSchedulerRepository{
		monitors: []*monitor.Monitor{mon},
	}

	checkQueue := &fakeCheckQueue{}

	scheduler := New(
		repository,
		checkQueue,
	)

	_, err = scheduler.schedule(context.Background())
	if err != nil {
		t.Fatalf("schedule failed: %v", err)
	}

	expected := now.Add(50 * time.Second)

	if mon.NextCheckAt.Before(expected.Add(-time.Second)) ||
		mon.NextCheckAt.After(expected.Add(time.Second)) {
		t.Fatalf(
			"expected next check around %s, got %s",
			expected,
			mon.NextCheckAt,
		)
	}

	if len(checkQueue.jobs) != 1 {
		t.Fatalf(
			"expected 1 queued job, got %d",
			len(checkQueue.jobs),
		)
	}

	if checkQueue.jobs[0].MonitorID != mon.ID {
		t.Fatalf(
			"expected queued monitor %q, got %q",
			mon.ID,
			checkQueue.jobs[0].MonitorID,
		)
	}
}

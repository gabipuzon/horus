package monitor

import (
	"context"
	"net/http"
	"net/http/httptest"
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

type fakeSchedulerCheckRepository struct {
	called    bool
	monitorID string
	result    CheckResult
}

func (f *fakeSchedulerCheckRepository) Create(
	ctx context.Context,
	monitorID string,
	result CheckResult,
) error {
	f.called = true
	f.monitorID = monitorID
	f.result = result

	return nil
}

func TestSchedulerSchedule(t *testing.T) {
	now := time.Now()

	server := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}),
	)
	defer server.Close()

	dueMonitor, err := New(
		"Due",
		server.URL,
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
		server.URL,
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
		server.URL,
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

	checkRepository := &fakeSchedulerCheckRepository{}

	checker := NewChecker(http.DefaultClient)

	checkService := NewCheckService(
		checker,
		checkRepository,
	)

	scheduler := NewScheduler(
		repository,
		checkService,
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

	if !checkRepository.called {
		t.Fatal("expected check service to run check")
	}

	if checkRepository.monitorID != dueMonitor.ID {
		t.Fatalf(
			"expected check for monitor %q, got %q",
			dueMonitor.ID,
			checkRepository.monitorID,
		)
	}

	if !checkRepository.result.Success {
		t.Fatal("expected check to succeed")
	}
}

func TestSchedulerAdvancesNextCheckAt(t *testing.T) {
	now := time.Now()

	server := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}),
	)
	defer server.Close()

	monitor, err := New(
		"Example",
		server.URL,
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

	checkRepository := &fakeSchedulerCheckRepository{}

	checker := NewChecker(http.DefaultClient)

	checkService := NewCheckService(
		checker,
		checkRepository,
	)

	scheduler := NewScheduler(
		repository,
		checkService,
	)

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

package monitor

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

type fakeWorkerCheckRepository struct {
	called    chan struct{}
	monitorID string
}

func (f *fakeWorkerCheckRepository) Create(
	ctx context.Context,
	monitorID string,
	result CheckResult,
) error {
	f.monitorID = monitorID

	select {
	case f.called <- struct{}{}:
	default:
	}

	return nil
}

func TestCheckWorkerPoolSubmitsMonitor(t *testing.T) {
	server := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}),
	)
	defer server.Close()

	monitor, err := New(
		"Example",
		server.URL,
		time.Minute,
		5*time.Second,
		200,
	)
	if err != nil {
		t.Fatalf("failed to create monitor: %v", err)
	}

	repository := &fakeWorkerCheckRepository{
		called: make(chan struct{}, 1),
	}

	checker := NewChecker(http.DefaultClient)

	checkService := NewCheckService(
		checker,
		repository,
	)

	pool := NewCheckWorkerPool(
		1,
		checkService,
	)

	pool.Submit(monitor)

	select {
	case <-repository.called:
	case <-time.After(time.Second):
		t.Fatal("expected worker to execute check")
	}

	if repository.monitorID != monitor.ID {
		t.Fatalf(
			"expected monitor ID %q, got %q",
			monitor.ID,
			repository.monitorID,
		)
	}
}

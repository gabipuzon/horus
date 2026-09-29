package worker

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gabipuzon/horus/internal/check"
	"github.com/gabipuzon/horus/internal/monitor"
	"github.com/gabipuzon/horus/internal/queue"
)

type fakeWorkerCheckRepository struct {
	called    chan struct{}
	monitorID string
}

func (f *fakeWorkerCheckRepository) Create(
	ctx context.Context,
	monitorID string,
	result check.Result,
) error {
	f.monitorID = monitorID

	select {
	case f.called <- struct{}{}:
	default:
	}

	return nil
}

type fakeWorkerQueue struct {
	jobs chan queue.CheckJob
}

func (f *fakeWorkerQueue) DequeueCheck(
	ctx context.Context,
) (queue.CheckJob, error) {
	select {
	case <-ctx.Done():
		return queue.CheckJob{}, ctx.Err()
	case job := <-f.jobs:
		return job, nil
	}
}

type fakeWorkerMonitorRepository struct {
	monitor *monitor.Monitor
}

func (f *fakeWorkerMonitorRepository) GetByID(
	ctx context.Context,
	id string,
) (*monitor.Monitor, error) {
	if f.monitor.ID != id {
		return nil, context.DeadlineExceeded
	}

	return f.monitor, nil
}

func TestCheckWorkerPoolProcessesQueuedMonitor(t *testing.T) {
	server := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}),
	)
	defer server.Close()

	m, err := monitor.New(
		"Example",
		server.URL,
		time.Minute,
		5*time.Second,
		200,
	)
	if err != nil {
		t.Fatalf("failed to create monitor: %v", err)
	}

	checkRepository := &fakeWorkerCheckRepository{
		called: make(chan struct{}, 1),
	}

	checker := check.NewChecker(http.DefaultClient)
	checkService := check.NewService(checker, checkRepository)

	checkQueue := &fakeWorkerQueue{
		jobs: make(chan queue.CheckJob, 1),
	}

	monitorRepository := &fakeWorkerMonitorRepository{
		monitor: m,
	}

	checkQueue.jobs <- queue.CheckJob{
		MonitorID: m.ID,
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	pool := NewPool(
		ctx,
		3,
		checkQueue,
		monitorRepository,
		checkService,
	)
	defer pool.Shutdown()

	select {
	case <-checkRepository.called:
	case <-time.After(2 * time.Second):
		t.Fatal("expected worker to execute check")
	}

	if checkRepository.monitorID != m.ID {
		t.Fatalf(
			"expected monitor ID %q, got %q",
			m.ID,
			checkRepository.monitorID,
		)
	}
}

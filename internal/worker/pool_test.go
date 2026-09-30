package worker

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gabipuzon/horus/internal/check"
	"github.com/gabipuzon/horus/internal/incident"
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

func (f *fakeWorkerCheckRepository) OpenIncident(context.Context, incident.Incident) error {
	return nil
}

func (f *fakeWorkerCheckRepository) ResolveIncident(context.Context, string) error {
	return nil
}

type fakeWorkerQueue struct {
	jobs    chan queue.CheckJob
	results chan workerDequeueResult
	calls   chan struct{}
}

type workerDequeueResult struct {
	job queue.CheckJob
	err error
}

func (f *fakeWorkerQueue) DequeueCheck(
	ctx context.Context,
) (queue.CheckJob, error) {
	if f.calls != nil {
		select {
		case f.calls <- struct{}{}:
		default:
		}
	}
	if f.results != nil {
		select {
		case <-ctx.Done():
			return queue.CheckJob{}, ctx.Err()
		case result := <-f.results:
			return result.job, result.err
		}
	}

	select {
	case <-ctx.Done():
		return queue.CheckJob{}, ctx.Err()
	case job := <-f.jobs:
		return job, nil
	}
}

func TestWorkerRetriesDequeueFailureAndProcessesNextJob(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	m, err := monitor.New("Example", server.URL, time.Minute, 5*time.Second, 200)
	if err != nil {
		t.Fatalf("failed to create monitor: %v", err)
	}
	checkRepository := &fakeWorkerCheckRepository{called: make(chan struct{}, 1)}
	checkService := check.NewService(check.NewChecker(http.DefaultClient), checkRepository)
	checkQueue := &fakeWorkerQueue{
		results: make(chan workerDequeueResult, 2),
		calls:   make(chan struct{}, 3),
	}
	checkQueue.results <- workerDequeueResult{err: errors.New("temporary Redis failure")}
	checkQueue.results <- workerDequeueResult{job: queue.CheckJob{MonitorID: m.ID}}
	pool := NewPool(context.Background(), 1, checkQueue, &fakeWorkerMonitorRepository{monitor: m}, checkService)
	defer pool.Shutdown()

	for i := 0; i < 3; i++ {
		select {
		case <-checkQueue.calls:
		case <-time.After(2 * time.Second):
			t.Fatalf("expected dequeue attempt %d", i+1)
		}
	}
	select {
	case <-checkRepository.called:
	case <-time.After(2 * time.Second):
		t.Fatal("expected worker to process job after retry")
	}
}

func TestWorkerCancellationStopsBackoff(t *testing.T) {
	checkQueue := &fakeWorkerQueue{
		results: make(chan workerDequeueResult, 1),
		calls:   make(chan struct{}, 1),
	}
	checkQueue.results <- workerDequeueResult{err: errors.New("temporary Redis failure")}
	ctx, cancel := context.WithCancel(context.Background())
	pool := NewPool(ctx, 1, checkQueue, nil, nil)

	select {
	case <-checkQueue.calls:
	case <-time.After(time.Second):
		cancel()
		pool.Shutdown()
		t.Fatal("expected worker to attempt dequeue")
	}
	// Give the worker time to leave DequeueCheck and enter its retry wait.
	time.Sleep(10 * time.Millisecond)
	cancel()
	done := make(chan struct{})
	go func() {
		pool.Shutdown()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(100 * time.Millisecond):
		t.Fatal("expected cancellation to stop worker during backoff")
	}
}

func TestDequeueBackoffResetsAfterSuccess(t *testing.T) {
	var backoff dequeueBackoff
	if got := backoff.next(); got != 250*time.Millisecond {
		t.Fatalf("expected initial backoff 250ms, got %s", got)
	}
	if got := backoff.next(); got != 500*time.Millisecond {
		t.Fatalf("expected second backoff 500ms, got %s", got)
	}
	backoff.reset()
	if got := backoff.next(); got != 250*time.Millisecond {
		t.Fatalf("expected reset backoff 250ms, got %s", got)
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

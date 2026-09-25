package monitor

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

type fakeCheckRepository struct {
	monitorID string
	result    CheckResult
	called    bool
}

func (f *fakeCheckRepository) Create(
	ctx context.Context,
	monitorID string,
	result CheckResult,
) error {
	f.monitorID = monitorID
	f.result = result
	f.called = true

	return nil
}

func TestCheckServiceCheck(t *testing.T) {
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

	checker := NewChecker(server.Client())
	repository := &fakeCheckRepository{}

	service := NewCheckService(checker, repository)

	result, err := service.Check(
		context.Background(),
		monitor,
	)
	if err != nil {
		t.Fatalf("check failed: %v", err)
	}

	if !result.Success {
		t.Fatal("expected check to succeed")
	}

	if result.StatusCode != http.StatusOK {
		t.Fatalf(
			"expected status code %d, got %d",
			http.StatusOK,
			result.StatusCode,
		)
	}

	if !repository.called {
		t.Fatal("expected repository Create to be called")
	}

	if repository.monitorID != monitor.ID {
		t.Fatalf(
			"expected monitor ID %q, got %q",
			monitor.ID,
			repository.monitorID,
		)
	}

	if !repository.result.Success {
		t.Fatal("expected persisted result to be successful")
	}

	if repository.result.StatusCode != http.StatusOK {
		t.Fatalf(
			"expected persisted status code %d, got %d",
			http.StatusOK,
			repository.result.StatusCode,
		)
	}
}

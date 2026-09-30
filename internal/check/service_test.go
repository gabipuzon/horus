package check

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gabipuzon/horus/internal/incident"
	"github.com/gabipuzon/horus/internal/monitor"
)

type fakeCheckRepository struct {
	monitorID  string
	result     Result
	called     bool
	events     []string
	incidents  []incident.Incident
	openErr    error
	resolveErr error
}

func (f *fakeCheckRepository) Create(
	ctx context.Context,
	monitorID string,
	result Result,
) error {
	f.monitorID = monitorID
	f.result = result
	f.called = true
	f.events = append(f.events, "check")

	return nil
}

func (f *fakeCheckRepository) OpenIncident(
	ctx context.Context,
	value incident.Incident,
) error {
	f.events = append(f.events, "open")
	if f.openErr != nil {
		return f.openErr
	}
	f.incidents = append(f.incidents, value)
	return nil
}

func (f *fakeCheckRepository) ResolveIncident(
	ctx context.Context,
	monitorID string,
) error {
	f.events = append(f.events, "resolve")
	return f.resolveErr
}

func TestCheckServiceCheck(t *testing.T) {
	server := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}),
	)
	defer server.Close()

	mon, err := monitor.New(
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
	checker.dialContext = server.Client().Transport.(*http.Transport).DialContext
	repository := &fakeCheckRepository{}

	service := NewService(checker, repository)

	result, err := service.Check(
		context.Background(),
		mon,
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

	if repository.monitorID != mon.ID {
		t.Fatalf(
			"expected monitor ID %q, got %q",
			mon.ID,
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

func TestCheckServiceIncidentLifecycle(t *testing.T) {
	var request int
	statuses := []int{http.StatusServiceUnavailable, http.StatusServiceUnavailable, http.StatusOK}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(statuses[request])
		request++
	}))
	defer server.Close()

	mon, err := monitor.New("Example", server.URL, time.Minute, 5*time.Second, http.StatusOK)
	if err != nil {
		t.Fatalf("failed to create monitor: %v", err)
	}
	checker := NewChecker(server.Client())
	checker.dialContext = server.Client().Transport.(*http.Transport).DialContext
	repository := &fakeCheckRepository{}
	service := NewService(checker, repository)

	for i := 0; i < 3; i++ {
		if _, err := service.Check(context.Background(), mon); err != nil {
			t.Fatalf("check %d failed: %v", i+1, err)
		}
	}

	if got, want := len(repository.incidents), 2; got != want {
		t.Fatalf("expected failed checks to request incident opening twice, got %d", got)
	}
	for _, value := range repository.incidents {
		if value.MonitorID != mon.ID || value.FailureType != string(FailureHTTP) || value.StatusCode != http.StatusServiceUnavailable {
			t.Fatalf("unexpected incident failure context: %+v", value)
		}
	}
	if got, want := repository.events, []string{"check", "open", "check", "open", "check", "resolve"}; !equalStrings(got, want) {
		t.Fatalf("expected check persistence before incident transitions, got %v", got)
	}
}

func TestCheckServiceReportsIncidentTransitionFailureAfterPersistingCheck(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()
	mon, err := monitor.New("Example", server.URL, time.Minute, 5*time.Second, http.StatusOK)
	if err != nil {
		t.Fatalf("failed to create monitor: %v", err)
	}
	checker := NewChecker(server.Client())
	checker.dialContext = server.Client().Transport.(*http.Transport).DialContext
	repository := &fakeCheckRepository{openErr: context.DeadlineExceeded}
	service := NewService(checker, repository)

	_, err = service.Check(context.Background(), mon)
	if err == nil || !strings.Contains(err.Error(), "check result was persisted but incident opening failed") {
		t.Fatalf("expected explicit incident transition error, got %v", err)
	}
	if !repository.called || len(repository.events) != 2 || repository.events[0] != "check" || repository.events[1] != "open" {
		t.Fatalf("expected check persistence to precede failed incident transition, got %v", repository.events)
	}
}

func equalStrings(got, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range got {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}

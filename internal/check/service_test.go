package check

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gabipuzon/horus/internal/incident"
	"github.com/gabipuzon/horus/internal/monitor"
	"github.com/gabipuzon/horus/internal/notification"
)

type fakeCheckRepository struct {
	monitorID  string
	result     Result
	called     bool
	events     []string
	incidents  []incident.Incident
	open       *incident.Incident
	openErr    error
	resolveErr error
}

type fakeNotifier struct {
	events []notification.Event
}

func (f *fakeNotifier) Notify(ctx context.Context, event notification.Event) error {
	f.events = append(f.events, event)
	return nil
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
) (*incident.Incident, error) {
	f.events = append(f.events, "open")
	if f.openErr != nil {
		return nil, f.openErr
	}
	if f.open != nil {
		return nil, nil
	}
	value.ID = "incident-1"
	f.open = &value
	f.incidents = append(f.incidents, value)
	return &value, nil
}

func (f *fakeCheckRepository) ResolveIncident(
	ctx context.Context,
	monitorID string,
) (*incident.Incident, error) {
	f.events = append(f.events, "resolve")
	if f.resolveErr != nil {
		return nil, f.resolveErr
	}
	if f.open == nil {
		return nil, nil
	}
	resolved := *f.open
	now := time.Now()
	resolved.ResolvedAt = &now
	f.open = nil
	return &resolved, nil
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

	service := NewService(checker, repository, nil)

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
	statuses := []int{http.StatusServiceUnavailable, http.StatusServiceUnavailable, http.StatusOK, http.StatusOK}
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
	notifier := &fakeNotifier{}
	service := NewService(checker, repository, notifier)

	for i := 0; i < len(statuses); i++ {
		if _, err := service.Check(context.Background(), mon); err != nil {
			t.Fatalf("check %d failed: %v", i+1, err)
		}
	}

	if got, want := len(repository.incidents), 1; got != want {
		t.Fatalf("expected repeated failures to create one incident, got %d", got)
	}
	for _, value := range repository.incidents {
		if value.MonitorID != mon.ID || value.FailureType != string(FailureHTTP) || value.StatusCode != http.StatusServiceUnavailable {
			t.Fatalf("unexpected incident failure context: %+v", value)
		}
	}
	if got, want := repository.events, []string{"check", "open", "check", "open", "check", "resolve", "check", "resolve"}; !equalStrings(got, want) {
		t.Fatalf("expected check persistence before incident transitions, got %v", got)
	}
	if len(notifier.events) != 2 || notifier.events[0].State != notification.Down || notifier.events[1].State != notification.Recovered {
		t.Fatalf("expected one DOWN and one RECOVERED notification, got %+v", notifier.events)
	}
	if notifier.events[0].Incident.ID != notifier.events[1].Incident.ID || notifier.events[0].MonitorName != mon.Name || notifier.events[0].MonitorURL != mon.URL || notifier.events[1].Incident.ResolvedAt == nil {
		t.Fatalf("expected incident and monitor context in notifications, got %+v", notifier.events)
	}
}

func TestCheckServiceNotificationFailureKeepsIncidentTransitions(t *testing.T) {
	webhook := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer webhook.Close()
	var status atomic.Int32
	status.Store(http.StatusServiceUnavailable)
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(int(status.Load()))
	}))
	defer target.Close()
	mon, err := monitor.New("Example", target.URL, time.Minute, 5*time.Second, http.StatusOK)
	if err != nil {
		t.Fatalf("failed to create monitor: %v", err)
	}
	checker := NewChecker(target.Client())
	checker.dialContext = target.Client().Transport.(*http.Transport).DialContext
	repository := &fakeCheckRepository{}
	service := NewService(checker, repository, notification.NewDiscord(webhook.URL, webhook.Client()))
	if _, err := service.Check(context.Background(), mon); err == nil || !strings.Contains(err.Error(), "incident was opened but notification failed") {
		t.Fatalf("expected notification failure after opening incident, got %v", err)
	}
	if !repository.called || repository.open == nil {
		t.Fatal("expected check and open incident to remain persisted")
	}
	status.Store(http.StatusOK)
	if _, err := service.Check(context.Background(), mon); err == nil || !strings.Contains(err.Error(), "incident was resolved but notification failed") {
		t.Fatalf("expected notification failure after resolving incident, got %v", err)
	}
	if repository.open != nil {
		t.Fatal("expected incident to remain resolved after notification failure")
	}
}

func TestCheckServiceWithoutNotifierStillOpensIncident(t *testing.T) {
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer target.Close()
	mon, err := monitor.New("Example", target.URL, time.Minute, 5*time.Second, http.StatusOK)
	if err != nil {
		t.Fatalf("failed to create monitor: %v", err)
	}
	checker := NewChecker(target.Client())
	checker.dialContext = target.Client().Transport.(*http.Transport).DialContext
	repository := &fakeCheckRepository{}
	service := NewService(checker, repository, nil)
	if _, err := service.Check(context.Background(), mon); err != nil {
		t.Fatalf("check failed without notifier: %v", err)
	}
	if repository.open == nil {
		t.Fatal("expected incident to open without webhook configuration")
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
	service := NewService(checker, repository, nil)

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

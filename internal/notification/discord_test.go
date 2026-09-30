package notification

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gabipuzon/horus/internal/incident"
)

func TestDiscordNotificationContent(t *testing.T) {
	requests := make(chan discordPayload, 2)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.Header.Get("Content-Type") != "application/json" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		var payload discordPayload
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		requests <- payload
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	started := time.Date(2026, 9, 30, 12, 0, 0, 0, time.UTC)
	recovered := started.Add(90 * time.Second)
	value := incident.Incident{
		MonitorID: "monitor-1", StartedAt: started, FailureType: "http", StatusCode: 503,
		FailureMessage: "bad gateway",
	}
	notifier := NewDiscord(server.URL, server.Client())
	for _, test := range []struct {
		state State
		value incident.Incident
		want  []string
	}{
		{state: Down, value: value, want: []string{"DOWN", "Example", "https://example.com", "http", "503", "bad gateway", started.Format(time.RFC3339)}},
		{state: Recovered, value: incident.Incident{
			MonitorID: value.MonitorID, StartedAt: started, ResolvedAt: &recovered,
			FailureType: value.FailureType, StatusCode: value.StatusCode, FailureMessage: value.FailureMessage,
		}, want: []string{"RECOVERED", "Example", "https://example.com", "http", "503", "bad gateway", started.Format(time.RFC3339), recovered.Format(time.RFC3339), "1m30s"}},
	} {
		if err := notifier.Notify(context.Background(), Event{
			State: test.state, MonitorName: "Example", MonitorURL: "https://example.com", Incident: test.value,
		}); err != nil {
			t.Fatalf("failed to send %s notification: %v", test.state, err)
		}
		payload := <-requests
		for _, part := range test.want {
			if !strings.Contains(payload.Content, part) {
				t.Fatalf("%s notification missing %q: %s", test.state, part, payload.Content)
			}
		}
		if payload.AllowedMentions.Parse == nil || len(payload.AllowedMentions.Parse) != 0 {
			t.Fatalf("expected mentions to be disabled, got %+v", payload.AllowedMentions)
		}
	}
}

func TestDiscordHTTPFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()
	err := NewDiscord(server.URL, server.Client()).Notify(context.Background(), Event{State: Down})
	if err == nil || !strings.Contains(err.Error(), "HTTP 500") {
		t.Fatalf("expected webhook HTTP error, got %v", err)
	}
}

func TestDiscordRespectsCancellationAndTimeout(t *testing.T) {
	release := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-release
	}))
	defer server.Close()
	defer close(release)
	notifier := NewDiscord(server.URL, server.Client())
	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	if err := notifier.Notify(canceled, Event{State: Down}); !errors.Is(err, context.Canceled) {
		t.Fatalf("expected cancellation, got %v", err)
	}
	notifier.timeout = 20 * time.Millisecond
	if err := notifier.Notify(context.Background(), Event{State: Down}); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected bounded request timeout, got %v", err)
	}
}

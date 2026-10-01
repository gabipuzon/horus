package api

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

type fakeIncidentRepository struct {
	values    []incident.Incident
	listErr   error
	openErr   error
	listCalls int
	gotLimit  int
	gotOffset int
}

func (f *fakeIncidentRepository) ListByMonitor(
	ctx context.Context, monitorID string, limit, offset int,
) ([]incident.Incident, error) {
	f.listCalls++
	f.gotLimit, f.gotOffset = limit, offset
	if f.listErr != nil {
		return nil, f.listErr
	}
	if offset >= len(f.values) {
		return nil, nil
	}
	end := offset + limit
	if end > len(f.values) {
		end = len(f.values)
	}
	return f.values[offset:end], nil
}

func TestIncidentHistoryPaginationDefaults(t *testing.T) {
	for _, test := range []struct {
		name       string
		query      string
		wantLimit  int
		wantOffset int
	}{
		{name: "defaults", wantLimit: 50},
		{name: "custom", query: "?limit=2&offset=3", wantLimit: 2, wantOffset: 3},
		{name: "maximum", query: "?limit=100", wantLimit: 100},
	} {
		t.Run(test.name, func(t *testing.T) {
			repository := &fakeIncidentRepository{}
			handler := NewIncidentHandler(repository, &fakeCheckRepository{})
			recorder := httptest.NewRecorder()
			handler.ListByMonitor(recorder, incidentRequest("/monitors/monitor-1/incidents"+test.query))
			if recorder.Code != http.StatusOK || repository.listCalls != 1 || repository.gotLimit != test.wantLimit || repository.gotOffset != test.wantOffset {
				t.Fatalf("expected 200 and pagination %d/%d, got %d and calls=%d limit=%d offset=%d", test.wantLimit, test.wantOffset, recorder.Code, repository.listCalls, repository.gotLimit, repository.gotOffset)
			}
		})
	}
}

func TestIncidentResponseDurationsAndNullableContext(t *testing.T) {
	started := time.Date(2026, 9, 25, 13, 0, 0, 0, time.UTC)
	open := incident.Incident{ID: "open", MonitorID: "monitor-1", StartedAt: started, FailureType: "network"}
	first := newIncidentResponse(open, started.Add(1500*time.Millisecond))
	later := newIncidentResponse(open, started.Add(3500*time.Millisecond))
	if !first.IsOpen || first.ResolvedAt != nil || first.DurationMs != 1500 || later.DurationMs != 3500 {
		t.Fatalf("unexpected open durations: first=%+v later=%+v", first, later)
	}
	if future := newIncidentResponse(open, started.Add(-time.Second)); future.DurationMs != 0 {
		t.Fatalf("expected future-start duration to clamp to zero, got %+v", future)
	}
	encoded, err := json.Marshal(first)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(encoded), `"resolved_at":null`) || !strings.Contains(string(encoded), `"status_code":0`) || strings.Contains(string(encoded), `"failure_message"`) {
		t.Fatalf("unexpected absent failure context response: %s", encoded)
	}

	resolvedAt := started.Add(2250 * time.Millisecond)
	resolved := incident.Incident{ID: "resolved", MonitorID: "monitor-1", StartedAt: started, ResolvedAt: &resolvedAt, FailureType: "http", StatusCode: 503, FailureMessage: "initial failure"}
	a := newIncidentResponse(resolved, started.Add(time.Minute))
	b := newIncidentResponse(resolved, started.Add(time.Hour))
	if a.IsOpen || a.ResolvedAt == nil || *a.ResolvedAt != resolvedAt.Format(time.RFC3339Nano) || a.DurationMs != 2250 || b.DurationMs != a.DurationMs || a.FailureMessage != "initial failure" {
		t.Fatalf("unexpected fixed resolved duration or context: first=%+v later=%+v", a, b)
	}
	beforeStart := started.Add(-time.Second)
	resolved.ResolvedAt = &beforeStart
	if negative := newIncidentResponse(resolved, started.Add(time.Hour)); negative.DurationMs != 0 {
		t.Fatalf("expected negative resolved duration to clamp to zero, got %+v", negative)
	}
}

func (f *fakeIncidentRepository) GetOpenByMonitor(
	ctx context.Context, monitorID string,
) (*incident.Incident, error) {
	if f.openErr != nil {
		return nil, f.openErr
	}
	for _, value := range f.values {
		if value.ResolvedAt == nil {
			return &value, nil
		}
	}
	return nil, nil
}

func incidentRequest(path string) *http.Request {
	request := httptest.NewRequest(http.MethodGet, path, nil)
	request.SetPathValue("id", "monitor-1")
	return request
}

func TestListIncidents(t *testing.T) {
	started := time.Date(2026, 9, 25, 13, 0, 0, 0, time.UTC)
	resolved := started.Add(90 * time.Second)
	repository := &fakeIncidentRepository{values: []incident.Incident{
		{ID: "incident-2", MonitorID: "monitor-1", StartedAt: started.Add(time.Hour), FailureType: "network", FailureMessage: "connection refused"},
		{ID: "incident-1", MonitorID: "monitor-1", StartedAt: started, ResolvedAt: &resolved, FailureType: "http", StatusCode: 503},
	}}
	handler := NewIncidentHandler(repository, &fakeCheckRepository{})
	request := incidentRequest("/monitors/monitor-1/incidents?limit=1&offset=1")
	recorder := httptest.NewRecorder()
	handler.ListByMonitor(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recorder.Code)
	}
	var response []incidentResponse
	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode incidents: %v", err)
	}
	if len(response) != 1 || response[0].ID != "incident-1" {
		t.Fatalf("expected paginated resolved incident, got %+v", response)
	}
	got := response[0]
	if got.MonitorID != "monitor-1" || got.StartedAt != started.Format(time.RFC3339Nano) || got.ResolvedAt == nil || *got.ResolvedAt != resolved.Format(time.RFC3339Nano) || got.IsOpen || got.DurationMs != 90000 || got.FailureType != "http" || got.StatusCode != 503 {
		t.Fatalf("unexpected resolved incident: %+v", got)
	}
}

func TestCurrentIncidentAndEmptyStates(t *testing.T) {
	started := time.Now().Add(-2 * time.Minute)
	repository := &fakeIncidentRepository{values: []incident.Incident{{
		ID: "incident-1", MonitorID: "monitor-1", StartedAt: started, FailureType: "network",
	}}}
	handler := NewIncidentHandler(repository, &fakeCheckRepository{})
	recorder := httptest.NewRecorder()
	handler.GetCurrent(recorder, incidentRequest("/monitors/monitor-1/incidents/current"))
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recorder.Code)
	}
	var current incidentResponse
	if err := json.NewDecoder(recorder.Body).Decode(&current); err != nil {
		t.Fatalf("failed to decode current incident: %v", err)
	}
	if !current.IsOpen || current.ResolvedAt != nil || current.DurationMs < 120000 {
		t.Fatalf("unexpected open incident: %+v", current)
	}

	repository.values = nil
	recorder = httptest.NewRecorder()
	handler.GetCurrent(recorder, incidentRequest("/monitors/monitor-1/incidents/current"))
	if recorder.Code != http.StatusNoContent || recorder.Body.Len() != 0 {
		t.Fatalf("expected 204 with no body, got %d and %q", recorder.Code, recorder.Body.String())
	}
	repository.values = []incident.Incident{{ID: "resolved", MonitorID: "monitor-1", StartedAt: started, ResolvedAt: &started}}
	recorder = httptest.NewRecorder()
	handler.GetCurrent(recorder, incidentRequest("/monitors/monitor-1/incidents/current"))
	if recorder.Code != http.StatusNoContent || recorder.Body.Len() != 0 {
		t.Fatalf("expected resolved-only history to have no current incident, got %d and %q", recorder.Code, recorder.Body.String())
	}
	repository.values = nil
	recorder = httptest.NewRecorder()
	handler.ListByMonitor(recorder, incidentRequest("/monitors/monitor-1/incidents"))
	if recorder.Code != http.StatusOK || recorder.Body.String() != "[]\n" {
		t.Fatalf("expected empty incident history, got %d and %q", recorder.Code, recorder.Body.String())
	}
}

func TestIncidentEndpointsRequireExistingMonitor(t *testing.T) {
	for _, id := range []string{"00000000-0000-0000-0000-000000000000", "not-a-uuid"} {
		handler := NewIncidentHandler(&fakeIncidentRepository{}, &fakeCheckRepository{missing: true})
		for _, endpoint := range []struct {
			path  string
			serve http.HandlerFunc
		}{
			{path: "/incidents", serve: handler.ListByMonitor},
			{path: "/incidents/current", serve: handler.GetCurrent},
		} {
			request := incidentRequest("/monitors/" + id + endpoint.path)
			request.SetPathValue("id", id)
			recorder := httptest.NewRecorder()
			endpoint.serve(recorder, request)
			assertJSONError(t, recorder, http.StatusNotFound, "monitor not found")
		}
	}
}

func TestListIncidentsRejectsMalformedPagination(t *testing.T) {
	handler := NewIncidentHandler(&fakeIncidentRepository{}, &fakeCheckRepository{})
	for _, query := range []string{"?limit=", "?limit=abc", "?limit=0", "?limit=-1", "?limit=101", "?limit=999999", "?offset=", "?offset=abc", "?offset=-1"} {
		recorder := httptest.NewRecorder()
		handler.ListByMonitor(recorder, incidentRequest("/monitors/monitor-1/incidents"+query))
		assertJSONError(t, recorder, http.StatusBadRequest, "invalid limit or offset")
	}
}

func TestIncidentRepositoryErrors(t *testing.T) {
	repository := &fakeIncidentRepository{listErr: errors.New("database password=secret"), openErr: errors.New("database password=secret")}
	handler := NewIncidentHandler(repository, &fakeCheckRepository{})
	for _, endpoint := range []struct {
		path  string
		serve http.HandlerFunc
	}{
		{path: "/monitors/monitor-1/incidents", serve: handler.ListByMonitor},
		{path: "/monitors/monitor-1/incidents/current", serve: handler.GetCurrent},
	} {
		recorder := httptest.NewRecorder()
		endpoint.serve(recorder, incidentRequest(endpoint.path))
		assertJSONError(t, recorder, http.StatusInternalServerError, "internal server error")
	}
}

func TestIncidentEndpointsHideMonitorRepositoryErrors(t *testing.T) {
	handler := NewIncidentHandler(&fakeIncidentRepository{}, &fakeCheckRepository{existsErr: errors.New("database password=secret")})
	for _, endpoint := range []struct {
		path  string
		serve http.HandlerFunc
	}{
		{path: "/incidents", serve: handler.ListByMonitor},
		{path: "/incidents/current", serve: handler.GetCurrent},
	} {
		recorder := httptest.NewRecorder()
		endpoint.serve(recorder, incidentRequest("/monitors/monitor-1"+endpoint.path))
		assertJSONError(t, recorder, http.StatusInternalServerError, "internal server error")
	}
}

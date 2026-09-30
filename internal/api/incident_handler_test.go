package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gabipuzon/horus/internal/incident"
)

type fakeIncidentRepository struct {
	values  []incident.Incident
	listErr error
	openErr error
}

func (f *fakeIncidentRepository) ListByMonitor(
	ctx context.Context, monitorID string, limit, offset int,
) ([]incident.Incident, error) {
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
	recorder = httptest.NewRecorder()
	handler.ListByMonitor(recorder, incidentRequest("/monitors/monitor-1/incidents"))
	if recorder.Code != http.StatusOK || recorder.Body.String() != "[]\n" {
		t.Fatalf("expected empty incident history, got %d and %q", recorder.Code, recorder.Body.String())
	}
}

func TestIncidentEndpointsRequireExistingMonitor(t *testing.T) {
	handler := NewIncidentHandler(&fakeIncidentRepository{}, &fakeCheckRepository{missing: true})
	for _, endpoint := range []struct {
		path  string
		serve http.HandlerFunc
	}{
		{path: "/monitors/missing/incidents", serve: handler.ListByMonitor},
		{path: "/monitors/missing/incidents/current", serve: handler.GetCurrent},
	} {
		request := incidentRequest(endpoint.path)
		request.SetPathValue("id", "missing")
		recorder := httptest.NewRecorder()
		endpoint.serve(recorder, request)
		if recorder.Code != http.StatusNotFound {
			t.Fatalf("%s: expected 404, got %d", endpoint.path, recorder.Code)
		}
	}
}

func TestListIncidentsRejectsMalformedPagination(t *testing.T) {
	handler := NewIncidentHandler(&fakeIncidentRepository{}, &fakeCheckRepository{})
	for _, query := range []string{"?limit=", "?limit=abc", "?limit=0", "?limit=101", "?offset=", "?offset=abc", "?offset=-1"} {
		recorder := httptest.NewRecorder()
		handler.ListByMonitor(recorder, incidentRequest("/monitors/monitor-1/incidents"+query))
		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("%s: expected 400, got %d", query, recorder.Code)
		}
	}
}

func TestIncidentRepositoryErrors(t *testing.T) {
	repository := &fakeIncidentRepository{listErr: errors.New("database error"), openErr: errors.New("database error")}
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
		if recorder.Code != http.StatusInternalServerError {
			t.Fatalf("%s: expected 500, got %d", endpoint.path, recorder.Code)
		}
	}
}

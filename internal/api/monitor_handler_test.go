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

	"github.com/gabipuzon/horus/internal/monitor"
	"github.com/google/uuid"
)

type fakeMonitorRepository struct {
	created *monitor.Monitor
	err     error
}

func (f *fakeMonitorRepository) Create(
	ctx context.Context,
	m *monitor.Monitor,
) error {
	if f.err != nil {
		return f.err
	}
	f.created = m
	return nil
}

func TestCreateMonitor(t *testing.T) {
	repository := &fakeMonitorRepository{}
	handler := NewMonitorHandler(repository)

	body := `{
		"name": "Example",
		"url": "https://example.com",
		"interval_seconds": 60,
		"timeout_seconds": 5,
		"expected_status": 200
	}`

	request := httptest.NewRequest(
		http.MethodPost,
		"/monitors",
		strings.NewReader(body),
	)

	request.Header.Set("Content-Type", "application/json")

	recorder := httptest.NewRecorder()

	handler.Create(recorder, request)

	if recorder.Code != http.StatusCreated {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusCreated,
			recorder.Code,
		)
	}

	if repository.created == nil {
		t.Fatal("expected monitor to be created")
	}

	var response monitorResponse

	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response.Name != "Example" {
		t.Fatalf(
			"expected name %q, got %q",
			"Example",
			response.Name,
		)
	}

	if response.URL != "https://example.com" {
		t.Fatalf(
			"expected URL %q, got %q",
			"https://example.com",
			response.URL,
		)
	}

	if response.Interval != 60 {
		t.Fatalf(
			"expected interval %d, got %d",
			60,
			response.Interval,
		)
	}

	if response.Timeout != 5 {
		t.Fatalf(
			"expected timeout %d, got %d",
			5,
			response.Timeout,
		)
	}

	if response.ExpectedStatus != 200 {
		t.Fatalf(
			"expected status %d, got %d",
			200,
			response.ExpectedStatus,
		)
	}

	if !response.Enabled {
		t.Fatal("expected monitor to be enabled")
	}
}

func TestCreateMonitorInvalidRequest(t *testing.T) {
	repository := &fakeMonitorRepository{}
	handler := NewMonitorHandler(repository)

	body := `{
		"name": "",
		"url": "https://example.com",
		"interval_seconds": 60,
		"timeout_seconds": 5,
		"expected_status": 200
	}`

	request := httptest.NewRequest(
		http.MethodPost,
		"/monitors",
		strings.NewReader(body),
	)

	request.Header.Set("Content-Type", "application/json")

	recorder := httptest.NewRecorder()

	handler.Create(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusBadRequest,
			recorder.Code,
		)
	}
}

func TestCreateMonitorRejectsInvalidJSON(t *testing.T) {
	valid := `{"name":"Example","url":"https://example.com","interval_seconds":60,"timeout_seconds":5,"expected_status":200}`
	for _, test := range []struct {
		name string
		body string
	}{
		{name: "malformed", body: `{"name":`},
		{name: "unknown field", body: `{"name":"Example","url":"https://example.com","interval_seconds":60,"timeout_seconds":5,"expected_status":200,"typo_field":true}`},
		{name: "second value", body: valid + ` true`},
		{name: "trailing garbage", body: valid + ` trailing`},
	} {
		t.Run(test.name, func(t *testing.T) {
			repository := &fakeMonitorRepository{}
			recorder := httptest.NewRecorder()
			NewMonitorHandler(repository).Create(recorder, httptest.NewRequest(http.MethodPost, "/monitors", strings.NewReader(test.body)))
			if recorder.Code != http.StatusBadRequest || recorder.Body.String() != "invalid request body\n" || repository.created != nil {
				t.Fatalf("unexpected response or create for %s: %d %s", test.name, recorder.Code, recorder.Body.String())
			}
		})
	}
}

func TestCreateMonitorRejectsInvalidValues(t *testing.T) {
	for _, test := range []struct {
		name           string
		monitorName    string
		url            string
		interval       int64
		timeout        int64
		expectedStatus int
	}{
		{name: "empty name", url: "https://example.com", interval: 60, timeout: 5, expectedStatus: 200},
		{name: "blank name", monitorName: "  ", url: "https://example.com", interval: 60, timeout: 5, expectedStatus: 200},
		{name: "empty URL", monitorName: "Example", interval: 60, timeout: 5, expectedStatus: 200},
		{name: "relative URL", monitorName: "Example", url: "not-a-url", interval: 60, timeout: 5, expectedStatus: 200},
		{name: "unsupported scheme", monitorName: "Example", url: "ftp://example.com", interval: 60, timeout: 5, expectedStatus: 200},
		{name: "URL credentials", monitorName: "Example", url: "https://user:password@example.com", interval: 60, timeout: 5, expectedStatus: 200},
		{name: "zero interval", monitorName: "Example", url: "https://example.com", timeout: 5, expectedStatus: 200},
		{name: "negative interval", monitorName: "Example", url: "https://example.com", interval: -1, timeout: 5, expectedStatus: 200},
		{name: "zero timeout", monitorName: "Example", url: "https://example.com", interval: 60, expectedStatus: 200},
		{name: "negative timeout", monitorName: "Example", url: "https://example.com", interval: 60, timeout: -1, expectedStatus: 200},
		{name: "interval exceeds storage", monitorName: "Example", url: "https://example.com", interval: 1 << 31, timeout: 5, expectedStatus: 200},
		{name: "timeout exceeds storage", monitorName: "Example", url: "https://example.com", interval: 60, timeout: 1 << 31, expectedStatus: 200},
		{name: "low status", monitorName: "Example", url: "https://example.com", interval: 60, timeout: 5, expectedStatus: 99},
		{name: "high status", monitorName: "Example", url: "https://example.com", interval: 60, timeout: 5, expectedStatus: 600},
	} {
		t.Run(test.name, func(t *testing.T) {
			body, err := json.Marshal(createMonitorRequest{
				Name: test.monitorName, URL: test.url, Interval: test.interval,
				Timeout: test.timeout, ExpectedStatus: test.expectedStatus,
			})
			if err != nil {
				t.Fatal(err)
			}
			repository := &fakeMonitorRepository{}
			recorder := httptest.NewRecorder()
			NewMonitorHandler(repository).Create(recorder, httptest.NewRequest(http.MethodPost, "/monitors", strings.NewReader(string(body))))
			if recorder.Code != http.StatusBadRequest || repository.created != nil {
				t.Fatalf("unexpected response or create for %s: %d %s", test.name, recorder.Code, recorder.Body.String())
			}
		})
	}
}

func (f *fakeMonitorRepository) List(
	ctx context.Context,
) ([]*monitor.Monitor, error) {
	if f.err != nil {
		return nil, f.err
	}
	if f.created == nil {
		return []*monitor.Monitor{}, nil
	}

	return []*monitor.Monitor{f.created}, nil
}

func TestListMonitors(t *testing.T) {
	repository := &fakeMonitorRepository{}

	createdMonitor, err := monitor.New(
		"Example",
		"https://example.com",
		60*time.Second,
		5*time.Second,
		200,
	)
	if err != nil {
		t.Fatalf("failed to create monitor: %v", err)
	}

	repository.created = createdMonitor

	handler := NewMonitorHandler(repository)

	request := httptest.NewRequest(
		http.MethodGet,
		"/monitors",
		nil,
	)

	recorder := httptest.NewRecorder()

	handler.List(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusOK,
			recorder.Code,
		)
	}

	var response []monitorResponse

	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(response) != 1 {
		t.Fatalf(
			"expected 1 monitor, got %d",
			len(response),
		)
	}

	if response[0].Name != "Example" {
		t.Fatalf(
			"expected name %q, got %q",
			"Example",
			response[0].Name,
		)
	}
}

func (f *fakeMonitorRepository) GetByID(
	ctx context.Context,
	id string,
) (*monitor.Monitor, error) {
	if f.err != nil {
		return nil, f.err
	}
	if f.created == nil {
		return nil, monitor.ErrNotFound
	}

	if f.created.ID != id {
		return nil, monitor.ErrNotFound
	}

	return f.created, nil
}

func TestGetMonitor(t *testing.T) {
	repository := &fakeMonitorRepository{}

	createdMonitor, err := monitor.New(
		"Example",
		"https://example.com",
		60*time.Second,
		5*time.Second,
		200,
	)
	if err != nil {
		t.Fatalf("failed to create monitor: %v", err)
	}

	repository.created = createdMonitor

	handler := NewMonitorHandler(repository)

	request := httptest.NewRequest(
		http.MethodGet,
		"/monitors/"+createdMonitor.ID,
		nil,
	)

	request.SetPathValue("id", createdMonitor.ID)

	recorder := httptest.NewRecorder()

	handler.GetByID(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusOK,
			recorder.Code,
		)
	}

	var response monitorResponse

	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response.ID != createdMonitor.ID {
		t.Fatalf(
			"expected ID %q, got %q",
			createdMonitor.ID,
			response.ID,
		)
	}

	if response.Name != "Example" {
		t.Fatalf(
			"expected name %q, got %q",
			"Example",
			response.Name,
		)
	}
}

func TestGetMonitorNotFound(t *testing.T) {
	repository := &fakeMonitorRepository{}

	handler := NewMonitorHandler(repository)
	id := uuid.NewString()

	request := httptest.NewRequest(
		http.MethodGet,
		"/monitors/"+id,
		nil,
	)

	request.SetPathValue("id", id)

	recorder := httptest.NewRecorder()

	handler.GetByID(recorder, request)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusNotFound,
			recorder.Code,
		)
	}
}

func (f *fakeMonitorRepository) Delete(
	ctx context.Context,
	id string,
) error {
	if f.err != nil {
		return f.err
	}
	if f.created == nil {
		return monitor.ErrNotFound
	}

	if f.created.ID != id {
		return monitor.ErrNotFound
	}

	f.created = nil

	return nil
}

func TestDeleteMonitor(t *testing.T) {
	repository := &fakeMonitorRepository{}

	createdMonitor, err := monitor.New(
		"Example",
		"https://example.com",
		60*time.Second,
		5*time.Second,
		200,
	)
	if err != nil {
		t.Fatalf("failed to create monitor: %v", err)
	}

	repository.created = createdMonitor

	handler := NewMonitorHandler(repository)

	request := httptest.NewRequest(
		http.MethodDelete,
		"/monitors/"+createdMonitor.ID,
		nil,
	)

	request.SetPathValue("id", createdMonitor.ID)

	recorder := httptest.NewRecorder()

	handler.Delete(recorder, request)

	if recorder.Code != http.StatusNoContent {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusNoContent,
			recorder.Code,
		)
	}
	if recorder.Body.Len() != 0 {
		t.Fatalf("expected empty delete response, got %q", recorder.Body.String())
	}

	if repository.created != nil {
		t.Fatal("expected monitor to be deleted")
	}
}

func TestDeleteMonitorNotFound(t *testing.T) {
	repository := &fakeMonitorRepository{}

	handler := NewMonitorHandler(repository)

	request := httptest.NewRequest(
		http.MethodDelete,
		"/monitors/missing",
		nil,
	)

	request.SetPathValue("id", "missing")

	recorder := httptest.NewRecorder()

	handler.Delete(recorder, request)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusNotFound,
			recorder.Code,
		)
	}
}

func (f *fakeMonitorRepository) SetEnabled(
	ctx context.Context,
	id string,
	enabled bool,
) error {
	if f.err != nil {
		return f.err
	}
	if f.created == nil {
		return monitor.ErrNotFound
	}

	if f.created.ID != id {
		return monitor.ErrNotFound
	}

	f.created.Enabled = enabled

	return nil
}

func TestEnableMonitor(t *testing.T) {
	repository := &fakeMonitorRepository{}

	createdMonitor, err := monitor.New(
		"Example",
		"https://example.com",
		60*time.Second,
		5*time.Second,
		200,
	)
	if err != nil {
		t.Fatalf("failed to create monitor: %v", err)
	}

	createdMonitor.Enabled = false
	repository.created = createdMonitor

	handler := NewMonitorHandler(repository)

	request := httptest.NewRequest(
		http.MethodPatch,
		"/monitors/"+createdMonitor.ID+"/enable",
		nil,
	)
	request.SetPathValue("id", createdMonitor.ID)

	recorder := httptest.NewRecorder()

	handler.Enable(recorder, request)

	if recorder.Code != http.StatusNoContent {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusNoContent,
			recorder.Code,
		)
	}
	if recorder.Body.Len() != 0 {
		t.Fatalf("expected empty enable response, got %q", recorder.Body.String())
	}

	if !repository.created.Enabled {
		t.Fatal("expected monitor to be enabled")
	}
}

func TestDisableMonitor(t *testing.T) {
	repository := &fakeMonitorRepository{}

	createdMonitor, err := monitor.New(
		"Example",
		"https://example.com",
		60*time.Second,
		5*time.Second,
		200,
	)
	if err != nil {
		t.Fatalf("failed to create monitor: %v", err)
	}

	repository.created = createdMonitor

	handler := NewMonitorHandler(repository)

	request := httptest.NewRequest(
		http.MethodPatch,
		"/monitors/"+createdMonitor.ID+"/disable",
		nil,
	)
	request.SetPathValue("id", createdMonitor.ID)

	recorder := httptest.NewRecorder()

	handler.Disable(recorder, request)

	if recorder.Code != http.StatusNoContent {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusNoContent,
			recorder.Code,
		)
	}
	if recorder.Body.Len() != 0 {
		t.Fatalf("expected empty disable response, got %q", recorder.Body.String())
	}

	if repository.created.Enabled {
		t.Fatal("expected monitor to be disabled")
	}
}

func TestMonitorMutationNotFound(t *testing.T) {
	for _, test := range []struct {
		name    string
		verb    string
		suffix  string
		handler func(*MonitorHandler, http.ResponseWriter, *http.Request)
	}{
		{name: "delete", verb: http.MethodDelete, handler: (*MonitorHandler).Delete},
		{name: "enable", verb: http.MethodPatch, suffix: "/enable", handler: (*MonitorHandler).Enable},
		{name: "disable", verb: http.MethodPatch, suffix: "/disable", handler: (*MonitorHandler).Disable},
	} {
		t.Run(test.name, func(t *testing.T) {
			repository := &fakeMonitorRepository{}
			id := uuid.NewString()
			request := httptest.NewRequest(test.verb, "/monitors/"+id+test.suffix, nil)
			request.SetPathValue("id", id)
			recorder := httptest.NewRecorder()
			test.handler(NewMonitorHandler(repository), recorder, request)
			if recorder.Code != http.StatusNotFound || recorder.Body.String() != "monitor not found\n" {
				t.Fatalf("expected safe 404, got %d %s", recorder.Code, recorder.Body.String())
			}
		})
	}
}

func TestMalformedMonitorIDReturnsNotFound(t *testing.T) {
	for _, test := range []struct {
		name    string
		verb    string
		suffix  string
		handler func(*MonitorHandler, http.ResponseWriter, *http.Request)
	}{
		{name: "get", verb: http.MethodGet, handler: (*MonitorHandler).GetByID},
		{name: "delete", verb: http.MethodDelete, handler: (*MonitorHandler).Delete},
		{name: "enable", verb: http.MethodPatch, suffix: "/enable", handler: (*MonitorHandler).Enable},
		{name: "disable", verb: http.MethodPatch, suffix: "/disable", handler: (*MonitorHandler).Disable},
	} {
		t.Run(test.name, func(t *testing.T) {
			repository := &fakeMonitorRepository{}
			request := httptest.NewRequest(test.verb, "/monitors/not-a-uuid"+test.suffix, nil)
			request.SetPathValue("id", "not-a-uuid")
			recorder := httptest.NewRecorder()
			test.handler(NewMonitorHandler(repository), recorder, request)
			if recorder.Code != http.StatusNotFound {
				t.Fatalf("expected 404, got %d %s", recorder.Code, recorder.Body.String())
			}
		})
	}
}

func TestMonitorRepositoryErrorsAreSafe(t *testing.T) {
	for _, test := range []struct {
		name    string
		verb    string
		suffix  string
		handler func(*MonitorHandler, http.ResponseWriter, *http.Request)
	}{
		{name: "get", verb: http.MethodGet, handler: (*MonitorHandler).GetByID},
		{name: "delete", verb: http.MethodDelete, handler: (*MonitorHandler).Delete},
		{name: "enable", verb: http.MethodPatch, suffix: "/enable", handler: (*MonitorHandler).Enable},
		{name: "disable", verb: http.MethodPatch, suffix: "/disable", handler: (*MonitorHandler).Disable},
	} {
		t.Run(test.name, func(t *testing.T) {
			repository := &fakeMonitorRepository{err: errors.New("database password=secret")}
			id := uuid.NewString()
			request := httptest.NewRequest(test.verb, "/monitors/"+id+test.suffix, nil)
			request.SetPathValue("id", id)
			recorder := httptest.NewRecorder()
			test.handler(NewMonitorHandler(repository), recorder, request)
			if recorder.Code != http.StatusInternalServerError || strings.Contains(recorder.Body.String(), "secret") {
				t.Fatalf("expected safe 500, got %d %s", recorder.Code, recorder.Body.String())
			}
		})
	}
}

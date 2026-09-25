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
)

type fakeMonitorRepository struct {
	created *monitor.Monitor
}

func (f *fakeMonitorRepository) Create(
	ctx context.Context,
	m *monitor.Monitor,
) error {
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

func (f *fakeMonitorRepository) List(
	ctx context.Context,
) ([]*monitor.Monitor, error) {
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
	if f.created == nil {
		return nil, errors.New("monitor not found")
	}

	if f.created.ID != id {
		return nil, errors.New("monitor not found")
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

	request := httptest.NewRequest(
		http.MethodGet,
		"/monitors/missing",
		nil,
	)

	request.SetPathValue("id", "missing")

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
	if f.created == nil {
		return errors.New("monitor not found")
	}

	if f.created.ID != id {
		return errors.New("monitor not found")
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

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusInternalServerError,
			recorder.Code,
		)
	}
}

package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCreateMonitor(t *testing.T) {
	handler := NewMonitorHandler()

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
	handler := NewMonitorHandler()

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

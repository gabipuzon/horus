package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gabipuzon/horus/internal/monitor"
)

type fakeCheckRepository struct {
	checks  []monitor.Check
	summary monitor.CheckSummary
}

func (f *fakeCheckRepository) ListByMonitor(
	ctx context.Context,
	monitorID string,
	limit int,
	offset int,
) ([]monitor.Check, error) {
	if offset >= len(f.checks) {
		return []monitor.Check{}, nil
	}

	end := offset + limit

	if end > len(f.checks) {
		end = len(f.checks)
	}

	return f.checks[offset:end], nil
}

func TestListChecks(t *testing.T) {
	checkedAt := time.Date(
		2026,
		9,
		25,
		13,
		0,
		0,
		0,
		time.UTC,
	)

	repository := &fakeCheckRepository{
		checks: []monitor.Check{
			{
				ID:          "check-1",
				MonitorID:   "monitor-1",
				StatusCode:  200,
				Latency:     150 * time.Millisecond,
				Success:     true,
				FailureType: monitor.FailureNone,
				CheckedAt:   checkedAt,
			},
		},
	}

	handler := NewCheckHandler(repository, repository)

	request := httptest.NewRequest(
		http.MethodGet,
		"/monitors/monitor-1/checks",
		nil,
	)

	request.SetPathValue("id", "monitor-1")

	recorder := httptest.NewRecorder()

	handler.ListByMonitor(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusOK,
			recorder.Code,
		)
	}

	var response []checkResponse

	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(response) != 1 {
		t.Fatalf(
			"expected 1 check, got %d",
			len(response),
		)
	}

	if response[0].ID != "check-1" {
		t.Fatalf(
			"expected ID %q, got %q",
			"check-1",
			response[0].ID,
		)
	}

	if response[0].MonitorID != "monitor-1" {
		t.Fatalf(
			"expected monitor ID %q, got %q",
			"monitor-1",
			response[0].MonitorID,
		)
	}

	if response[0].StatusCode != 200 {
		t.Fatalf(
			"expected status code %d, got %d",
			200,
			response[0].StatusCode,
		)
	}

	if response[0].LatencyMs != 150 {
		t.Fatalf(
			"expected latency %dms, got %dms",
			150,
			response[0].LatencyMs,
		)
	}

	if !response[0].Success {
		t.Fatal("expected check to be successful")
	}

	if response[0].FailureType != "" {
		t.Fatalf(
			"expected empty failure type, got %q",
			response[0].FailureType,
		)
	}
}

func TestListChecksWithLimit(t *testing.T) {
	repository := &fakeCheckRepository{
		checks: []monitor.Check{
			{ID: "check-1", MonitorID: "monitor-1"},
			{ID: "check-2", MonitorID: "monitor-1"},
			{ID: "check-3", MonitorID: "monitor-1"},
		},
	}

	handler := NewCheckHandler(repository, repository)

	request := httptest.NewRequest(
		http.MethodGet,
		"/monitors/monitor-1/checks?limit=2",
		nil,
	)

	request.SetPathValue("id", "monitor-1")

	recorder := httptest.NewRecorder()

	handler.ListByMonitor(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusOK,
			recorder.Code,
		)
	}

	var response []checkResponse

	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(response) != 2 {
		t.Fatalf("expected 2 checks, got %d", len(response))
	}
}

func TestListChecksWithOffset(t *testing.T) {
	repository := &fakeCheckRepository{
		checks: []monitor.Check{
			{ID: "check-1", MonitorID: "monitor-1"},
			{ID: "check-2", MonitorID: "monitor-1"},
			{ID: "check-3", MonitorID: "monitor-1"},
		},
	}

	handler := NewCheckHandler(repository, repository)

	request := httptest.NewRequest(
		http.MethodGet,
		"/monitors/monitor-1/checks?offset=1",
		nil,
	)

	request.SetPathValue("id", "monitor-1")

	recorder := httptest.NewRecorder()

	handler.ListByMonitor(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusOK,
			recorder.Code,
		)
	}

	var response []checkResponse

	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(response) != 2 {
		t.Fatalf("expected 2 checks, got %d", len(response))
	}

	if response[0].ID != "check-2" {
		t.Fatalf(
			"expected first check to be check-2, got %s",
			response[0].ID,
		)
	}
}

func TestListChecksWithInvalidLimit(t *testing.T) {
	repository := &fakeCheckRepository{}

	handler := NewCheckHandler(repository, repository)

	request := httptest.NewRequest(
		http.MethodGet,
		"/monitors/monitor-1/checks?limit=101",
		nil,
	)

	request.SetPathValue("id", "monitor-1")

	recorder := httptest.NewRecorder()

	handler.ListByMonitor(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusBadRequest,
			recorder.Code,
		)
	}
}

func TestListChecksWithNegativeOffset(t *testing.T) {
	repository := &fakeCheckRepository{}

	handler := NewCheckHandler(repository, repository)

	request := httptest.NewRequest(
		http.MethodGet,
		"/monitors/monitor-1/checks?offset=-1",
		nil,
	)

	request.SetPathValue("id", "monitor-1")

	recorder := httptest.NewRecorder()

	handler.ListByMonitor(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusBadRequest,
			recorder.Code,
		)
	}
}

func (f *fakeCheckRepository) GetSummary(
	ctx context.Context,
	monitorID string,
) (monitor.CheckSummary, error) {
	return f.summary, nil
}

func TestGetSummary(t *testing.T) {
	repository := &fakeCheckRepository{
		summary: monitor.CheckSummary{
			TotalChecks:      100,
			SuccessfulChecks: 95,
			FailedChecks:     5,
			AverageLatency:   150 * time.Millisecond,
			LatestStatus:     500,
		},
	}

	handler := NewCheckHandler(repository, repository)

	request := httptest.NewRequest(
		http.MethodGet,
		"/monitors/monitor-1/summary",
		nil,
	)

	request.SetPathValue("id", "monitor-1")

	recorder := httptest.NewRecorder()

	handler.GetSummary(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusOK,
			recorder.Code,
		)
	}

	var response checkSummaryResponse

	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response.TotalChecks != 100 {
		t.Fatalf(
			"expected 100 total checks, got %d",
			response.TotalChecks,
		)
	}

	if response.SuccessfulChecks != 95 {
		t.Fatalf(
			"expected 95 successful checks, got %d",
			response.SuccessfulChecks,
		)
	}

	if response.FailedChecks != 5 {
		t.Fatalf(
			"expected 5 failed checks, got %d",
			response.FailedChecks,
		)
	}

	if response.AverageLatencyMs != 150 {
		t.Fatalf(
			"expected average latency 150ms, got %d",
			response.AverageLatencyMs,
		)
	}

	if response.LatestStatus != 500 {
		t.Fatalf(
			"expected latest status 500, got %d",
			response.LatestStatus,
		)
	}
}

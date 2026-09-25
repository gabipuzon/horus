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
	checks []monitor.Check
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

	handler := NewCheckHandler(repository)

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

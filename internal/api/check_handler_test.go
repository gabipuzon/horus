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

	"github.com/gabipuzon/horus/internal/check"
)

type fakeCheckRepository struct {
	checks     []check.Record
	summary    check.Summary
	missing    bool
	existsErr  error
	listErr    error
	summaryErr error
	listCalls  int
	gotLimit   int
	gotOffset  int
}

func (f *fakeCheckRepository) Exists(ctx context.Context, id string) (bool, error) {
	return !f.missing, f.existsErr
}

func (f *fakeCheckRepository) ListByMonitor(
	ctx context.Context,
	monitorID string,
	limit int,
	offset int,
) ([]check.Record, error) {
	f.listCalls++
	f.gotLimit, f.gotOffset = limit, offset
	if f.listErr != nil {
		return nil, f.listErr
	}
	if offset >= len(f.checks) {
		return []check.Record{}, nil
	}

	end := offset + limit

	if end > len(f.checks) {
		end = len(f.checks)
	}

	return f.checks[offset:end], nil
}

func TestListChecksEmptyHistory(t *testing.T) {
	repository := &fakeCheckRepository{}
	request := httptest.NewRequest(http.MethodGet, "/monitors/monitor-1/checks", nil)
	request.SetPathValue("id", "monitor-1")
	recorder := httptest.NewRecorder()
	NewCheckHandler(repository, repository, repository).ListByMonitor(recorder, request)
	if recorder.Code != http.StatusOK || recorder.Body.String() != "[]\n" {
		t.Fatalf("expected 200 empty array, got %d %q", recorder.Code, recorder.Body.String())
	}
}

func TestListChecksPagination(t *testing.T) {
	for _, test := range []struct {
		name       string
		query      string
		wantStatus int
		wantLimit  int
		wantOffset int
	}{
		{name: "defaults", wantStatus: http.StatusOK, wantLimit: 50},
		{name: "custom", query: "?limit=2&offset=3", wantStatus: http.StatusOK, wantLimit: 2, wantOffset: 3},
		{name: "maximum", query: "?limit=100", wantStatus: http.StatusOK, wantLimit: 100},
		{name: "empty limit", query: "?limit=", wantStatus: http.StatusBadRequest},
		{name: "non-numeric limit", query: "?limit=abc", wantStatus: http.StatusBadRequest},
		{name: "negative limit", query: "?limit=-1", wantStatus: http.StatusBadRequest},
		{name: "zero limit", query: "?limit=0", wantStatus: http.StatusBadRequest},
		{name: "oversized limit", query: "?limit=999999", wantStatus: http.StatusBadRequest},
		{name: "empty offset", query: "?offset=", wantStatus: http.StatusBadRequest},
		{name: "non-numeric offset", query: "?offset=abc", wantStatus: http.StatusBadRequest},
		{name: "negative offset", query: "?offset=-1", wantStatus: http.StatusBadRequest},
	} {
		t.Run(test.name, func(t *testing.T) {
			repository := &fakeCheckRepository{}
			request := httptest.NewRequest(http.MethodGet, "/monitors/monitor-1/checks"+test.query, nil)
			request.SetPathValue("id", "monitor-1")
			recorder := httptest.NewRecorder()
			NewCheckHandler(repository, repository, repository).ListByMonitor(recorder, request)
			if recorder.Code != test.wantStatus {
				t.Fatalf("expected %d, got %d: %s", test.wantStatus, recorder.Code, recorder.Body.String())
			}
			if test.wantStatus == http.StatusOK {
				if repository.listCalls != 1 || repository.gotLimit != test.wantLimit || repository.gotOffset != test.wantOffset {
					t.Fatalf("expected one list call with limit %d offset %d; got calls %d limit %d offset %d", test.wantLimit, test.wantOffset, repository.listCalls, repository.gotLimit, repository.gotOffset)
				}
			} else if repository.listCalls != 0 {
				t.Fatal("invalid pagination should not query checks")
			}
		})
	}
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
		checks: []check.Record{
			{
				ID:          "check-1",
				MonitorID:   "monitor-1",
				StatusCode:  200,
				Latency:     150 * time.Millisecond,
				Success:     true,
				FailureType: check.FailureNone,
				CheckedAt:   checkedAt,
			},
		},
	}

	handler := NewCheckHandler(repository, repository, repository)

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
	if response[0].CheckedAt != checkedAt.Format(time.RFC3339) {
		t.Fatalf("expected RFC3339 check time, got %q", response[0].CheckedAt)
	}
}

func TestListChecksWithLimit(t *testing.T) {
	repository := &fakeCheckRepository{
		checks: []check.Record{
			{ID: "check-1", MonitorID: "monitor-1"},
			{ID: "check-2", MonitorID: "monitor-1"},
			{ID: "check-3", MonitorID: "monitor-1"},
		},
	}

	handler := NewCheckHandler(repository, repository, repository)

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
		checks: []check.Record{
			{ID: "check-1", MonitorID: "monitor-1"},
			{ID: "check-2", MonitorID: "monitor-1"},
			{ID: "check-3", MonitorID: "monitor-1"},
		},
	}

	handler := NewCheckHandler(repository, repository, repository)

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

	handler := NewCheckHandler(repository, repository, repository)

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

	handler := NewCheckHandler(repository, repository, repository)

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
) (check.Summary, error) {
	return f.summary, f.summaryErr
}

func TestGetSummary(t *testing.T) {
	repository := &fakeCheckRepository{
		summary: check.Summary{
			TotalChecks:      100,
			SuccessfulChecks: 95,
			FailedChecks:     5,
			AverageLatency:   150 * time.Millisecond,
			LatestStatus:     500,
		},
	}

	handler := NewCheckHandler(repository, repository, repository)

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
	if response.UptimePercentage == nil || *response.UptimePercentage != 95 {
		t.Fatalf("expected 95%% check uptime, got %v", response.UptimePercentage)
	}
}

func TestGetSummaryWithoutChecks(t *testing.T) {
	repository := &fakeCheckRepository{}
	handler := NewCheckHandler(repository, repository, repository)
	request := httptest.NewRequest(http.MethodGet, "/monitors/monitor-1/summary", nil)
	request.SetPathValue("id", "monitor-1")
	recorder := httptest.NewRecorder()
	handler.GetSummary(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recorder.Code)
	}
	body := recorder.Body.String()
	var response checkSummaryResponse
	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if response != (checkSummaryResponse{}) {
		t.Fatalf("expected zero counts, latency, status and nil uptime, got %+v", response)
	}
	if !strings.Contains(body, `"uptime_percentage":null`) {
		t.Fatalf("expected explicit null uptime, got %q", body)
	}
}

func TestCheckEndpointsRequireExistingMonitor(t *testing.T) {
	for _, test := range []struct {
		name      string
		missing   bool
		existsErr error
		want      int
	}{
		{name: "missing", missing: true, want: http.StatusNotFound},
		{name: "malformed UUID", missing: true, want: http.StatusNotFound},
		{name: "repository error", existsErr: errors.New("database unavailable"), want: http.StatusInternalServerError},
	} {
		t.Run(test.name, func(t *testing.T) {
			repository := &fakeCheckRepository{missing: test.missing, existsErr: test.existsErr}
			handler := NewCheckHandler(repository, repository, repository)
			id := "monitor-1"
			if test.name == "malformed UUID" {
				id = "not-a-uuid"
			}
			for _, endpoint := range []struct {
				path  string
				serve http.HandlerFunc
			}{
				{path: "/checks", serve: handler.ListByMonitor},
				{path: "/summary", serve: handler.GetSummary},
			} {
				request := httptest.NewRequest(http.MethodGet, "/monitors/"+id+endpoint.path, nil)
				request.SetPathValue("id", id)
				recorder := httptest.NewRecorder()
				endpoint.serve(recorder, request)
				if recorder.Code != test.want {
					t.Fatalf("%s: expected %d, got %d", endpoint.path, test.want, recorder.Code)
				}
			}
		})
	}
}

func TestCheckEndpointsHideRepositoryErrors(t *testing.T) {
	repository := &fakeCheckRepository{
		listErr:    errors.New("database password=secret"),
		summaryErr: errors.New("database password=secret"),
	}
	handler := NewCheckHandler(repository, repository, repository)
	for _, endpoint := range []struct {
		path  string
		serve http.HandlerFunc
	}{
		{path: "/checks", serve: handler.ListByMonitor},
		{path: "/summary", serve: handler.GetSummary},
	} {
		request := httptest.NewRequest(http.MethodGet, "/monitors/monitor-1"+endpoint.path, nil)
		request.SetPathValue("id", "monitor-1")
		recorder := httptest.NewRecorder()
		endpoint.serve(recorder, request)
		if recorder.Code != http.StatusInternalServerError || strings.Contains(recorder.Body.String(), "secret") {
			t.Fatalf("%s: expected safe 500, got %d %q", endpoint.path, recorder.Code, recorder.Body.String())
		}
	}
}

package api

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/gabipuzon/horus/internal/check"
)

type checkRepository interface {
	ListByMonitor(
		ctx context.Context,
		monitorID string,
		limit int,
		offset int,
	) ([]check.Record, error)
}

type checkSummaryRepository interface {
	GetSummary(
		ctx context.Context,
		monitorID string,
	) (check.Summary, error)
}

type monitorExistenceRepository interface {
	Exists(ctx context.Context, id string) (bool, error)
}

func requireMonitor(w http.ResponseWriter, r *http.Request, repository monitorExistenceRepository) bool {
	exists, err := repository.Exists(r.Context(), r.PathValue("id"))
	if err != nil {
		http.Error(w, "failed to find monitor", http.StatusInternalServerError)
		return false
	}
	if !exists {
		http.Error(w, "monitor not found", http.StatusNotFound)
		return false
	}
	return true
}

type CheckHandler struct {
	repository        checkRepository
	summaryRepository checkSummaryRepository
	monitors          monitorExistenceRepository
}

type checkResponse struct {
	ID          string `json:"id"`
	MonitorID   string `json:"monitor_id"`
	StatusCode  int    `json:"status_code"`
	LatencyMs   int64  `json:"latency_ms"`
	Success     bool   `json:"success"`
	FailureType string `json:"failure_type"`
	Error       string `json:"error,omitempty"`
	CheckedAt   string `json:"checked_at"`
}

type checkSummaryResponse struct {
	TotalChecks      int      `json:"total_checks"`
	SuccessfulChecks int      `json:"successful_checks"`
	FailedChecks     int      `json:"failed_checks"`
	AverageLatencyMs int64    `json:"average_latency_ms"`
	LatestStatus     int      `json:"latest_status"`
	UptimePercentage *float64 `json:"uptime_percentage"`
}

func NewCheckHandler(
	repository checkRepository,
	summaryRepository checkSummaryRepository,
	monitors monitorExistenceRepository,
) *CheckHandler {
	return &CheckHandler{
		repository:        repository,
		summaryRepository: summaryRepository,
		monitors:          monitors,
	}
}

func newCheckResponse(check check.Record) checkResponse {
	return checkResponse{
		ID:          check.ID,
		MonitorID:   check.MonitorID,
		StatusCode:  check.StatusCode,
		LatencyMs:   check.Latency.Milliseconds(),
		Success:     check.Success,
		FailureType: string(check.FailureType),
		Error:       check.Error,
		CheckedAt:   check.CheckedAt.Format(time.RFC3339),
	}
}

func (h *CheckHandler) ListByMonitor(
	w http.ResponseWriter,
	r *http.Request,
) {
	monitorID := r.PathValue("id")

	limit := 50
	offset := 0

	if value := r.URL.Query().Get("limit"); value != "" {
		if parsed, err := strconv.Atoi(value); err == nil {
			limit = parsed
		}
	}

	if value := r.URL.Query().Get("offset"); value != "" {
		if parsed, err := strconv.Atoi(value); err == nil {
			offset = parsed
		}
	}

	if limit < 1 || limit > 100 {
		http.Error(
			w,
			"limit must be between 1 and 100",
			http.StatusBadRequest,
		)
		return
	}

	if offset < 0 {
		http.Error(
			w,
			"offset must not be negative",
			http.StatusBadRequest,
		)
		return
	}
	if !requireMonitor(w, r, h.monitors) {
		return
	}

	checks, err := h.repository.ListByMonitor(
		r.Context(),
		monitorID,
		limit,
		offset,
	)
	if err != nil {
		http.Error(
			w,
			"failed to list checks",
			http.StatusInternalServerError,
		)
		return
	}

	response := make([]checkResponse, 0, len(checks))

	for _, check := range checks {
		response = append(response, newCheckResponse(check))
	}

	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(
			w,
			"failed to encode response",
			http.StatusInternalServerError,
		)
		return
	}
}

func (h *CheckHandler) GetSummary(
	w http.ResponseWriter,
	r *http.Request,
) {
	monitorID := r.PathValue("id")
	if !requireMonitor(w, r, h.monitors) {
		return
	}

	summary, err := h.summaryRepository.GetSummary(
		r.Context(),
		monitorID,
	)
	if err != nil {
		http.Error(
			w,
			"failed to get check summary",
			http.StatusInternalServerError,
		)
		return
	}

	response := checkSummaryResponse{
		TotalChecks:      summary.TotalChecks,
		SuccessfulChecks: summary.SuccessfulChecks,
		FailedChecks:     summary.FailedChecks,
		AverageLatencyMs: summary.AverageLatency.Milliseconds(),
		LatestStatus:     summary.LatestStatus,
		UptimePercentage: summary.UptimePercentage(),
	}

	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(
			w,
			"failed to encode response",
			http.StatusInternalServerError,
		)
		return
	}
}

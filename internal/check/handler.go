package check

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/gabipuzon/horus/internal/httpresponse"
)

type checkRepository interface {
	ListByMonitor(
		ctx context.Context,
		monitorID string,
		limit int,
		offset int,
	) ([]Record, error)
}

type checkSummaryRepository interface {
	GetSummary(
		ctx context.Context,
		monitorID string,
	) (Summary, error)
}

type CheckHandler struct {
	repository        checkRepository
	summaryRepository checkSummaryRepository
	monitors          httpresponse.MonitorExistenceRepository
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
	monitors httpresponse.MonitorExistenceRepository,
) *CheckHandler {
	return &CheckHandler{
		repository:        repository,
		summaryRepository: summaryRepository,
		monitors:          monitors,
	}
}

func newCheckResponse(check Record) checkResponse {
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
	query := r.URL.Query()

	if query.Has("limit") {
		parsed, err := strconv.Atoi(query.Get("limit"))
		if err != nil || parsed < 1 || parsed > 100 {
			httpresponse.WriteError(w, http.StatusBadRequest, "limit must be between 1 and 100")
			return
		}
		limit = parsed
	}

	if query.Has("offset") {
		parsed, err := strconv.Atoi(query.Get("offset"))
		if err != nil || parsed < 0 {
			httpresponse.WriteError(w, http.StatusBadRequest, "offset must not be negative")
			return
		}
		offset = parsed
	}
	if !httpresponse.RequireMonitor(w, r, h.monitors) {
		return
	}

	checks, err := h.repository.ListByMonitor(
		r.Context(),
		monitorID,
		limit,
		offset,
	)
	if err != nil {
		httpresponse.WriteInternalError(w, "list checks", err)
		return
	}

	response := make([]checkResponse, 0, len(checks))

	for _, check := range checks {
		response = append(response, newCheckResponse(check))
	}

	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(response); err != nil {
		httpresponse.LogResponseWriteError("list checks", err)
	}
}

func (h *CheckHandler) GetSummary(
	w http.ResponseWriter,
	r *http.Request,
) {
	monitorID := r.PathValue("id")
	if !httpresponse.RequireMonitor(w, r, h.monitors) {
		return
	}

	summary, err := h.summaryRepository.GetSummary(
		r.Context(),
		monitorID,
	)
	if err != nil {
		httpresponse.WriteInternalError(w, "get check summary", err)
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
		httpresponse.LogResponseWriteError("get check summary", err)
	}
}

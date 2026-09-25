package api

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/gabipuzon/horus/internal/monitor"
)

type CheckRepository interface {
	ListByMonitor(
		ctx context.Context,
		monitorID string,
		limit int,
		offset int,
	) ([]monitor.Check, error)
}

type CheckHandler struct {
	repository CheckRepository
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

func NewCheckHandler(repository CheckRepository) *CheckHandler {
	return &CheckHandler{
		repository: repository,
	}
}

func newCheckResponse(check monitor.Check) checkResponse {
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

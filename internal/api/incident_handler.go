package api

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/gabipuzon/horus/internal/incident"
)

type incidentRepository interface {
	ListByMonitor(ctx context.Context, monitorID string, limit, offset int) ([]incident.Incident, error)
	GetOpenByMonitor(ctx context.Context, monitorID string) (*incident.Incident, error)
}

type IncidentHandler struct {
	repository incidentRepository
	monitors   monitorExistenceRepository
}

func NewIncidentHandler(repository incidentRepository, monitors monitorExistenceRepository) *IncidentHandler {
	return &IncidentHandler{repository: repository, monitors: monitors}
}

type incidentResponse struct {
	ID             string  `json:"id"`
	MonitorID      string  `json:"monitor_id"`
	StartedAt      string  `json:"started_at"`
	ResolvedAt     *string `json:"resolved_at"`
	IsOpen         bool    `json:"is_open"`
	DurationMs     int64   `json:"duration_ms"`
	FailureType    string  `json:"failure_type"`
	StatusCode     int     `json:"status_code"`
	FailureMessage string  `json:"failure_message,omitempty"`
}

func newIncidentResponse(value incident.Incident, now time.Time) incidentResponse {
	end := now
	var resolvedAt *string
	if value.ResolvedAt != nil {
		end = *value.ResolvedAt
		formatted := end.UTC().Format(time.RFC3339Nano)
		resolvedAt = &formatted
	}
	duration := end.Sub(value.StartedAt)
	if duration < 0 {
		duration = 0
	}
	return incidentResponse{
		ID:             value.ID,
		MonitorID:      value.MonitorID,
		StartedAt:      value.StartedAt.UTC().Format(time.RFC3339Nano),
		ResolvedAt:     resolvedAt,
		IsOpen:         value.ResolvedAt == nil,
		DurationMs:     duration.Milliseconds(),
		FailureType:    value.FailureType,
		StatusCode:     value.StatusCode,
		FailureMessage: value.FailureMessage,
	}
}

func incidentPagination(r *http.Request) (int, int, bool) {
	limit, offset := 50, 0
	query := r.URL.Query()
	if query.Has("limit") {
		parsed, err := strconv.Atoi(query.Get("limit"))
		if err != nil || parsed < 1 || parsed > 100 {
			return 0, 0, false
		}
		limit = parsed
	}
	if query.Has("offset") {
		parsed, err := strconv.Atoi(query.Get("offset"))
		if err != nil || parsed < 0 {
			return 0, 0, false
		}
		offset = parsed
	}
	return limit, offset, true
}

func (h *IncidentHandler) ListByMonitor(w http.ResponseWriter, r *http.Request) {
	limit, offset, valid := incidentPagination(r)
	if !valid {
		writeError(w, http.StatusBadRequest, "invalid limit or offset")
		return
	}
	if !requireMonitor(w, r, h.monitors) {
		return
	}

	values, err := h.repository.ListByMonitor(r.Context(), r.PathValue("id"), limit, offset)
	if err != nil {
		writeInternalError(w, "list incidents", err)
		return
	}
	response := make([]incidentResponse, 0, len(values))
	now := time.Now()
	for _, value := range values {
		response = append(response, newIncidentResponse(value, now))
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		logResponseWriteError("list incidents", err)
	}
}

func (h *IncidentHandler) GetCurrent(w http.ResponseWriter, r *http.Request) {
	if !requireMonitor(w, r, h.monitors) {
		return
	}
	value, err := h.repository.GetOpenByMonitor(r.Context(), r.PathValue("id"))
	if err != nil {
		writeInternalError(w, "get current incident", err)
		return
	}
	if value == nil {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(newIncidentResponse(*value, time.Now())); err != nil {
		logResponseWriteError("get current incident", err)
	}
}

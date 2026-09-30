package api

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"time"

	"github.com/gabipuzon/horus/internal/monitor"
)

type monitorRepository interface {
	Create(ctx context.Context, m *monitor.Monitor) error
	List(ctx context.Context) ([]*monitor.Monitor, error)
	GetByID(ctx context.Context, id string) (*monitor.Monitor, error)
	Delete(ctx context.Context, id string) error
	SetEnabled(ctx context.Context, id string, enabled bool) error
}

type MonitorHandler struct {
	repository monitorRepository
}

type createMonitorRequest struct {
	Name           string `json:"name"`
	URL            string `json:"url"`
	Interval       int64  `json:"interval_seconds"`
	Timeout        int64  `json:"timeout_seconds"`
	ExpectedStatus int    `json:"expected_status"`
}

type monitorResponse struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	URL            string `json:"url"`
	Interval       int64  `json:"interval_seconds"`
	Timeout        int64  `json:"timeout_seconds"`
	ExpectedStatus int    `json:"expected_status"`
	Enabled        bool   `json:"enabled"`
}

func NewMonitorHandler(repository monitorRepository) *MonitorHandler {
	return &MonitorHandler{
		repository: repository,
	}
}

func newMonitorResponse(m *monitor.Monitor) monitorResponse {
	return monitorResponse{
		ID:             m.ID,
		Name:           m.Name,
		URL:            m.URL,
		Interval:       int64(m.Interval / time.Second),
		Timeout:        int64(m.Timeout / time.Second),
		ExpectedStatus: m.ExpectedStatus,
		Enabled:        m.Enabled,
	}
}

func (h *MonitorHandler) Create(w http.ResponseWriter, r *http.Request) {
	var request createMonitorRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&request); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	// The schema stores these values as signed 32-bit seconds, and this bound
	// also keeps the conversion to time.Duration from overflowing.
	if request.Interval > 1<<31-1 || request.Timeout > 1<<31-1 {
		http.Error(w, "monitor interval or timeout is too large", http.StatusBadRequest)
		return
	}

	m, err := monitor.New(
		request.Name,
		request.URL,
		time.Duration(request.Interval)*time.Second,
		time.Duration(request.Timeout)*time.Second,
		request.ExpectedStatus,
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := h.repository.Create(r.Context(), m); err != nil {
		http.Error(w, "failed to create monitor", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	response := newMonitorResponse(m)

	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
		return
	}
}

func (h *MonitorHandler) List(w http.ResponseWriter, r *http.Request) {
	monitors, err := h.repository.List(r.Context())
	if err != nil {
		http.Error(w, "failed to list monitors", http.StatusInternalServerError)
		return
	}

	response := make([]monitorResponse, 0, len(monitors))

	for _, m := range monitors {
		response = append(response, newMonitorResponse(m))
	}

	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
		return
	}
}

func (h *MonitorHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	m, err := h.repository.GetByID(r.Context(), id)
	if err != nil {
		writeMonitorError(w, err, "failed to get monitor")
		return
	}

	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(newMonitorResponse(m)); err != nil {
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
		return
	}
}

func (h *MonitorHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	if err := h.repository.Delete(r.Context(), id); err != nil {
		writeMonitorError(w, err, "failed to delete monitor")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *MonitorHandler) Enable(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	if err := h.repository.SetEnabled(r.Context(), id, true); err != nil {
		writeMonitorError(w, err, "failed to enable monitor")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *MonitorHandler) Disable(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	if err := h.repository.SetEnabled(r.Context(), id, false); err != nil {
		writeMonitorError(w, err, "failed to disable monitor")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func writeMonitorError(w http.ResponseWriter, err error, internalMessage string) {
	if errors.Is(err, monitor.ErrNotFound) {
		http.Error(w, "monitor not found", http.StatusNotFound)
		return
	}
	http.Error(w, internalMessage, http.StatusInternalServerError)
}

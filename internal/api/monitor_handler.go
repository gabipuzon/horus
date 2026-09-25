package api

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/gabipuzon/horus/internal/monitor"
)

type MonitorRepository interface {
	Create(ctx context.Context, m *monitor.Monitor) error
	List(ctx context.Context) ([]*monitor.Monitor, error)
	GetByID(ctx context.Context, id string) (*monitor.Monitor, error)
	Delete(ctx context.Context, id string) error
}

type MonitorHandler struct {
	repository MonitorRepository
}

type createMonitorRequest struct {
	Name           string `json:"name"`
	URL            string `json:"url"`
	Interval       int    `json:"interval_seconds"`
	Timeout        int    `json:"timeout_seconds"`
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

func NewMonitorHandler(repository MonitorRepository) *MonitorHandler {
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

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
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
		http.Error(w, "monitor not found", http.StatusNotFound)
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
		http.Error(w, "failed to delete monitor", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

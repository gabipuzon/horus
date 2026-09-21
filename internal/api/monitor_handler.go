package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/gabipuzon/horus/internal/monitor"
)

type MonitorHandler struct {
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

func NewMonitorHandler() *MonitorHandler {
	return &MonitorHandler{}
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
		"temporary-id",
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

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	response := newMonitorResponse(m)

	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
		return
	}
}

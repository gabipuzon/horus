package main

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/gabipuzon/horus/internal/api"
)

type HealthResponse struct {
	Status string `json:"status"`
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	response := HealthResponse{
		Status: "ok",
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
		return
	}
}

func main() {
	mux := http.NewServeMux()

	monitorHandler := api.NewMonitorHandler()

	mux.HandleFunc("GET /health", healthHandler)
	mux.HandleFunc("POST /monitors", monitorHandler.Create)

	server := &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	log.Println("horus server listening on :8080")

	if err := server.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}

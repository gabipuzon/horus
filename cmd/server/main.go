package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"

	"github.com/gabipuzon/horus/internal/api"
	"github.com/gabipuzon/horus/internal/database"
	"github.com/gabipuzon/horus/internal/monitor"
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
	ctx := context.Background()

	db, err := database.NewPool(ctx, database.Config{
		Host:     "localhost",
		Port:     "5432",
		User:     "horus",
		Password: "horus",
		Name:     "horus",
	})
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	monitorRepository := monitor.NewRepository(db)
	monitorHandler := api.NewMonitorHandler(monitorRepository)

	checkRepository := monitor.NewCheckRepository(db)
	checkHandler := api.NewCheckHandler(checkRepository)

	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", healthHandler)
	mux.HandleFunc("POST /monitors", monitorHandler.Create)
	mux.HandleFunc("GET /monitors", monitorHandler.List)
	mux.HandleFunc("GET /monitors/{id}", monitorHandler.GetByID)
	mux.HandleFunc("DELETE /monitors/{id}", monitorHandler.Delete)
	mux.HandleFunc("PATCH /monitors/{id}/enable", monitorHandler.Enable)
	mux.HandleFunc("PATCH /monitors/{id}/disable", monitorHandler.Disable)

	mux.HandleFunc("GET /monitors/{id}/checks", checkHandler.ListByMonitor)
	server := &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	log.Println("horus server listening on :8080")

	if err := server.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}

package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gabipuzon/horus/internal/api"
	"github.com/gabipuzon/horus/internal/check"
	"github.com/gabipuzon/horus/internal/database"
	"github.com/gabipuzon/horus/internal/postgres"
	"github.com/gabipuzon/horus/internal/queue"
	"github.com/gabipuzon/horus/internal/scheduler"
	"github.com/gabipuzon/horus/internal/worker"
)

type healthResponse struct {
	Status string `json:"status"`
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	response := healthResponse{
		Status: "ok",
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(
			w,
			"failed to encode response",
			http.StatusInternalServerError,
		)
		return
	}
}

func main() {
	ctx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stop()

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

	redis := queue.NewRedis(queue.Config{
		Host: "localhost",
		Port: "6379",
	})

	if err := redis.Ping(ctx); err != nil {
		log.Fatal(err)
	}
	defer redis.Close()

	monitorRepository := postgres.NewMonitorRepository(db)
	monitorHandler := api.NewMonitorHandler(monitorRepository)

	checkRepository := postgres.NewCheckRepository(db)
	checkHandler := api.NewCheckHandler(
		checkRepository,
		checkRepository,
	)

	checker := check.NewChecker(http.DefaultClient)
	checkService := check.NewService(
		checker,
		checkRepository,
	)

	workerPool := worker.NewPool(
		ctx,
		3,
		redis,
		monitorRepository,
		checkService,
	)

	checkScheduler := scheduler.New(
		monitorRepository,
		redis,
	)

	go func() {
		if err := checkScheduler.Run(ctx); err != nil &&
			ctx.Err() == nil {
			log.Printf("scheduler stopped: %v", err)
		}
	}()

	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", healthHandler)
	mux.HandleFunc("POST /monitors", monitorHandler.Create)
	mux.HandleFunc("GET /monitors", monitorHandler.List)
	mux.HandleFunc("GET /monitors/{id}", monitorHandler.GetByID)
	mux.HandleFunc("DELETE /monitors/{id}", monitorHandler.Delete)
	mux.HandleFunc("PATCH /monitors/{id}/enable", monitorHandler.Enable)
	mux.HandleFunc("PATCH /monitors/{id}/disable", monitorHandler.Disable)

	mux.HandleFunc(
		"GET /monitors/{id}/checks",
		checkHandler.ListByMonitor,
	)
	mux.HandleFunc(
		"GET /monitors/{id}/summary",
		checkHandler.GetSummary,
	)

	server := &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	go func() {
		log.Println("horus server listening on :8080")

		if err := server.ListenAndServe(); err != nil &&
			err != http.ErrServerClosed {
			log.Printf("HTTP server failed: %v", err)
			stop()
		}
	}()

	<-ctx.Done()

	log.Println("shutting down horus")

	workerPool.Shutdown()

	shutdownCtx, cancel := context.WithTimeout(
		context.Background(),
		5*time.Second,
	)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("server shutdown failed: %v", err)
	}
}

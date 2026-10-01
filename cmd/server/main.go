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
	"github.com/gabipuzon/horus/internal/config"
	"github.com/gabipuzon/horus/internal/database"
	"github.com/gabipuzon/horus/internal/notification"
	"github.com/gabipuzon/horus/internal/postgres"
	"github.com/gabipuzon/horus/internal/queue"
	"github.com/gabipuzon/horus/internal/scheduler"
	"github.com/gabipuzon/horus/internal/worker"
)

type healthResponse struct {
	Status string `json:"status"`
}

type dependencyPinger interface {
	Ping(ctx context.Context) error
}

const readinessTimeout = time.Second

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	response := healthResponse{
		Status: "ok",
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("failed to write liveness response (%T)", err)
	}
}

func readyHandler(postgres, redis dependencyPinger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), readinessTimeout)
		defer cancel()

		postgresErr := postgres.Ping(ctx)
		redisErr := redis.Ping(ctx)
		status := http.StatusOK
		response := healthResponse{Status: "ready"}
		if postgresErr != nil || redisErr != nil || ctx.Err() != nil {
			status = http.StatusServiceUnavailable
			response.Status = "not_ready"
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		if err := json.NewEncoder(w).Encode(response); err != nil {
			log.Printf("failed to write readiness response (%T)", err)
		}
	}
}

func newNotifier(cfg config.Config) notification.Notifier {
	if cfg.DiscordWebhookURL == "" {
		log.Println("Discord notifications disabled")
		return nil
	}
	log.Println("Discord notifications enabled")
	return notification.NewDiscord(cfg.DiscordWebhookURL, &http.Client{})
}

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}
	ctx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stop()

	db, err := database.NewPool(ctx, database.Config{
		Host:     cfg.DatabaseHost,
		Port:     cfg.DatabasePort,
		User:     cfg.DatabaseUser,
		Password: cfg.DatabasePassword,
		Name:     cfg.DatabaseName,
	})
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	redis := queue.NewRedis(queue.Config{
		Host: cfg.RedisHost,
		Port: cfg.RedisPort,
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
		monitorRepository,
	)
	incidentHandler := api.NewIncidentHandler(postgres.NewIncidentRepository(db), monitorRepository)

	checker := check.NewChecker(http.DefaultClient)
	notifier := newNotifier(cfg)
	checkService := check.NewService(
		checker,
		checkRepository,
		notifier,
	)

	workerPool := worker.NewPool(
		ctx,
		cfg.WorkerCount,
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
	mux.HandleFunc("GET /ready", readyHandler(db, redis))
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
	mux.HandleFunc("GET /monitors/{id}/incidents", incidentHandler.ListByMonitor)
	mux.HandleFunc("GET /monitors/{id}/incidents/current", incidentHandler.GetCurrent)

	server := &http.Server{
		Addr:    cfg.HTTPAddr,
		Handler: mux,
	}

	go func() {
		log.Printf("horus server listening on %s", cfg.HTTPAddr)

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

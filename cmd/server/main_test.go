package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gabipuzon/horus/internal/config"
	"github.com/gabipuzon/horus/internal/database"
	"github.com/gabipuzon/horus/internal/notification"
	"github.com/gabipuzon/horus/internal/queue"
)

type testPinger struct {
	err   error
	calls int
	wait  bool
}

func (p *testPinger) Ping(ctx context.Context) error {
	p.calls++
	if p.wait {
		<-ctx.Done()
		return ctx.Err()
	}
	return p.err
}

func TestHealthHandler(t *testing.T) {
	recorder := httptest.NewRecorder()
	healthHandler(recorder, httptest.NewRequest(http.MethodGet, "/health", nil))
	if recorder.Code != http.StatusOK || recorder.Body.String() != "{\"status\":\"ok\"}\n" {
		t.Fatalf("unexpected liveness response: %d %s", recorder.Code, recorder.Body.String())
	}
}

func TestHealthDoesNotCheckDependencies(t *testing.T) {
	postgres := &testPinger{err: errors.New("postgres unavailable")}
	redis := &testPinger{err: errors.New("redis unavailable")}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", healthHandler)
	mux.HandleFunc("GET /ready", readyHandler(postgres, redis))

	for _, path := range []string{"/health", "/ready", "/health"} {
		recorder := httptest.NewRecorder()
		mux.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, path, nil))
		if path == "/health" {
			if recorder.Code != http.StatusOK || recorder.Body.String() != "{\"status\":\"ok\"}\n" {
				t.Fatalf("liveness changed with unavailable dependencies: %d %s", recorder.Code, recorder.Body.String())
			}
		} else if recorder.Code != http.StatusServiceUnavailable {
			t.Fatalf("expected failed readiness, got %d", recorder.Code)
		}
	}
	if postgres.calls != 1 || redis.calls != 1 {
		t.Fatalf("liveness called a dependency: postgres=%d redis=%d", postgres.calls, redis.calls)
	}
}

func TestReadyHandlerDependencyStates(t *testing.T) {
	for _, test := range []struct {
		name        string
		postgresErr error
		redisErr    error
		wantStatus  int
		wantBody    string
	}{
		{"both ready", nil, nil, http.StatusOK, "ready"},
		{"postgres unavailable", errors.New("postgres password=secret"), nil, http.StatusServiceUnavailable, "not_ready"},
		{"redis unavailable", nil, errors.New("redis token=secret"), http.StatusServiceUnavailable, "not_ready"},
		{"both unavailable", errors.New("postgres password=secret"), errors.New("redis token=secret"), http.StatusServiceUnavailable, "not_ready"},
	} {
		t.Run(test.name, func(t *testing.T) {
			postgres := &testPinger{err: test.postgresErr}
			redis := &testPinger{err: test.redisErr}
			recorder := httptest.NewRecorder()
			readyHandler(postgres, redis)(recorder, httptest.NewRequest(http.MethodGet, "/ready", nil))
			if recorder.Code != test.wantStatus {
				t.Fatalf("expected HTTP %d, got %d", test.wantStatus, recorder.Code)
			}
			if recorder.Header().Get("Content-Type") != "application/json" {
				t.Fatalf("unexpected content type: %s", recorder.Header().Get("Content-Type"))
			}
			var body healthResponse
			if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil || body.Status != test.wantBody {
				t.Fatalf("unexpected readiness response: %s", recorder.Body.String())
			}
			if strings.Contains(recorder.Body.String(), "secret") {
				t.Fatal("readiness response leaked dependency error")
			}
			if postgres.calls != 1 || redis.calls != 1 {
				t.Fatalf("expected one check per dependency, got postgres=%d redis=%d", postgres.calls, redis.calls)
			}
		})
	}
}

func TestReadyHandlerTimeout(t *testing.T) {
	for _, stalled := range []string{"postgres", "redis"} {
		t.Run(stalled, func(t *testing.T) {
			postgres := &testPinger{wait: stalled == "postgres"}
			redis := &testPinger{wait: stalled == "redis"}
			recorder := httptest.NewRecorder()
			start := time.Now()
			readyHandler(postgres, redis)(recorder, httptest.NewRequest(http.MethodGet, "/ready", nil))
			if recorder.Code != http.StatusServiceUnavailable || recorder.Body.String() != "{\"status\":\"not_ready\"}\n" || time.Since(start) > 2*time.Second {
				t.Fatalf("expected bounded HTTP 503, got %d after %s: %s", recorder.Code, time.Since(start), recorder.Body.String())
			}
		})
	}
}

func TestReadyHandlerWithLocalDependencies(t *testing.T) {
	ctx := context.Background()
	postgres, err := database.NewPool(ctx, database.Config{
		Host: "localhost", Port: "5432", User: "horus", Password: "horus", Name: "horus",
	})
	if err != nil {
		t.Fatal(err)
	}
	defer postgres.Close()
	redis := queue.NewRedis(queue.Config{Host: "localhost", Port: "6379"})
	defer redis.Close()
	recorder := httptest.NewRecorder()
	readyHandler(postgres, redis)(recorder, httptest.NewRequest(http.MethodGet, "/ready", nil))
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected local dependencies to be ready, got %d", recorder.Code)
	}
	if err := redis.Close(); err != nil {
		t.Fatal(err)
	}
	recorder = httptest.NewRecorder()
	readyHandler(postgres, redis)(recorder, httptest.NewRequest(http.MethodGet, "/ready", nil))
	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected closed Redis client to make readiness fail, got %d", recorder.Code)
	}
	recorder = httptest.NewRecorder()
	healthHandler(recorder, httptest.NewRequest(http.MethodGet, "/health", nil))
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected liveness after Redis failure, got %d", recorder.Code)
	}
}

func TestNotifierWiringFromEnvironment(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()
	for _, enabled := range []bool{false, true} {
		t.Run(map[bool]string{false: "disabled", true: "enabled"}[enabled], func(t *testing.T) {
			webhook := ""
			state := "disabled"
			if enabled {
				webhook = server.URL + "/api/webhooks/test/secret-token"
				state = "enabled"
			}
			t.Setenv("HORUS_DISCORD_WEBHOOK_URL", webhook)
			cfg, err := config.Load()
			if err != nil {
				t.Fatal(err)
			}
			var output bytes.Buffer
			previous := log.Writer()
			log.SetOutput(&output)
			t.Cleanup(func() { log.SetOutput(previous) })
			notifier := newNotifier(cfg)
			if (notifier != nil) != enabled {
				t.Fatal("notifier does not match environment configuration")
			}
			if !strings.Contains(output.String(), "Discord notifications "+state) {
				t.Fatalf("missing startup state: %s", output.String())
			}
			if strings.Contains(output.String(), server.URL) || strings.Contains(output.String(), "secret-token") {
				t.Fatal("startup log exposes webhook configuration")
			}
			if enabled {
				if err := notifier.Notify(context.Background(), notification.Event{State: notification.Down}); err != nil {
					t.Fatal(err)
				}
			}
		})
	}
	if requests.Load() != 1 {
		t.Fatalf("expected one configured webhook request, got %d", requests.Load())
	}
}

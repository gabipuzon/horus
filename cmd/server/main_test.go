package main

import (
	"bytes"
	"context"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/gabipuzon/horus/internal/config"
	"github.com/gabipuzon/horus/internal/notification"
)

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

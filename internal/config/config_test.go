package config

import (
	"os"
	"strings"
	"testing"
)

func unsetConfigEnvironment(t *testing.T) {
	t.Helper()
	for _, key := range []string{
		"HORUS_DB_HOST", "HORUS_DB_PORT", "HORUS_DB_USER", "HORUS_DB_PASSWORD", "HORUS_DB_NAME",
		"HORUS_REDIS_HOST", "HORUS_REDIS_PORT", "HORUS_HTTP_ADDR", "HORUS_WORKER_COUNT",
		"HORUS_DISCORD_WEBHOOK_URL",
	} {
		t.Setenv(key, "")
		if err := os.Unsetenv(key); err != nil {
			t.Fatal(err)
		}
	}
}

func TestLoadRequiresExportedDiscordWebhook(t *testing.T) {
	unsetConfigEnvironment(t)
	t.Chdir(t.TempDir())
	const webhook = "https://discord.example/api/webhooks/test/token"
	if err := os.WriteFile(".env", []byte("HORUS_DISCORD_WEBHOOK_URL="+webhook+"\n"), 0600); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.DiscordWebhookURL != "" {
		t.Fatal("a .env file alone must not enable notifications")
	}
	t.Setenv("HORUS_DISCORD_WEBHOOK_URL", webhook)
	cfg, err = Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.DiscordWebhookURL != webhook {
		t.Fatal("exported webhook must enable notification configuration")
	}
}

func TestLoadDefaults(t *testing.T) {
	unsetConfigEnvironment(t)
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.DatabaseHost != "localhost" || cfg.DatabasePort != "5432" || cfg.DatabaseUser != "horus" || cfg.DatabasePassword != "horus" || cfg.DatabaseName != "horus" ||
		cfg.RedisHost != "localhost" || cfg.RedisPort != "6379" || cfg.HTTPAddr != ":8080" || cfg.WorkerCount != 3 || cfg.DiscordWebhookURL != "" {
		t.Fatalf("unexpected defaults: %+v", cfg)
	}
}

func TestLoadOverrides(t *testing.T) {
	unsetConfigEnvironment(t)
	for key, value := range map[string]string{
		"HORUS_DB_HOST": "db.example", "HORUS_DB_PORT": "15432", "HORUS_DB_USER": "app",
		"HORUS_DB_PASSWORD": "", "HORUS_DB_NAME": "appdb", "HORUS_REDIS_HOST": "cache.example",
		"HORUS_REDIS_PORT": "16379", "HORUS_HTTP_ADDR": "127.0.0.1:18080", "HORUS_WORKER_COUNT": "4",
	} {
		t.Setenv(key, value)
	}
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.DatabaseHost != "db.example" || cfg.DatabasePort != "15432" || cfg.DatabaseUser != "app" || cfg.DatabasePassword != "" || cfg.DatabaseName != "appdb" ||
		cfg.RedisHost != "cache.example" || cfg.RedisPort != "16379" || cfg.HTTPAddr != "127.0.0.1:18080" || cfg.WorkerCount != 4 {
		t.Fatalf("environment overrides were not applied: %+v", cfg)
	}
}

func TestLoadOptionalDiscordWebhook(t *testing.T) {
	unsetConfigEnvironment(t)
	t.Setenv("HORUS_DISCORD_WEBHOOK_URL", "https://discord.example/webhook")
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.DiscordWebhookURL != "https://discord.example/webhook" {
		t.Fatalf("unexpected webhook URL: %q", cfg.DiscordWebhookURL)
	}
	for _, value := range []string{"not-a-url", "ftp://discord.example/webhook", "https://user:secret@discord.example/webhook", "https://:123/webhook"} {
		t.Setenv("HORUS_DISCORD_WEBHOOK_URL", value)
		if _, err := Load(); err == nil || !strings.Contains(err.Error(), "HORUS_DISCORD_WEBHOOK_URL") || strings.Contains(err.Error(), "secret") {
			t.Fatalf("expected safe webhook validation error for %q, got %v", value, err)
		}
	}
	t.Setenv("HORUS_DISCORD_WEBHOOK_URL", "   ")
	cfg, err = Load()
	if err != nil || cfg.DiscordWebhookURL != "" {
		t.Fatalf("blank webhook should disable notifications, got %q, %v", cfg.DiscordWebhookURL, err)
	}
}

func TestLoadRejectsInvalidValues(t *testing.T) {
	unsetConfigEnvironment(t)
	for _, test := range []struct {
		key   string
		value string
	}{
		{"HORUS_WORKER_COUNT", ""}, {"HORUS_WORKER_COUNT", "abc"}, {"HORUS_WORKER_COUNT", "0"}, {"HORUS_WORKER_COUNT", "-1"}, {"HORUS_WORKER_COUNT", "1001"},
		{"HORUS_DB_PORT", ""}, {"HORUS_DB_PORT", "abc"}, {"HORUS_DB_PORT", "0"}, {"HORUS_DB_PORT", "70000"},
		{"HORUS_REDIS_PORT", ""}, {"HORUS_REDIS_PORT", "abc"}, {"HORUS_REDIS_PORT", "0"}, {"HORUS_REDIS_PORT", "70000"},
		{"HORUS_DB_HOST", ""}, {"HORUS_DB_USER", ""}, {"HORUS_DB_NAME", ""},
		{"HORUS_REDIS_HOST", ""}, {"HORUS_HTTP_ADDR", ""},
	} {
		t.Run(test.key+"="+test.value, func(t *testing.T) {
			t.Setenv(test.key, test.value)
			if _, err := Load(); err == nil || !strings.Contains(err.Error(), test.key) {
				t.Fatalf("expected %s validation error, got %v", test.key, err)
			}
		})
	}
	for _, count := range []string{"1", "1000"} {
		t.Setenv("HORUS_WORKER_COUNT", count)
		if _, err := Load(); err != nil {
			t.Fatalf("worker count %s should be allowed: %v", count, err)
		}
	}
}

package config

import (
	"os"
	"testing"
)

func TestLoadRequiresExportedDiscordWebhook(t *testing.T) {
	t.Chdir(t.TempDir())
	t.Setenv("HORUS_DISCORD_WEBHOOK_URL", "")
	if err := os.Unsetenv("HORUS_DISCORD_WEBHOOK_URL"); err != nil {
		t.Fatal(err)
	}
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
	t.Setenv("HORUS_WORKER_COUNT", "3")
	t.Setenv("HORUS_DISCORD_WEBHOOK_URL", "")
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.DatabaseHost != "localhost" || cfg.RedisPort != "6379" || cfg.HTTPAddr != ":8080" || cfg.WorkerCount != 3 || cfg.DiscordWebhookURL != "" {
		t.Fatalf("unexpected defaults: %+v", cfg)
	}
}

func TestLoadOptionalDiscordWebhook(t *testing.T) {
	t.Setenv("HORUS_DISCORD_WEBHOOK_URL", "https://discord.example/webhook")
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.DiscordWebhookURL != "https://discord.example/webhook" {
		t.Fatalf("unexpected webhook URL: %q", cfg.DiscordWebhookURL)
	}
	t.Setenv("HORUS_DISCORD_WEBHOOK_URL", "not-a-url")
	if _, err := Load(); err == nil {
		t.Fatal("expected invalid webhook URL to fail")
	}
}

func TestLoadRejectsInvalidValues(t *testing.T) {
	t.Setenv("HORUS_WORKER_COUNT", "0")
	if _, err := Load(); err == nil {
		t.Fatal("expected invalid worker count to fail")
	}
	t.Setenv("HORUS_WORKER_COUNT", "3")
	t.Setenv("HORUS_DB_PORT", "70000")
	if _, err := Load(); err == nil {
		t.Fatal("expected invalid database port to fail")
	}
}

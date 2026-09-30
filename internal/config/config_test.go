package config

import "testing"

func TestLoadDefaults(t *testing.T) {
	t.Setenv("HORUS_WORKER_COUNT", "3")
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.DatabaseHost != "localhost" || cfg.RedisPort != "6379" || cfg.HTTPAddr != ":8080" || cfg.WorkerCount != 3 {
		t.Fatalf("unexpected defaults: %+v", cfg)
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

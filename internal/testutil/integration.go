package testutil

import (
	"net/url"
	"os"
	"testing"
)

// RequireIntegration skips tests that need external PostgreSQL or Redis unless
// the integration suite is explicitly enabled.
func RequireIntegration(t *testing.T) {
	t.Helper()
	if os.Getenv("HORUS_TEST_INTEGRATION") != "1" {
		t.Skip("set HORUS_TEST_INTEGRATION=1 to run PostgreSQL/Redis integration tests")
	}
}

func Env(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func DatabaseURL() string {
	value := url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(Env("HORUS_DB_USER", "horus"), Env("HORUS_DB_PASSWORD", "horus")),
		Host:   Env("HORUS_DB_HOST", "localhost") + ":" + Env("HORUS_DB_PORT", "5432"),
		Path:   "/" + Env("HORUS_DB_NAME", "horus"),
	}
	return value.String()
}

func RedisAddress() string {
	return Env("HORUS_REDIS_HOST", "localhost") + ":" + Env("HORUS_REDIS_PORT", "6379")
}

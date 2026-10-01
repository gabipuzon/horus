package config

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	DatabaseHost      string
	DatabasePort      string
	DatabaseUser      string
	DatabasePassword  string
	DatabaseName      string
	RedisHost         string
	RedisPort         string
	HTTPAddr          string
	WorkerCount       int
	DiscordWebhookURL string
}

func Load() (Config, error) {
	workerCount, err := intEnv("HORUS_WORKER_COUNT", 3)
	if err != nil || workerCount < 1 || workerCount > 1000 {
		return Config{}, fmt.Errorf("HORUS_WORKER_COUNT must be an integer between 1 and 1000")
	}
	cfg := Config{
		DatabaseHost:      stringEnv("HORUS_DB_HOST", "localhost"),
		DatabasePort:      stringEnv("HORUS_DB_PORT", "5432"),
		DatabaseUser:      stringEnv("HORUS_DB_USER", "horus"),
		DatabasePassword:  stringEnv("HORUS_DB_PASSWORD", "horus"),
		DatabaseName:      stringEnv("HORUS_DB_NAME", "horus"),
		RedisHost:         stringEnv("HORUS_REDIS_HOST", "localhost"),
		RedisPort:         stringEnv("HORUS_REDIS_PORT", "6379"),
		HTTPAddr:          stringEnv("HORUS_HTTP_ADDR", ":8080"),
		WorkerCount:       workerCount,
		DiscordWebhookURL: strings.TrimSpace(stringEnv("HORUS_DISCORD_WEBHOOK_URL", "")),
	}
	if cfg.DiscordWebhookURL != "" {
		webhook, err := url.Parse(cfg.DiscordWebhookURL)
		if err != nil || (webhook.Scheme != "http" && webhook.Scheme != "https") || webhook.Hostname() == "" || webhook.User != nil {
			return Config{}, fmt.Errorf("HORUS_DISCORD_WEBHOOK_URL must be an absolute HTTP or HTTPS URL without credentials")
		}
	}
	for key, value := range map[string]string{
		"HORUS_DB_HOST": cfg.DatabaseHost, "HORUS_DB_PORT": cfg.DatabasePort,
		"HORUS_DB_USER": cfg.DatabaseUser, "HORUS_DB_NAME": cfg.DatabaseName,
		"HORUS_REDIS_HOST": cfg.RedisHost, "HORUS_REDIS_PORT": cfg.RedisPort,
		"HORUS_HTTP_ADDR": cfg.HTTPAddr,
	} {
		if strings.TrimSpace(value) == "" {
			return Config{}, fmt.Errorf("%s cannot be empty", key)
		}
	}
	for key, value := range map[string]string{"HORUS_DB_PORT": cfg.DatabasePort, "HORUS_REDIS_PORT": cfg.RedisPort} {
		port, err := strconv.Atoi(value)
		if err != nil || port < 1 || port > 65535 {
			return Config{}, fmt.Errorf("%s must be a port between 1 and 65535", key)
		}
	}
	return cfg, nil
}

func stringEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func intEnv(key string, fallback int) (int, error) {
	value, ok := os.LookupEnv(key)
	if !ok {
		return fallback, nil
	}
	return strconv.Atoi(value)
}

// Package config loads notify configuration from environment variables with fail-fast validation.
package config

import (
	"fmt"
	"os"
)

// Config holds all tunable parameters for the notify service.
type Config struct {
	RedisAddr     string
	WebhookURL    string
	WebhookSecret string
	HTTPAddr      string
}

// Load reads configuration from environment variables and returns a Config or an error
// naming the first missing required variable.
func Load() (Config, error) {
	var cfg Config
	var err error

	if cfg.RedisAddr, err = mustenv("REDIS_ADDR"); err != nil {
		return Config{}, err
	}
	if cfg.WebhookURL, err = mustenv("WEBHOOK_URL"); err != nil {
		return Config{}, err
	}
	if cfg.WebhookSecret, err = mustenv("WEBHOOK_SECRET"); err != nil {
		return Config{}, err
	}
	cfg.HTTPAddr = getenv("HTTP_ADDR", ":8090")

	return cfg, nil
}

// mustenv returns the value of an env var or an error if it is empty / unset.
func mustenv(key string) (string, error) {
	v := os.Getenv(key)
	if v == "" {
		return "", fmt.Errorf("required env var %s is not set", key)
	}
	return v, nil
}

// getenv returns the env var value or a default when the var is empty / unset.
func getenv(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}

// Package config loads gateway configuration from environment variables with fail-fast validation.
package config

import (
	"fmt"
	"os"
	"time"
)

// Config holds all tunable parameters for the gateway service.
type Config struct {
	IndexerGRPCAddr string
	HTTPAddr        string
	RequestTimeout  time.Duration
}

// Load reads configuration from environment variables and returns a Config or an error
// naming the first missing required variable.
func Load() (Config, error) {
	addr, err := mustenv("INDEXER_GRPC_ADDR")
	if err != nil {
		return Config{}, err
	}

	return Config{
		IndexerGRPCAddr: addr,
		HTTPAddr:        getenv("HTTP_ADDR", ":8080"),
		RequestTimeout:  parseDuration(getenv("REQUEST_TIMEOUT", "5s")),
	}, nil
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

// parseDuration parses a duration string, returning 5s on error.
func parseDuration(s string) time.Duration {
	d, err := time.ParseDuration(s)
	if err != nil {
		return 5 * time.Second
	}
	return d
}

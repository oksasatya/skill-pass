// Package config loads indexer configuration from environment variables with fail-fast validation.
package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config holds all tunable parameters for the indexer service.
// Required fields have no default; Load returns an error if any are missing.
type Config struct {
	DatabaseURL     string
	EthRPCURL       string
	ContractAddress string
	ChainID         int64
	GRPCAddr        string
	StartBlock      uint64
	BatchSize       uint64
	PollInterval    time.Duration
}

// Load reads configuration from environment variables and returns a Config or an error
// naming the first missing required variable.
func Load() (Config, error) {
	var cfg Config
	var err error

	if cfg.DatabaseURL, err = mustenv("DATABASE_URL"); err != nil {
		return Config{}, err
	}
	if cfg.EthRPCURL, err = mustenv("ETH_RPC_URL"); err != nil {
		return Config{}, err
	}
	if cfg.ContractAddress, err = mustenv("CONTRACT_ADDRESS"); err != nil {
		return Config{}, err
	}

	chainStr, err := mustenv("CHAIN_ID")
	if err != nil {
		return Config{}, err
	}
	cfg.ChainID, err = strconv.ParseInt(chainStr, 10, 64)
	if err != nil {
		return Config{}, fmt.Errorf("CHAIN_ID: %w", err)
	}

	cfg.GRPCAddr = getenv("GRPC_ADDR", ":50051")
	cfg.StartBlock = parseUint64(getenv("START_BLOCK", "0"))
	cfg.BatchSize = parseUint64(getenv("BATCH_SIZE", "2000"))
	cfg.PollInterval = parseDuration(getenv("POLL_INTERVAL", "5s"))

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

// parseUint64 converts a string to uint64, returning 0 on error.
func parseUint64(s string) uint64 {
	n, _ := strconv.ParseUint(s, 10, 64)
	return n
}

// parseDuration parses a duration string, returning 5s on error.
func parseDuration(s string) time.Duration {
	d, err := time.ParseDuration(s)
	if err != nil {
		return 5 * time.Second
	}
	return d
}

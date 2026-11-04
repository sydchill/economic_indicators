package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config holds every environment-driven setting used by the API server
// and the ingestion worker. Each field may be empty when loaded by a
// binary that does not require it; validation is performed by the
// per-binary loaders (LoadServer, LoadWorker).
type Config struct {
	// Required by both binaries.
	DatabaseURL string

	// Server-only. Ignored by the worker.
	HTTPAddr       string
	GoogleClientID string

	// Worker-only. Ignored by the server.
	IngestInterval       time.Duration
	IngestOnBoot         bool
	YieldsIngestInterval time.Duration
}

// LoadCommon loads and validates the settings shared by both binaries.
// Only DATABASE_URL is mandatory at this layer; everything else has a
// safe default so the worker can start with a minimal environment.
func LoadCommon() (*Config, error) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}

	interval, err := parseDurationEnv("INGEST_INTERVAL", 30*time.Minute)
	if err != nil {
		return nil, err
	}

	return &Config{
		DatabaseURL:    dsn,
		HTTPAddr:       envOrDefault("HTTP_ADDR", ":8080"),
		GoogleClientID: os.Getenv("GOOGLE_CLIENT_ID"),
		IngestInterval: interval,
		IngestOnBoot:   parseBoolEnv("INGEST_ON_BOOT", false),
	}, nil
}

// LoadServer loads the config and enforces requirements specific to the
// HTTP API server. GOOGLE_CLIENT_ID must be set because the auth service
// relies on it for ID-token verification; a missing value would make
// idtoken.Validate skip the audience check and accept tokens from any
// Google project.
func LoadServer() (*Config, error) {
	c, err := LoadCommon()
	if err != nil {
		return nil, err
	}
	if c.GoogleClientID == "" {
		return nil, fmt.Errorf("GOOGLE_CLIENT_ID is required for the API server")
	}
	if c.HTTPAddr == "" {
		return nil, fmt.Errorf("HTTP_ADDR must not be empty")
	}
	return c, nil
}

// LoadWorker is the worker's counterpart to LoadServer. It deliberately
// does not require GOOGLE_CLIENT_ID because the ingestion path never
// touches OAuth.
func LoadWorker() (*Config, error) {
	return LoadCommon()
}

// --- helpers ---

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func parseDurationEnv(key string, fallback time.Duration) (time.Duration, error) {
	v := os.Getenv(key)
	if v == "" {
		return fallback, nil
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return 0, fmt.Errorf("%s is not a valid duration (e.g. 30m, 1h): %w", key, err)
	}
	if d <= 0 {
		return 0, fmt.Errorf("%s must be a positive duration", key)
	}
	return d, nil
}

func parseBoolEnv(key string, fallback bool) bool {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	if b, err := strconv.ParseBool(v); err == nil {
		return b
	}
	return fallback
}

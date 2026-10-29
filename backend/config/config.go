package config

import (
	"fmt"
	"os"
)

// Config database setup
type Config struct {
	DatabaseURL string
	HTTPAddr    string
}

func Load() (*Config, error) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		return nil, fmt.Errorf("DATABASE_URL not set")
	}
	addr := os.Getenv("HTTP_ADDR")
	if addr == "" {
		addr = ":8080"
	}
	return &Config{DatabaseURL: dsn, HTTPAddr: addr}, nil
}



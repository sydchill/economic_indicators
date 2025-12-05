package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

// Config database setup
type Config struct {
	Addr  string
	DBDSN string
}

// Load load env info
func Load() *Config {
	// load .env in dev (ignore error if missing)
	_ = godotenv.Load()

	addr := getEnv("ADDR", ":8080")

	dsn := os.Getenv("DB_DSN")
	if dsn == "" {
		log.Fatal("DB_DSN is required, e.g. user:pass@tcp(localhost:3306)/trading?charset=utf8mb4&parseTime=True&loc=UTC")
	}

	return &Config{
		Addr:  addr,
		DBDSN: dsn,
	}
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

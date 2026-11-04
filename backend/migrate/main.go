package main

import (
	"context"
	"log"
	"time"

	"github.com/joho/godotenv"

	"github.com/sydchill/economic_indicators/shared/config"
	"github.com/sydchill/economic_indicators/shared/db"
)

func main() {
	_ = godotenv.Load()

	cfg, err := config.LoadCommon()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	bunDB, err := db.New(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("db: %v", err)
	}
	defer bunDB.Close()

	if err := db.Migrate(bunDB); err != nil {
		log.Fatalf("migrate: %v", err)
	}
	log.Println("migrations applied")
}
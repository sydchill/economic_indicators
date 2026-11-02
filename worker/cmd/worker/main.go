// worker/cmd/worker/main.go
package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"

	"github.com/sydchill/economic_indicators/shared/config"
	"github.com/sydchill/economic_indicators/shared/db"
	"github.com/sydchill/economic_indicators/worker/internal/ingest"
)

func main() {
	_ = godotenv.Load()

	cfg, err := config.LoadCommon()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	bunDB, err := db.New(ctx, cfg.DatabaseURL)
	cancel()
	if err != nil {
		log.Fatalf("db: %v", err)
	}
	defer bunDB.Close()

	// NO db.Migrate() here. Migrations are owned by cmd/migrate (Component 5.5)
	// or a CI step, so multiple worker replicas never race on schema changes.

	client := ingest.NewBiquoteClient("")
	worker := ingest.NewWorker(bunDB, client)

	stopCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if cfg.IngestOnBoot {
		log.Println("ingest: running once on boot")
		ictx, icancel := context.WithTimeout(stopCtx, 5*time.Minute)
		if err := worker.RunIngestOnce(ictx, ingest.MajorCountryCodes); err != nil {
			log.Printf("ingest: %v", err)
		}
		icancel()
	}

	ticker := time.NewTicker(cfg.IngestInterval)
	defer ticker.Stop()

	log.Printf("worker ready; interval=%s", cfg.IngestInterval)
	for {
		select {
		case <-ticker.C:
			ictx, icancel := context.WithTimeout(context.Background(), 5*time.Minute)
			if err := worker.RunIngestOnce(ictx, ingest.MajorCountryCodes); err != nil {
				log.Printf("ingest: %v", err)
			}
			icancel()
		case <-stopCtx.Done():
			log.Println("worker shutting down")
			return
		}
	}
}

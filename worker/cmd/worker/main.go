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
	"github.com/sydchill/economic_indicators/worker/internal/ingest/yields"
	"github.com/sydchill/economic_indicators/worker/internal/seed"
)

func main() {
	_ = godotenv.Load()

	cfg, err := config.LoadWorker()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	bunDB, err := db.New(ctx, cfg.DatabaseURL)
	cancel()
	if err != nil {
		log.Fatalf("db: %v", err)
	}
	defer bunDB.Close()

	mappings, err := seed.LoadMapping("seed/indicator_mapping.json")
	if err != nil {
		log.Fatalf("mapping: %v", err)
	}

	econWorker := ingest.NewWorker(bunDB, ingest.NewBiquoteClient(""), mappings)
	yieldWorker := yields.NewWorker(bunDB)

	stopCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	runEcon := func() {
		c, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()
		if err := econWorker.RunIngestOnce(c, ingest.MajorCountryCodes); err != nil {
			log.Printf("econ ingest: %v", err)
		}
	}
	runYields := func() {
		c, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()
		if err := yieldWorker.RunIngestOnce(c); err != nil {
			log.Printf("yields ingest: %v", err)
		}
	}

	if cfg.IngestOnBoot {
		runEcon()
		runYields()
	}

	econTicker := time.NewTicker(cfg.IngestInterval)
	defer econTicker.Stop()
	yieldTicker := time.NewTicker(cfg.YieldsIngestInterval)
	defer yieldTicker.Stop()

	log.Printf("worker ready: econ=%s yields=%s", cfg.IngestInterval, cfg.YieldsIngestInterval)
	for {
		select {
		case <-econTicker.C:
			runEcon()
		case <-yieldTicker.C:
			runYields()
		case <-stopCtx.Done():
			log.Println("worker shutting down")
			return
		}
	}
}

package main

import (
	"context"
	"flag"
	"log"
	"time"

	"github.com/joho/godotenv"

	"github.com/sydchill/economic_indicators/shared/config"
	"github.com/sydchill/economic_indicators/shared/db"
	"github.com/sydchill/economic_indicators/worker/internal/ingest/yields"
)

func main() {
	file := flag.String("file", "", "CSV file to import (required)")
	flag.Parse()

	if *file == "" {
		log.Fatal("--file is required")
	}

	_ = godotenv.Load()
	cfg, err := config.LoadWorker()
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

	n, err := yields.ImportCSV(ctx, bunDB, *file)
	if err != nil {
		log.Fatalf("import: %v (imported %d rows before failure)", err, n)
	}
	log.Printf("imported %d yield rows from %s", n, *file)
}

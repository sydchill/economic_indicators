package main

import (
	"context"
	"flag"
	"log"
	"os"
	"sort"
	"time"

	"github.com/joho/godotenv"

	"github.com/sydchill/economic_indicators/shared/config"
	"github.com/sydchill/economic_indicators/shared/db"
	seed "github.com/sydchill/economic_indicators/worker/internal/ingest"
)

func main() {
	file := flag.String("file", "seed/scoring_config.json", "path to config JSON")
	apply := flag.Bool("apply", false, "write to database (default: dry-run)")
	flag.Parse()

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

	raw, err := seed.LoadFile(*file)
	if err != nil {
		log.Fatalf("load: %v", err)
	}
	log.Printf("loaded %s: version=%d templates=%d assets=%d",
		*file, raw.Version, len(raw.Templates), len(raw.Assets))

	plan, err := seed.Resolve(raw)
	if err != nil {
		log.Fatalf("resolve: %v", err)
	}
	log.Printf("resolved plan: %d assets", len(plan.Assets))

	if err := seed.Validate(ctx, bunDB, plan); err != nil {
		log.Fatalf("%v", err)
	}
	log.Println("validation passed")

	sort.Slice(plan.Assets, func(i, j int) bool {
		return plan.Assets[i].AssetID < plan.Assets[j].AssetID
	})
	for _, ap := range plan.Assets {
		log.Printf("  %-6s  %d category weights  %d indicator rows",
			ap.AssetID, len(ap.CategoryWeights), len(ap.Indicators))
	}

	if !*apply {
		log.Println("dry-run: pass --apply to write to database")
		return
	}

	if err := seed.Apply(ctx, bunDB, plan); err != nil {
		log.Fatalf("apply: %v", err)
	}
	log.Println("seed applied successfully")
	os.Exit(0)
}

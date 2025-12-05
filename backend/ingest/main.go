package main

import (
	"context"
	"economic_indicator/config"
	"economic_indicator/db"
	"log"
	"os"
)

func main() {
	cfg := config.Load()
	bunDB := db.Open(cfg.DBDSN)

	teAPIKEY := os.Getenv("TE_KEY")
	if teAPIKEY == "" {
		log.Fatal("TE_KEY env vars are required for TradingEconomics")
	}

	ctx := context.Background()

	// Currency → Country mapping for TradingEconomics
	currencyCountries := map[string]string{
		"USD": "united states",
		"EUR": "euro area",
		"GBP": "united kingdom",
		"JPY": "japan",
		"AUD": "australia",
		"NZD": "new zealand",
		"CHF": "switzerland",
	}

	for cur, country := range currencyCountries {
		log.Printf("🔄 Fetching events for %s (%s)", cur, country)

		if err := FetchAndStoreCountryIndicators(ctx, bunDB, teAPIKEY, country); err != nil {
			log.Printf("❌ failed to ingest %s (%s): %v", cur, country, err)
		} else {
			log.Printf("✅ done ingesting %s (%s)", cur, country)
		}
	}

	log.Println("🎉 Completed ingestion for all currencies.")
}

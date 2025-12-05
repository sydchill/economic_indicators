package main

import (
	"context"
	"economic_indicator/config"
	"economic_indicator/db"
	"economic_indicator/models"
	"log"

	"github.com/uptrace/bun"
)

func main() {
	cfg := config.Load()
	database := db.Open(cfg.DBDSN)

	ctx := context.Background()

	if err := seedCurrencies(ctx, database); err != nil {
		log.Fatalf("seed currencies failed: %v", err)
	}
	if err := seedInstruments(ctx, database); err != nil {
		log.Fatalf("seed instruments failed: %v", err)
	}

	log.Println("✅ Seeding done.")
}

func seedCurrencies(ctx context.Context, database *bun.DB) error {
	currencies := []models.Currency{
		{Code: "USD", Name: "US Dollar"},
		{Code: "GBP", Name: "British Pound"},
		{Code: "EUR", Name: "Euro"},
		{Code: "JPY", Name: "Japanese Yen"},
		{Code: "AUD", Name: "Australian Dollar"},
		{Code: "NZD", Name: "New Zealand Dollar"},
		{Code: "CHF", Name: "Swiss Franc"},
	}

	for _, c := range currencies {
		var existing models.Currency
		err := database.NewSelect().
			Model(&existing).
			Where("code = ?", c.Code).
			Scan(ctx)

		if err == nil {
			log.Printf("Currency %s already exists, skipping", c.Code)
			continue
		}

		// Insert new
		if _, err := database.NewInsert().Model(&c).Exec(ctx); err != nil {
			return err
		}
		log.Printf("Inserted currency %s", c.Code)
	}

	return nil
}

func seedInstruments(ctx context.Context, database *bun.DB) error {
	instruments := []models.Instrument{
		{Symbol: "US500", Name: "S&P 500", AssetType: "index"},
		{Symbol: "US100", Name: "Nasdaq 100", AssetType: "index"},
		{Symbol: "XAUUSD", Name: "Gold", AssetType: "metal"},
		{Symbol: "XAGUSD", Name: "Silver", AssetType: "metal"},
	}

	for _, inst := range instruments {
		var existing models.Instrument
		err := database.NewSelect().
			Model(&existing).
			Where("symbol = ?", inst.Symbol).
			Scan(ctx)

		if err == nil {
			log.Printf("Instrument %s already exists, skipping", inst.Symbol)
			continue
		}

		if _, err := database.NewInsert().Model(&inst).Exec(ctx); err != nil {
			return err
		}
		log.Printf("Inserted instrument %s", inst.Symbol)
	}

	return nil
}

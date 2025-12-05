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

	if err := createSchema(ctx, database); err != nil {
		log.Fatalf("createSchema: %v", err)
	}

	log.Println("✅ Schema created / updated")
}

func createSchema(ctx context.Context, database *bun.DB) error {
	// Order matters if you add foreign keys later: parents first, children after
	modelsToCreate := []interface{}{
		(*models.Currency)(nil),
		(*models.CurrencyScore)(nil),
		(*models.Instrument)(nil),
		(*models.InstrumentScore)(nil),
		(*models.EconIndicator)(nil),
	}

	for _, m := range modelsToCreate {
		_, err := database.NewCreateTable().
			Model(m).
			IfNotExists(). // so it doesn’t crash if already created
			Exec(ctx)
		if err != nil {
			return err
		}
	}

	return nil
}

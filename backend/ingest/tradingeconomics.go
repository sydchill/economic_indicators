package main

import (
	"context"
	"economic_indicator/models"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/uptrace/bun"
)

// TEIndicator struct
type TEIndicator struct {
	Country  string   `json:"Country"`
	Category string   `json:"Category"`
	Value    *float64 `json:"Value"`
	Previous *float64 `json:"Previous"`
	DateTime string   `json:"DateTime"`
}

// FetchAndStoreCountryIndicators fetches indicators for a country
// from the TradingEconomics /country API and upserts them into econ_indicators.
func FetchAndStoreCountryIndicators(
	ctx context.Context,
	db *bun.DB,
	apiKey string,
	country string,
) error {

	url := fmt.Sprintf(
		"https://api.tradingeconomics.com/country/%s?c=%s",
		country, apiKey,
	)

	log.Printf("Fetching TE indicators for %s → %s", country, url)

	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("http error: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("TE status %d", resp.StatusCode)
	}

	var indicators []TEIndicator
	if err := json.NewDecoder(resp.Body).Decode(&indicators); err != nil {
		return fmt.Errorf("decode: %w", err)
	}

	for _, ind := range indicators {
		if err := upsertIndicator(ctx, db, ind); err != nil {
			log.Printf("indicator upsert error: %v", err)
		}
	}

	return nil
}

func upsertIndicator(ctx context.Context, db *bun.DB, ind TEIndicator) error {
	t, _ := time.Parse(time.RFC3339, ind.DateTime)
	raw, _ := json.Marshal(ind)

	var existing models.EconIndicator
	err := db.NewSelect().
		Model(&existing).
		Where("country = ? AND category = ?", ind.Country, ind.Category).
		Scan(ctx)

	if err == nil {
		existing.Value = ind.Value
		existing.Previous = ind.Previous
		existing.DateTime = t
		existing.Raw = raw
		_, err = db.NewUpdate().Model(&existing).WherePK().Exec(ctx)
		return err
	}

	indicator := models.EconIndicator{
		Country:    ind.Country,
		Category:   ind.Category,
		Value:      ind.Value,
		Previous:   ind.Previous,
		DateTime:   t,
		Raw:        raw,
		IngestedAt: time.Now().UTC(),
	}

	_, err = db.NewInsert().Model(&indicator).Exec(ctx)
	return err
}

func abs(f float64) float64 {
	if f < 0 {
		return -f
	}
	return f
}

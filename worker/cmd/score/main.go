package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/joho/godotenv"
	_ "github.com/uptrace/bun/driver/pgdriver"

	"github.com/sydchill/economic_indicators/worker/internal/scoring"
)

type configFile struct {
	Version      int                 `json:"version"`
	LookbackDays int                 `json:"lookback_days"`
	Templates    map[string]template `json:"templates"`
	Assets       map[string]assetCfg `json:"assets"`
}

type template struct {
	CategoryWeights map[string]categoryW `json:"category_weights"`
	Indicators      []indicatorCfg       `json:"indicators"`
}

type categoryW struct {
	Weight float64 `json:"weight"`
}

type indicatorCfg struct {
	Slug      string  `json:"slug"`
	Category  string  `json:"category"`
	Direction int     `json:"direction"`
	Weight    float64 `json:"weight"`
}

type assetCfg struct {
	Template string `json:"template"`
	Name     string `json:"name"`
	Category string `json:"category"`
	Currency string `json:"currency"`
}

func main() {
	configPath := flag.String("config", "seed/scoring_config.json", "path to scoring config")
	dateStr := flag.String("date", "", "scored_at date (YYYY-MM-DD, defaults to today UTC)")
	lookbackOverride := flag.Int("lookback", 0, "override lookback_days from config")
	apply := flag.Bool("apply", false, "write to market.asset_scores")
	verbose := flag.Bool("v", false, "print per-indicator breakdown")
	flag.Parse()

	if err := run(*configPath, *dateStr, *lookbackOverride, *apply, *verbose); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run(configPath, dateStr string, lookbackOverride int, apply, verbose bool) error {
	// 1. Scored-at timestamp
	var scoredAt time.Time
	if dateStr == "" {
		now := time.Now().UTC()
		scoredAt = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	} else {
		t, err := time.Parse("2006-01-02", dateStr)
		if err != nil {
			return fmt.Errorf("--date: %w", err)
		}
		scoredAt = t
	}

	// 2. Config
	b, err := os.ReadFile(configPath)
	if err != nil {
		return err
	}
	var cfg configFile
	if err := json.Unmarshal(b, &cfg); err != nil {
		return fmt.Errorf("config: %w", err)
	}
	lookback := cfg.LookbackDays
	if lookbackOverride > 0 {
		lookback = lookbackOverride
	}
	if lookback <= 0 {
		return fmt.Errorf("lookback_days must be positive (got %d)", lookback)
	}
	cutoff := scoredAt.AddDate(0, 0, -lookback)

	// 3. DB
	_ = godotenv.Load()
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		return fmt.Errorf("DATABASE_URL required")
	}
	db, err := sql.Open("pg", dsn)
	if err != nil {
		return err
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	fmt.Printf("mode=trend signal=sign(change)×100 lookback=%dd asof=%s\n",
		lookback, scoredAt.Format("2006-01-02"))
	fmt.Printf("cutoff: period >= %s\n\n", cutoff.Format("2006-01-02"))

	// 4. Load registry (slug→id)
	slugToID, err := loadRegistry(ctx, db)
	if err != nil {
		return err
	}

	// 5. Score each asset
	results := make([]scoring.AssetScore, 0, len(cfg.Assets))
	assetIDs := map[string]int64{}
	droppedCount := 0
	for code, ac := range cfg.Assets {
		tmpl := cfg.Templates[ac.Template]

		// Load asset_id
		var assetID int64
		if err := db.QueryRowContext(ctx, `SELECT id FROM market.assets WHERE code = $1`, code).Scan(&assetID); err != nil {
			return fmt.Errorf("asset %s not in DB (run seed_scoring_config --apply): %w", code, err)
		}
		assetIDs[code] = assetID

		// Build config slices
		var catWeights []scoring.CategoryWeight
		for cat, cw := range tmpl.CategoryWeights {
			catWeights = append(catWeights, scoring.CategoryWeight{Category: cat, Weight: cw.Weight})
		}
		var indConfigs []scoring.IndicatorConfig
		for _, ic := range tmpl.Indicators {
			id, ok := slugToID[ic.Slug]
			if !ok {
				return fmt.Errorf("asset %s: slug %s not in registry", code, ic.Slug)
			}
			indConfigs = append(indConfigs, scoring.IndicatorConfig{
				IndicatorID: id,
				Slug:        ic.Slug,
				Category:    ic.Category,
				Direction:   ic.Direction,
				Weight:      ic.Weight,
			})
		}

		// Load latest observation per indicator (for this asset's currency)
		obs, err := loadObservations(ctx, db, indConfigs, ac.Currency)
		if err != nil {
			return fmt.Errorf("asset %s: %w", code, err)
		}

		score := scoring.ScoreAsset(code, ac.Currency, catWeights, indConfigs, obs, lookback, scoredAt)
		results = append(results, score)

		for _, cat := range score.Categories {
			droppedCount += cat.TotalIndicators - cat.ActiveIndicators
		}
	}

	// 6. Print summary
	for _, r := range results {
		fmt.Printf("%-8s composite=%+7.2f  bucket=%-12s  currency=%s\n",
			r.AssetCode, r.Composite, r.Bucket, r.Currency)
		for _, cat := range r.Categories {
			if cat.ActiveIndicators == 0 {
				fmt.Printf("    %-14s --- dropped (0/%d active) ---\n", cat.Category, cat.TotalIndicators)
				continue
			}
			fmt.Printf("    %-14s score=%+7.2f  w=%.3f  (%d/%d active)\n",
				cat.Category, cat.Score, cat.WeightRenormalized, cat.ActiveIndicators, cat.TotalIndicators)
			if verbose {
				for _, ind := range cat.Indicators {
					if ind.Mode == scoring.ModeDropped {
						fmt.Printf("        %-35s DROPPED  (%s)\n", ind.Slug, ind.DropReason)
					} else {
						fmt.Printf("        %-35s signal=%+6.1f  value=%v prev=%v\n",
							ind.Slug, ind.Signal, ind.Value, ptr(ind.PreviousValue))
					}
				}
			}
		}
		fmt.Println()
	}

	fmt.Printf("summary: assets=%d lookback=%dd dropped_indicators=%d\n",
		len(results), lookback, droppedCount)

	// 7. Apply
	if !apply {
		fmt.Println("(dry-run — pass --apply to write market.asset_scores)")
		return nil
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	inserted, updated := 0, 0
	for _, r := range results {
		// Build category_scores and category_inputs JSONB
		catScores := map[string]float64{}
		catInputs := map[string]any{}
		for _, cat := range r.Categories {
			catScores[cat.Category] = cat.Score
			catInputs[cat.Category] = cat
		}
		scoresJSON, _ := json.Marshal(catScores)
		inputsJSON, _ := json.Marshal(catInputs)

		var action string
		err := tx.QueryRowContext(ctx, `
			INSERT INTO market.asset_scores
				(asset_id, scored_at, composite_score, bucket, category_scores, category_inputs)
			VALUES ($1, $2, $3, $4, $5, $6)
			ON CONFLICT (asset_id, scored_at) DO UPDATE
			  SET composite_score = EXCLUDED.composite_score,
			      bucket          = EXCLUDED.bucket,
			      category_scores = EXCLUDED.category_scores,
			      category_inputs = EXCLUDED.category_inputs
			RETURNING CASE WHEN xmax = 0 THEN 'inserted' ELSE 'updated' END
		`, assetIDs[r.AssetCode], r.ScoredAt, r.Composite, r.Bucket, string(scoresJSON), string(inputsJSON)).Scan(&action)
		if err != nil {
			return fmt.Errorf("upsert score %s: %w", r.AssetCode, err)
		}
		if action == "inserted" {
			inserted++
		} else {
			updated++
		}
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	fmt.Printf("\napplied: inserted=%d updated=%d\n", inserted, updated)
	return nil
}

// --- helpers ---

func loadRegistry(ctx context.Context, db *sql.DB) (map[string]int64, error) {
	rows, err := db.QueryContext(ctx, `SELECT slug, id FROM market.economic_indicators`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]int64{}
	for rows.Next() {
		var slug string
		var id int64
		if err := rows.Scan(&slug, &id); err != nil {
			return nil, err
		}
		out[slug] = id
	}
	return out, rows.Err()
}
func loadObservations(
	ctx context.Context,
	db *sql.DB,
	cfgs []scoring.IndicatorConfig,
	currency string,
) (map[int64]*scoring.Observation, error) {
	if len(cfgs) == 0 {
		return map[int64]*scoring.Observation{}, nil
	}
	out := map[int64]*scoring.Observation{}
	for _, ic := range cfgs {
		var (
			periodStr      string
			value          float64
			previousValue  sql.NullFloat64
			forecastValue  sql.NullFloat64
			releaseDateStr string
		)
		err := db.QueryRowContext(ctx, `
			SELECT period::text, value, previous_value, forecast_value, release_date::text
			FROM market.economic_data
			WHERE indicator_id = $1 AND currency = $2
			ORDER BY period DESC
			LIMIT 1
		`, ic.IndicatorID, currency).Scan(
			&periodStr, &value, &previousValue, &forecastValue, &releaseDateStr,
		)
		if err == sql.ErrNoRows {
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("indicator %d/%s: %w", ic.IndicatorID, currency, err)
		}
		period, err := time.Parse("2006-01-02", periodStr)
		if err != nil {
			return nil, fmt.Errorf("indicator %d/%s: parse period %q: %w", ic.IndicatorID, currency, periodStr, err)
		}
		releaseDate, err := time.Parse("2006-01-02", releaseDateStr)
		if err != nil {
			return nil, fmt.Errorf("indicator %d/%s: parse release_date %q: %w", ic.IndicatorID, currency, releaseDateStr, err)
		}
		obs := &scoring.Observation{
			Period:      period,
			Value:       value,
			ReleaseDate: releaseDate,
		}
		if previousValue.Valid {
			obs.PreviousValue = &previousValue.Float64
		}
		if forecastValue.Valid {
			obs.ForecastValue = &forecastValue.Float64
		}
		out[ic.IndicatorID] = obs
	}
	return out, nil
}

func ptr(p *float64) any {
	if p == nil {
		return "nil"
	}
	return *p
}

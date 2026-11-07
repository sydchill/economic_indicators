package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"math"
	"os"
	"time"

	"github.com/joho/godotenv"
	_ "github.com/uptrace/bun/driver/pgdriver"
)

// ── JSON shape ────────────────────────────────────────────────────────────

type ConfigFile struct {
	Version   int                    `json:"version"`
	Templates map[string]Template    `json:"templates"`
	Assets    map[string]AssetConfig `json:"assets"`
}

type Template struct {
	CategoryWeights map[string]CategoryWeight `json:"category_weights"`
	Indicators      []IndicatorConfig         `json:"indicators"`
}

type CategoryWeight struct {
	Weight float64 `json:"weight"`
}

type IndicatorConfig struct {
	Slug      string  `json:"slug"`
	Category  string  `json:"category"`
	Direction int     `json:"direction"`
	Weight    float64 `json:"weight"`
}

type AssetConfig struct {
	Template string `json:"template"`
	Name     string `json:"name"`
	Category string `json:"category"`
	Currency string `json:"currency"`
}

// ── Registry ──────────────────────────────────────────────────────────────

type Indicator struct {
	ID       int
	Category string
}

// ── Main ──────────────────────────────────────────────────────────────────

func main() {
	file := flag.String("file", "seed/scoring_config.json", "path to scoring config")
	apply := flag.Bool("apply", false, "write to DB (default: dry-run)")
	reset := flag.Bool("reset", false, "delete existing config rows before applying (implies --apply)")
	flag.Parse()

	if *reset {
		*apply = true
	}

	if err := run(*file, *apply, *reset); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run(path string, apply, reset bool) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	var cfg ConfigFile
	if err := json.Unmarshal(b, &cfg); err != nil {
		return fmt.Errorf("json: %w", err)
	}

	// 1. Pure-shape validation (no DB needed)
	if err := validateShape(&cfg); err != nil {
		return err
	}

	// 2. Connect
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

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 3. Load indicator registry and cross-check slugs
	registry, err := loadIndicators(ctx, db)
	if err != nil {
		return err
	}
	warnings, err := crossCheck(&cfg, registry)
	if err != nil {
		return err
	}

	// 4. Dry-run path
	if !apply {
		printPlan(&cfg)
		fmt.Printf("\nsummary: templates=%d assets=%d indicators_in_registry=%d warnings=%d\n(dry-run — pass --apply or --reset to write)\n",
			len(cfg.Templates), len(cfg.Assets), len(registry), warnings)
		return nil
	}

	// 5. Apply
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if reset {
		fmt.Println("reset: clearing market.asset_indicator_config, market.asset_category_weights")
		if _, err := tx.ExecContext(ctx, `DELETE FROM market.asset_indicator_config`); err != nil {
			return fmt.Errorf("reset indicator config: %w", err)
		}
		if _, err := tx.ExecContext(ctx, `DELETE FROM market.asset_category_weights`); err != nil {
			return fmt.Errorf("reset category weights: %w", err)
		}
	}

	var assetsN, weightsN, indicatorsN int
	for code, ac := range cfg.Assets {
		tmpl := cfg.Templates[ac.Template]

		// Upsert asset (by code), get id back
		var assetID int64
		err := tx.QueryRowContext(ctx, `
			INSERT INTO market.assets (code, name, category, currency)
			VALUES ($1, $2, $3, $4)
			ON CONFLICT (code) DO UPDATE
			  SET name      = EXCLUDED.name,
			      category  = EXCLUDED.category,
			      currency  = EXCLUDED.currency,
			      updated_at = now()
			RETURNING id
		`, code, ac.Name, ac.Category, ac.Currency).Scan(&assetID)
		if err != nil {
			return fmt.Errorf("upsert asset %s: %w", code, err)
		}
		assetsN++

		// Category weights
		for cat, cw := range tmpl.CategoryWeights {
			_, err := tx.ExecContext(ctx, `
				INSERT INTO market.asset_category_weights (asset_id, category, weight)
				VALUES ($1, $2, $3)
				ON CONFLICT (asset_id, category) DO UPDATE
				  SET weight     = EXCLUDED.weight,
				      updated_at = now()
			`, assetID, cat, cw.Weight)
			if err != nil {
				return fmt.Errorf("upsert weight %s/%s: %w", code, cat, err)
			}
			weightsN++
		}

		// Indicator configs
		for _, ic := range tmpl.Indicators {
			reg := registry[ic.Slug]
			_, err := tx.ExecContext(ctx, `
				INSERT INTO market.asset_indicator_config (asset_id, indicator_id, category, direction, weight)
				VALUES ($1, $2, $3, $4, $5)
				ON CONFLICT (asset_id, indicator_id) DO UPDATE
				  SET category   = EXCLUDED.category,
				      direction  = EXCLUDED.direction,
				      weight     = EXCLUDED.weight,
				      updated_at = now()
			`, assetID, reg.ID, ic.Category, ic.Direction, ic.Weight)
			if err != nil {
				return fmt.Errorf("upsert indicator %s/%s: %w", code, ic.Slug, err)
			}
			indicatorsN++
		}
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	fmt.Printf("\napplied: assets=%d category_weights=%d indicator_configs=%d warnings=%d\n",
		assetsN, weightsN, indicatorsN, warnings)
	return nil
}

// ── Validation ────────────────────────────────────────────────────────────

func validateShape(cfg *ConfigFile) error {
	if cfg.Version == 0 {
		return fmt.Errorf("version: required")
	}

	validCategories := map[string]bool{
		"growth": true, "labour": true, "inflation": true,
		"central_bank": true, "sentiment": true, "cot": true,
	}

	for tname, tmpl := range cfg.Templates {
		// Category weights sum to 1.0
		sum := 0.0
		for cat, cw := range tmpl.CategoryWeights {
			if !validCategories[cat] {
				return fmt.Errorf("template %s: invalid category %q", tname, cat)
			}
			sum += cw.Weight
		}
		if math.Abs(sum-1.0) > 0.001 {
			return fmt.Errorf("template %s: category weights sum to %.4f, want 1.0", tname, sum)
		}

		// Per-indicator checks + within-category sum
		seen := map[string]bool{}
		perCategory := map[string]float64{}
		for i, ic := range tmpl.Indicators {
			if ic.Slug == "" {
				return fmt.Errorf("template %s, indicator %d: slug required", tname, i)
			}
			if seen[ic.Slug] {
				return fmt.Errorf("template %s: duplicate slug %q", tname, ic.Slug)
			}
			seen[ic.Slug] = true

			if ic.Direction != 1 && ic.Direction != -1 {
				return fmt.Errorf("template %s slug %s: direction must be ±1, got %d", tname, ic.Slug, ic.Direction)
			}
			if !validCategories[ic.Category] {
				return fmt.Errorf("template %s slug %s: invalid category %q", tname, ic.Slug, ic.Category)
			}
			perCategory[ic.Category] += ic.Weight
		}
		for cat, s := range perCategory {
			if math.Abs(s-1.0) > 0.001 {
				return fmt.Errorf("template %s category %s: indicator weights sum to %.4f, want 1.0", tname, cat, s)
			}
			if _, ok := tmpl.CategoryWeights[cat]; !ok {
				return fmt.Errorf("template %s: indicator category %s not present in category_weights", tname, cat)
			}
		}
	}

	validAssetCategories := map[string]bool{"currency": true, "index": true, "commodity": true, "crypto": true}
	for code, ac := range cfg.Assets {
		if _, ok := cfg.Templates[ac.Template]; !ok {
			return fmt.Errorf("asset %s: unknown template %q", code, ac.Template)
		}
		if ac.Name == "" || ac.Currency == "" {
			return fmt.Errorf("asset %s: name and currency required", code)
		}
		if !validAssetCategories[ac.Category] {
			return fmt.Errorf("asset %s: invalid category %q", code, ac.Category)
		}
	}

	return nil
}

// crossCheck verifies every referenced slug exists in the DB registry.
// Emits a warning (not error) when the config's category differs from the
// registry's native category — intentional in some cases, always worth seeing.
func crossCheck(cfg *ConfigFile, registry map[string]Indicator) (int, error) {
	warnings := 0
	for tname, tmpl := range cfg.Templates {
		for _, ic := range tmpl.Indicators {
			reg, ok := registry[ic.Slug]
			if !ok {
				return 0, fmt.Errorf("template %s: unknown slug %q (run seed-indicators first)", tname, ic.Slug)
			}
			if reg.Category != ic.Category {
				warnings++
				fmt.Printf("WARN  template=%s slug=%s config_category=%s registry_category=%s\n",
					tname, ic.Slug, ic.Category, reg.Category)
			}
		}
	}
	return warnings, nil
}

// ── DB ────────────────────────────────────────────────────────────────────

func loadIndicators(ctx context.Context, db *sql.DB) (map[string]Indicator, error) {
	rows, err := db.QueryContext(ctx, `SELECT id, slug, category FROM market.economic_indicators`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := map[string]Indicator{}
	for rows.Next() {
		var id int
		var slug, cat string
		if err := rows.Scan(&id, &slug, &cat); err != nil {
			return nil, err
		}
		out[slug] = Indicator{ID: id, Category: cat}
	}
	return out, rows.Err()
}

// ── Plan output ───────────────────────────────────────────────────────────

func printPlan(cfg *ConfigFile) {
	fmt.Printf("\ntemplates (%d):\n", len(cfg.Templates))
	for name, t := range cfg.Templates {
		fmt.Printf("  %-12s categories=%d indicators=%d\n", name, len(t.CategoryWeights), len(t.Indicators))
	}

	fmt.Printf("\nassets (%d):\n", len(cfg.Assets))
	totalWeights, totalIndicators := 0, 0
	for code, ac := range cfg.Assets {
		t := cfg.Templates[ac.Template]
		fmt.Printf("  %-8s %-18s category=%-10s currency=%-3s template=%s\n",
			code, ac.Name, ac.Category, ac.Currency, ac.Template)
		totalWeights += len(t.CategoryWeights)
		totalIndicators += len(t.Indicators)
	}
	fmt.Printf("\nprojected rows: assets=%d category_weights=%d indicator_configs=%d\n",
		len(cfg.Assets), totalWeights, totalIndicators)
}

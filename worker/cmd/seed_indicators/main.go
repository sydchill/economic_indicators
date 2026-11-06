package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	_ "github.com/uptrace/bun/driver/pgdriver"
)

type Indicator struct {
	Slug     string `json:"slug"`
	Name     string `json:"name"`
	TEName   string `json:"te_name"`
	Category string `json:"category"`
	Unit     string `json:"unit"`
}

type SeedFile struct {
	Indicators []Indicator `json:"indicators"`
}

func main() {
	path := flag.String("file", "seed/indicators.json", "path to seed file")
	apply := flag.Bool("apply", false, "write to DB")
	flag.Parse()

	b, err := os.ReadFile(*path)
	if err != nil {
		fatal(err)
	}
	var f SeedFile
	if err := json.Unmarshal(b, &f); err != nil {
		fatal(fmt.Errorf("json: %w", err))
	}
	if len(f.Indicators) == 0 {
		fatal(fmt.Errorf("no indicators in %s", *path))
	}

	seen := map[string]struct{}{}
	for i, ind := range f.Indicators {
		if ind.Slug == "" || ind.Name == "" || ind.TEName == "" || ind.Category == "" || ind.Unit == "" {
			fatal(fmt.Errorf("row %d: missing field (slug=%q)", i, ind.Slug))
		}
		if _, dup := seen[ind.Slug]; dup {
			fatal(fmt.Errorf("row %d: duplicate slug %q", i, ind.Slug))
		}
		seen[ind.Slug] = struct{}{}
	}

	if !*apply {
		fmt.Printf("dry-run: %d indicators validated\n", len(f.Indicators))
		for _, ind := range f.Indicators {
			fmt.Printf("  %-45s %-15s %s\n", ind.Slug, ind.Category, ind.Unit)
		}
		return
	}

	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		fatal(fmt.Errorf("DATABASE_URL required"))
	}
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		fatal(err)
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		fatal(err)
	}
	defer tx.Rollback()

	inserted, updated := 0, 0
	for _, ind := range f.Indicators {
		var action string
		err := tx.QueryRowContext(ctx, `
			INSERT INTO market.economic_indicators (slug, name, te_name, category, unit)
			VALUES ($1, $2, $3, $4, $5)
			ON CONFLICT (slug) DO UPDATE
			  SET name      = EXCLUDED.name,
			      te_name   = EXCLUDED.te_name,
			      category  = EXCLUDED.category,
			      unit      = EXCLUDED.unit,
			      updated_at = now()
			RETURNING CASE WHEN xmax = 0 THEN 'inserted' ELSE 'updated' END
		`, ind.Slug, ind.Name, ind.TEName, ind.Category, ind.Unit).Scan(&action)
		if err != nil {
			fatal(fmt.Errorf("%s: %w", ind.Slug, err))
		}
		if action == "inserted" {
			inserted++
		} else {
			updated++
		}
	}
	if err := tx.Commit(); err != nil {
		fatal(err)
	}
	fmt.Printf("applied: inserted=%d updated=%d\n", inserted, updated)
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "error:", err)
	os.Exit(1)
}

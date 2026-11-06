package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"sort"
	"time"

	"github.com/joho/godotenv"
	_ "github.com/uptrace/bun/driver/pgdriver"
)

type PairsFile struct {
	Pairs []Pair `json:"pairs"`
}

type Pair struct {
	Symbol string `json:"symbol"`
	Base   string `json:"base"`
	Quote  string `json:"quote"`
}

var validCurrencies = map[string]bool{
	"USD": true, "EUR": true, "GBP": true, "JPY": true,
	"AUD": true, "NZD": true, "CAD": true, "CHF": true,
}

func main() {
	file := flag.String("file", "seed/pairs.json", "path to pairs file")
	apply := flag.Bool("apply", false, "write to DB")
	flag.Parse()

	if err := run(*file, *apply); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run(path string, apply bool) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var f PairsFile
	if err := json.Unmarshal(b, &f); err != nil {
		return fmt.Errorf("json: %w", err)
	}
	if err := validate(&f); err != nil {
		return err
	}

	sort.Slice(f.Pairs, func(i, j int) bool { return f.Pairs[i].Symbol < f.Pairs[j].Symbol })

	fmt.Printf("validated %d pairs\n", len(f.Pairs))
	for _, p := range f.Pairs {
		fmt.Printf("  %-8s %s/%s\n", p.Symbol, p.Base, p.Quote)
	}

	if !apply {
		fmt.Println("\n(dry-run — pass --apply to write)")
		return nil
	}

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

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	inserted, updated := 0, 0
	for _, p := range f.Pairs {
		var action string
		err := tx.QueryRowContext(ctx, `
			INSERT INTO market.pairs (symbol, base_currency, quote_currency)
			VALUES ($1, $2, $3)
			ON CONFLICT (symbol) DO UPDATE
			  SET base_currency  = EXCLUDED.base_currency,
			      quote_currency = EXCLUDED.quote_currency,
			      updated_at     = now()
			RETURNING CASE WHEN xmax = 0 THEN 'inserted' ELSE 'updated' END
		`, p.Symbol, p.Base, p.Quote).Scan(&action)
		if err != nil {
			return fmt.Errorf("%s: %w", p.Symbol, err)
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

func validate(f *PairsFile) error {
	if len(f.Pairs) != 28 {
		return fmt.Errorf("expected 28 pairs, got %d", len(f.Pairs))
	}
	seenSymbol := map[string]bool{}
	seenCombo := map[string]bool{}
	for i, p := range f.Pairs {
		if p.Symbol == "" || p.Base == "" || p.Quote == "" {
			return fmt.Errorf("row %d: symbol/base/quote required", i)
		}
		if !validCurrencies[p.Base] {
			return fmt.Errorf("row %d (%s): unknown base %q", i, p.Symbol, p.Base)
		}
		if !validCurrencies[p.Quote] {
			return fmt.Errorf("row %d (%s): unknown quote %q", i, p.Symbol, p.Quote)
		}
		if p.Base == p.Quote {
			return fmt.Errorf("row %d (%s): base == quote", i, p.Symbol)
		}
		if p.Symbol != p.Base+p.Quote {
			return fmt.Errorf("row %d: symbol %q != base+quote %q", i, p.Symbol, p.Base+p.Quote)
		}
		if seenSymbol[p.Symbol] {
			return fmt.Errorf("row %d: duplicate symbol %q", i, p.Symbol)
		}
		seenSymbol[p.Symbol] = true

		// Canonical unordered combo — only one direction allowed
		combo := p.Base + "/" + p.Quote
		if p.Base > p.Quote {
			combo = p.Quote + "/" + p.Base
		}
		if seenCombo[combo] {
			return fmt.Errorf("row %d: pair %q already covered in the other direction", i, p.Symbol)
		}
		seenCombo[combo] = true
	}
	return nil
}

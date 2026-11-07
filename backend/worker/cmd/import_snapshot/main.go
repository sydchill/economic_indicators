package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/joho/godotenv"
	_ "github.com/uptrace/bun/driver/pgdriver"

	"github.com/sydchill/economic_indicators/worker/internal/snapshot"
	"github.com/sydchill/economic_indicators/worker/internal/store"
)

func main() {
	file := flag.String("file", "", "path to snapshot JSON (required)")
	apply := flag.Bool("apply", false, "write to DB (default: dry-run)")
	flag.Parse()

	if *file == "" {
		fmt.Fprintln(os.Stderr, "error: --file is required")
		os.Exit(2)
	}

	f, err := snapshot.Load(*file)
	if err != nil {
		fmt.Fprintf(os.Stderr, "parse: %v\n", err)
		os.Exit(1)
	}

	_ = godotenv.Load()
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		fmt.Fprintln(os.Stderr, "error: DATABASE_URL is required")
		os.Exit(2)
	}
	db, err := sql.Open("pg", dsn)
	if err != nil {
		fatal(err)
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	s := store.New(db)
	indicators, err := s.LoadIndicators(ctx)
	if err != nil {
		fatal(err)
	}

	// Validate every slug + unit against the registry before touching data.
	for i, d := range f.EconomicData {
		ind, ok := indicators[d.Slug]
		if !ok {
			fmt.Fprintf(os.Stderr, "row %d: unknown slug %q (run seed-indicators?)\n", i, d.Slug)
			os.Exit(1)
		}
		if ind.Unit != d.Unit {
			fmt.Fprintf(os.Stderr, "row %d: unit mismatch slug=%s file=%q registry=%q\n",
				i, d.Slug, d.Unit, ind.Unit)
			os.Exit(1)
		}
	}

	releaseDate, _ := time.Parse("2006-01-02", f.SnapshotDate)

	// Dry-run or apply.
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		fatal(err)
	}
	if !*apply {
		defer tx.Rollback()
	}

	counts := map[string]int{}
	warnings := 0

	for i, d := range f.EconomicData {
		ind := indicators[d.Slug]
		period, err := snapshot.PeriodDate(d.Period)
		if err != nil {
			fatal(fmt.Errorf("row %d: %w", i, err))
		}

		// Cross-check: file's previous_value vs the DB's prior-period value.
		if d.PreviousValue != nil {
			dbPrev, err := s.PreviousValue(ctx, tx, ind.ID, d.Currency, period)
			if err != nil {
				fatal(fmt.Errorf("row %d: prev lookup: %w", i, err))
			}
			if dbPrev != nil && *dbPrev != *d.PreviousValue {
				warnings++
				fmt.Printf("WARN  %-10s %-45s %s  prev_mismatch db=%v file=%v\n",
					d.Period, d.Slug, d.Currency, *dbPrev, *d.PreviousValue)
			}
		} else {
			fmt.Printf("WARN  %-10s %-45s %s  previous_value missing\n",
				d.Period, d.Slug, d.Currency)
			warnings++
		}

		if !*apply {
			counts["dry-run"]++
			continue
		}

		action, err := s.Upsert(ctx, tx, store.UpsertInput{
			IndicatorID:   ind.ID,
			Currency:      d.Currency,
			Period:        period,
			Value:         d.Value,
			PreviousValue: d.PreviousValue,
			ReleaseDate:   releaseDate,
			Source:        "manual",
		})
		if err != nil {
			tx.Rollback()
			fatal(fmt.Errorf("row %d (%s %s %s): %w", i, d.Slug, d.Currency, d.Period, err))
		}
		counts[action]++
	}

	if *apply {
		if err := tx.Commit(); err != nil {
			fatal(err)
		}
	}

	fmt.Printf("\nsummary: snapshot=%s rows=%d warnings=%d", f.SnapshotDate, len(f.EconomicData), warnings)
	for k, v := range counts {
		fmt.Printf(" %s=%d", k, v)
	}
	fmt.Println()
	if !*apply {
		fmt.Println("(dry-run — pass --apply to write)")
	}
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "error:", err)
	os.Exit(1)
}

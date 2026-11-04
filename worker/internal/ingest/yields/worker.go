package yields

import (
	"context"
	"log"
	"time"

	"github.com/uptrace/bun"
)

type Worker struct {
	db       *bun.DB
	treasury *TreasuryClient
	ecb      *ECBClient
}

func NewWorker(db *bun.DB) *Worker {
	return &Worker{
		db:       db,
		treasury: NewTreasuryClient(),
		ecb:      NewECBClient(),
	}
}

func (w *Worker) RunIngestOnce(ctx context.Context) error {
	now := time.Now().UTC()
	start := now.AddDate(0, 0, -30) // 30-day rolling window

	usRows, err := w.treasury.Fetch(ctx, now.Year())
	if err != nil {
		log.Printf("yields: treasury failed: %v", err)
	} else {
		n, err := w.upsert(ctx, usRows)
		if err != nil {
			log.Printf("yields: treasury upsert failed: %v", err)
		} else {
			log.Printf("yields: treasury upserted %d rows", n)
		}
	}

	euRows, err := w.ecb.Fetch(ctx, start)
	if err != nil {
		log.Printf("yields: ecb failed: %v", err)
	} else {
		n, err := w.upsert(ctx, euRows)
		if err != nil {
			log.Printf("yields: ecb upsert failed: %v", err)
		} else {
			log.Printf("yields: ecb upserted %d rows", n)
		}
	}
	return nil
}

func (w *Worker) upsert(ctx context.Context, rows []YieldRow) (int, error) {
	if len(rows) == 0 {
		return 0, nil
	}
	var count int
	for _, r := range rows {
		_, err := w.db.NewRaw(`
			INSERT INTO market.bond_yields
				(currency_code, maturity_years, record_date, yield_percent, source)
			VALUES (?, ?, ?, ?, ?)
			ON CONFLICT (currency_code, maturity_years, record_date)
			DO UPDATE SET yield_percent = EXCLUDED.yield_percent,
			              source = EXCLUDED.source
		`, r.CurrencyCode, r.MaturityYears, r.RecordDate, r.YieldPercent, r.Source).Exec(ctx)
		if err != nil {
			return count, err
		}
		count++
	}
	return count, nil
}

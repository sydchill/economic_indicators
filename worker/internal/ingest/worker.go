package ingest

import (
	"context"
	"encoding/json"
	"log"

	"github.com/uptrace/bun"
	"golang.org/x/time/rate"

	"github.com/sydchill/economic_indicators/shared/models"
	"github.com/sydchill/economic_indicators/worker/internal/seed"
)

type Worker struct {
	db       *bun.DB
	biquote  *BiquoteClient
	limiter  *rate.Limiter
	mappings []seed.CompiledMapping
}

func NewWorker(db *bun.DB, biquote *BiquoteClient, mappings []seed.CompiledMapping) *Worker {
	return &Worker{
		db:       db,
		biquote:  biquote,
		limiter:  rate.NewLimiter(rate.Every(2e9), 1), // 1 request / 2s
		mappings: mappings,
	}
}

func (w *Worker) RunIngestOnce(ctx context.Context, countries []string) error {
	indicators, err := w.loadIndicatorIndex(ctx)
	if err != nil {
		return err
	}

	inserted, updated, skipped, unmapped := 0, 0, 0, 0

	for _, importance := range []string{"high", "medium"} {
		if err := w.limiter.Wait(ctx); err != nil {
			return err
		}
		events, err := w.biquote.FetchCalendar(ctx, countries, importance)
		if err != nil {
			return err
		}

		for _, ev := range events {
			if ev.Type != "" && ev.Type != "indicator" {
				skipped++
				continue
			}

			suffix, ok := seed.ResolveSuffix(w.mappings, ev.Name)
			if !ok {
				unmapped++
				log.Printf("ingest: unmapped event %q (%s)", ev.Name, ev.Currency)
				continue
			}

			indicatorCode := ev.Currency + "_" + suffix
			indicatorID, ok := indicators[indicatorCode]
			if !ok {
				skipped++
				log.Printf("ingest: no indicator row for %s", indicatorCode)
				continue
			}

			outcome, err := w.upsertEconomicData(ctx, ev, indicatorID)
			if err != nil {
				skipped++
				log.Printf("ingest: upsert %s failed: %v", indicatorCode, err)
				continue
			}
			switch outcome {
			case upsertInserted:
				inserted++
			case upsertUpdated:
				updated++
			}
		}
	}

	log.Printf("econ ingest: inserted=%d updated=%d skipped=%d unmapped=%d",
		inserted, updated, skipped, unmapped)
	return nil
}

type upsertOutcome int

const (
	upsertInserted upsertOutcome = iota
	upsertUpdated
)

func (w *Worker) upsertEconomicData(ctx context.Context, ev BiquoteEvent, indicatorID int) (upsertOutcome, error) {
	raw, err := json.Marshal(ev)
	if err != nil {
		return 0, err
	}

	var xmax int
	err = w.db.NewRaw(`
		INSERT INTO market.economic_data
			(indicator_id, currency_code, release_time, period_date,
			 actual_value, forecast_value, previous_value, revised_value,
			 source, raw_payload)
		VALUES (?, ?, ?, NULL, ?, ?, ?, ?, 'biquote', ?)
		ON CONFLICT (indicator_id, release_time) DO UPDATE SET
			actual_value   = EXCLUDED.actual_value,
			forecast_value = EXCLUDED.forecast_value,
			previous_value = EXCLUDED.previous_value,
			revised_value  = EXCLUDED.revised_value,
			raw_payload    = EXCLUDED.raw_payload
		RETURNING (xmax = 0)::int
	`,
		indicatorID, ev.Currency, ev.Time.UTC(),
		ev.Actual, ev.Forecast, ev.Previous, ev.RevisedPrevious,
		raw,
	).Scan(ctx, &xmax)
	if err != nil {
		return 0, err
	}
	if xmax == 1 {
		return upsertInserted, nil
	}
	return upsertUpdated, nil
}

func (w *Worker) loadIndicatorIndex(ctx context.Context) (map[string]int, error) {
	var rows []models.EconomicIndicator
	if err := w.db.NewSelect().Model(&rows).Scan(ctx); err != nil {
		return nil, err
	}
	idx := make(map[string]int, len(rows))
	for _, r := range rows {
		idx[r.Code] = r.ID
	}
	return idx, nil
}

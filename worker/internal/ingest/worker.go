package ingest

import (
	"context"
	"encoding/json"
	"log"
	"strings"

	"github.com/uptrace/bun"
	"golang.org/x/time/rate"

	"github.com/sydchill/economic_indicators/shared/models"
)

// Worker pulls the economic calendar and upserts events into the market
// schema. It is safe to run repeatedly; the ON CONFLICT clauses make the
// writes idempotent.
type Worker struct {
	db      *bun.DB
	biquote *BiquoteClient
	limiter *rate.Limiter
}

func NewWorker(db *bun.DB, biquote *BiquoteClient) *Worker {
	return &Worker{
		db:      db,
		biquote: biquote,
		// One outbound request every two seconds. Applies per Worker
		// instance; if you run multiple worker replicas, use a shared
		// limiter (e.g. Redis token bucket) to respect the upstream quota.
		limiter: rate.NewLimiter(rate.Every(2e9), 1), // 2e9 ns = 2 s
	}
}

// RunIngestOnce performs a single ingestion pass. Returns an error only
// for failures that affect the whole run (network, decode). Per-event
// errors are logged and skipped so one bad row cannot stall the batch.
func (w *Worker) RunIngestOnce(ctx context.Context, countries []string) error {
	if err := w.limiter.Wait(ctx); err != nil {
		return err
	}

	events, err := w.biquote.FetchCalendar(ctx, countries, "high")
	if err != nil {
		return err
	}

	inserted, updated, skipped := 0, 0, 0
	for _, ev := range events {
		outcome, err := w.upsertEvent(ctx, ev)
		if err != nil {
			log.Printf("ingest skip event %s (%s): %v", ev.EventID, ev.Name, err)
			skipped++
			continue
		}
		switch outcome {
		case upsertInserted:
			inserted++
		case upsertUpdated:
			updated++
		}
	}

	log.Printf("ingest done: inserted=%d updated=%d skipped=%d total=%d",
		inserted, updated, skipped, len(events))
	return nil
}

type upsertOutcome int

const (
	upsertInserted upsertOutcome = iota
	upsertUpdated
)

func (w *Worker) upsertEvent(ctx context.Context, ev BiquoteEvent) (upsertOutcome, error) {
	if ev.Currency == "" || ev.Name == "" || ev.Time.IsZero() {
		return 0, errIncompleteEvent
	}

	// Guard: skip events for currencies we do not track.
	exists, err := w.db.NewSelect().
		Model((*models.Currency)(nil)).
		Where("c.code = ?", ev.Currency).
		Exists(ctx)
	if err != nil {
		return 0, err
	}
	if !exists {
		return 0, errUnknownCurrency
	}

	ind := &models.EconomicIndicator{
		Code:         indicatorCode(ev.Currency, ev.Name),
		Name:         ev.Name,
		CurrencyCode: ev.Currency,
		Importance:   normalizeImportance(ev.Importance),
		Unit:         ev.Unit,
	}
	if _, err := w.db.NewInsert().Model(ind).
		On("CONFLICT (code) DO UPDATE").
		Set("name = EXCLUDED.name").
		Set("importance = EXCLUDED.importance").
		Set("unit = EXCLUDED.unit").
		Returning("id").
		Exec(ctx); err != nil {
		return 0, err
	}

	raw, err := json.Marshal(ev)
	if err != nil {
		return 0, err
	}

	data := &models.EconomicData{
		IndicatorID:   ind.ID,
		CurrencyCode:  ev.Currency,
		ReleaseTime:   ev.Time.UTC(),
		ActualValue:   ev.Actual,
		ForecastValue: ev.Forecast,
		PreviousValue: ev.Previous,
		RevisedValue:  ev.Revised,
		Source:        "biquote",
		RawPayload:    raw,
	}

	// xmax = 0 distinguishes insert from update on Postgres.
	var xmax int
	err = w.db.NewRaw(`
		INSERT INTO market.economic_data
			(indicator_id, currency_code, release_time,
			 actual_value, forecast_value, previous_value, revised_value,
			 source, raw_payload)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT (indicator_id, release_time) DO UPDATE SET
			actual_value   = EXCLUDED.actual_value,
			forecast_value = EXCLUDED.forecast_value,
			previous_value = EXCLUDED.previous_value,
			revised_value  = EXCLUDED.revised_value,
			raw_payload    = EXCLUDED.raw_payload
		RETURNING (xmax = 0)::int
	`,
		data.IndicatorID, data.CurrencyCode, data.ReleaseTime,
		data.ActualValue, data.ForecastValue, data.PreviousValue, data.RevisedValue,
		data.Source, data.RawPayload,
	).Scan(ctx, &xmax)
	if err != nil {
		return 0, err
	}

	if xmax == 1 {
		return upsertInserted, nil
	}
	return upsertUpdated, nil
}

func indicatorCode(currency, name string) string {
	slug := strings.ToUpper(strings.ReplaceAll(strings.TrimSpace(name), " ", "_"))
	return currency + "_" + slug
}

func normalizeImportance(v string) string {
	switch strings.ToLower(v) {
	case "high":
		return "high"
	case "medium":
		return "medium"
	default:
		return "low"
	}
}

// internal sentinel errors; log-only, never returned to callers of RunIngestOnce.
var (
	errIncompleteEvent = sentinel("incomplete event")
	errUnknownCurrency = sentinel("currency not tracked")
)

type sentinel string

func (s sentinel) Error() string { return string(s) }

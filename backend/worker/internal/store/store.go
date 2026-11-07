package store

import (
	"context"
	"database/sql"
	"time"
)

type Store struct{ db *sql.DB }

func New(db *sql.DB) *Store { return &Store{db: db} }

// Indicator is the subset of market.economic_indicators the importer cares about.
type Indicator struct {
	ID       int
	Slug     string
	Unit     string
	Category string
}

// LoadIndicators returns every row keyed by slug. 27 rows — no need for filtering.
func (s *Store) LoadIndicators(ctx context.Context) (map[string]Indicator, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, slug, unit, category
		FROM market.economic_indicators
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make(map[string]Indicator, 32)
	for rows.Next() {
		var i Indicator
		if err := rows.Scan(&i.ID, &i.Slug, &i.Unit, &i.Category); err != nil {
			return nil, err
		}
		out[i.Slug] = i
	}
	return out, rows.Err()
}

// PreviousValue returns the most recent recorded value for (indicator, currency)
// strictly before period. Used only for the cross-check log.
func (s *Store) PreviousValue(ctx context.Context, q interface {
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}, indicatorID int, currency string, period time.Time) (*float64, error) {
	var v sql.NullFloat64
	err := q.QueryRowContext(ctx, `
		SELECT value
		FROM market.economic_data
		WHERE indicator_id = $1 AND currency = $2 AND period < $3
		ORDER BY period DESC
		LIMIT 1
	`, indicatorID, currency, period).Scan(&v)
	if err == sql.ErrNoRows || !v.Valid {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &v.Float64, nil
}

// Upsert inserts or updates the row for (indicator, currency, period).
// Returns one of: "inserted", "updated", "unchanged".
func (s *Store) Upsert(ctx context.Context, tx *sql.Tx, in UpsertInput) (string, error) {
	var existing sql.NullFloat64
	err := tx.QueryRowContext(ctx, `
		SELECT value
		FROM market.economic_data
		WHERE indicator_id = $1 AND currency = $2 AND period = $3
	`, in.IndicatorID, in.Currency, in.Period).Scan(&existing)

	switch {
	case err == sql.ErrNoRows:
		_, err := tx.ExecContext(ctx, `
			INSERT INTO market.economic_data
				(indicator_id, currency, period, value, previous_value, forecast_value, release_date, source)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		`, in.IndicatorID, in.Currency, in.Period, in.Value,
			nullFloat(in.PreviousValue), nullFloat(in.ForecastValue),
			in.ReleaseDate, in.Source)
		return "inserted", err
	case err != nil:
		return "", err
	}

	if existing.Valid && existing.Float64 == in.Value {
		// Value unchanged — but previous_value may have been revised.
		_, err := tx.ExecContext(ctx, `
			UPDATE market.economic_data
			SET previous_value = $4, updated_at = now()
			WHERE indicator_id = $1 AND currency = $2 AND period = $3
		`, in.IndicatorID, in.Currency, in.Period, nullFloat(in.PreviousValue))
		return "unchanged", err
	}

	_, err = tx.ExecContext(ctx, `
		UPDATE market.economic_data
		SET value = $4, previous_value = $5, forecast_value = $6,
		    release_date = $7, source = $8, updated_at = now()
		WHERE indicator_id = $1 AND currency = $2 AND period = $3
	`, in.IndicatorID, in.Currency, in.Period, in.Value,
		nullFloat(in.PreviousValue), nullFloat(in.ForecastValue),
		in.ReleaseDate, in.Source)
	return "updated", err
}

type UpsertInput struct {
	IndicatorID   int
	Currency      string
	Period        time.Time
	Value         float64
	PreviousValue *float64
	ForecastValue *float64
	ReleaseDate   time.Time
	Source        string
}

func nullFloat(p *float64) any {
	if p == nil {
		return nil
	}
	return *p
}

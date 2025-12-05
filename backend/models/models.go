package models

import (
	"time"

	"github.com/uptrace/bun"
)

// Currency Currency
type Currency struct {
	bun.BaseModel `bun:"table:currencies"`

	ID        int64     `bun:",pk,autoincrement"`
	Code      string    `bun:",unique,notnull"`
	Name      string    `bun:",nullzero"`
	CreatedAt time.Time `bun:",nullzero,notnull,default:current_timestamp"`
}

// CurrencyScore CurrencyScore
type CurrencyScore struct {
	bun.BaseModel `bun:"table:currency_scores"`

	ID         int64     `bun:",pk,autoincrement"`
	CurrencyID int64     `bun:",notnull"`
	TS         time.Time `bun:",notnull"`
	EconScore  float64   `bun:",nullzero"`
	CreatedAt  time.Time `bun:",nullzero,notnull,default:current_timestamp"`
}

// Instrument Instrument
type Instrument struct {
	bun.BaseModel `bun:"table:instruments"`

	ID        int64     `bun:",pk,autoincrement"`
	Symbol    string    `bun:",unique,notnull"`
	Name      string    `bun:",nullzero"`
	AssetType string    `bun:",nullzero"` // "index", "metal", "fx"
	CreatedAt time.Time `bun:",nullzero,notnull,default:current_timestamp"`
}

// InstrumentScore InstrumentScore
type InstrumentScore struct {
	bun.BaseModel `bun:"table:instrument_scores"`

	ID           int64     `bun:",pk,autoincrement"`
	InstrumentID int64     `bun:",notnull"`
	TS           time.Time `bun:",notnull"`
	FinalScore   float64   `bun:",nullzero"`
	CreatedAt    time.Time `bun:",nullzero,notnull,default:current_timestamp"`
}

// EconIndicator table
type EconIndicator struct {
	bun.BaseModel `bun:"table:econ_indicators"`

	ID         int64     `bun:",pk,autoincrement"`
	Country    string    `bun:"country"`
	Category   string    `bun:"category"`
	Value      *float64  `bun:"value"`
	Previous   *float64  `bun:"previous"`
	DateTime   time.Time `bun:"datetime"`
	Raw        []byte    `bun:"raw"`
	IngestedAt time.Time `bun:"ingested_at,notnull,default:current_timestamp"`
}

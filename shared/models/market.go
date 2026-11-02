// shared/models/market.go
package models

import (
	"encoding/json"
	"time"

	"github.com/uptrace/bun"
)

type Currency struct {
	bun.BaseModel `bun:"table:market.currencies,alias:c"`
	Code          string `bun:"code,pk"`
	Name          string `bun:"name,notnull"`
	IsMajor       bool   `bun:"is_major,notnull,default:false"`
}

type EconomicIndicator struct {
	bun.BaseModel `bun:"table:market.economic_indicators,alias:i"`
	ID            int       `bun:"id,pk,autoincrement"`
	Code          string    `bun:"code,notnull,unique"`
	Name          string    `bun:"name,notnull"`
	CurrencyCode  string    `bun:"currency_code,notnull"`
	Importance    string    `bun:"importance,notnull"`
	Unit          string    `bun:"unit"`
	CreatedAt     time.Time `bun:"created_at,notnull,default:now()"`
}

type EconomicData struct {
	bun.BaseModel `bun:"table:market.economic_data,alias:d"`
	ID            int64           `bun:"id,pk,autoincrement"`
	IndicatorID   int             `bun:"indicator_id,notnull"`
	CurrencyCode  string          `bun:"currency_code,notnull"`
	ReleaseTime   time.Time       `bun:"release_time,notnull"`
	ActualValue   *float64        `bun:"actual_value"`
	ForecastValue *float64        `bun:"forecast_value"`
	PreviousValue *float64        `bun:"previous_value"`
	RevisedValue  *float64        `bun:"revised_value"`
	Source        string          `bun:"source,notnull,default:'biquote'"`
	RawPayload    json.RawMessage `bun:"raw_payload,type:jsonb"`
	CreatedAt     time.Time       `bun:"created_at,notnull,default:now()"`
}

type BondYield struct {
	bun.BaseModel `bun:"table:market.bond_yields,alias:y"`
	ID            int64     `bun:"id,pk,autoincrement"`
	CurrencyCode  string    `bun:"currency_code,notnull"`
	MaturityYears int16     `bun:"maturity_years,notnull"`
	RecordDate    time.Time `bun:"record_date,notnull"`
	YieldPercent  float64   `bun:"yield_percent,notnull"`
	Source        string    `bun:"source,notnull,default:'fxmacrodata'"`
	CreatedAt     time.Time `bun:"created_at,notnull,default:now()"`
}

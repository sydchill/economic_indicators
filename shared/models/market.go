package models

import (
	"encoding/json"
	"time"

	"github.com/uptrace/bun"
)

type Asset struct {
	bun.BaseModel `bun:"table:market.assets,alias:a"`

	Code            string    `bun:"code,pk" json:"code"`
	Name            string    `bun:"name,notnull" json:"name"`
	DisplayName     string    `bun:"display_name,notnull" json:"display_name"`
	AssetClass      string    `bun:"asset_class,notnull" json:"asset_class"`
	PrimaryCurrency *string   `bun:"primary_currency" json:"primary_currency,omitempty"`
	IsMajor         bool      `bun:"is_major,notnull,default:false" json:"is_major"`
	CreatedAt       time.Time `bun:"created_at,notnull,default:now()" json:"created_at"`
}

type Pair struct {
	bun.BaseModel `bun:"table:market.pairs,alias:p"`

	Code          string    `bun:"code,pk" json:"code"`
	BaseCurrency  string    `bun:"base_currency,notnull" json:"base_currency"`
	QuoteCurrency string    `bun:"quote_currency,notnull" json:"quote_currency"`
	DisplayName   string    `bun:"display_name,notnull" json:"display_name"`
	IsMajor       bool      `bun:"is_major,notnull,default:false" json:"is_major"`
	CreatedAt     time.Time `bun:"created_at,notnull,default:now()" json:"created_at"`
}

type EconomicIndicator struct {
	bun.BaseModel `bun:"table:market.economic_indicators,alias:i"`

	ID           int       `bun:"id,pk,autoincrement" json:"id"`
	Code         string    `bun:"code,notnull,unique" json:"code"`
	Name         string    `bun:"name,notnull" json:"name"`
	CurrencyCode string    `bun:"currency_code,notnull" json:"currency_code"`
	Category     string    `bun:"category,notnull" json:"category"`
	Importance   string    `bun:"importance,notnull" json:"importance"`
	Unit         string    `bun:"unit" json:"unit,omitempty"`
	CreatedAt    time.Time `bun:"created_at,notnull,default:now()" json:"created_at"`
}

type EconomicData struct {
	bun.BaseModel `bun:"table:market.economic_data,alias:d"`

	ID            int64           `bun:"id,pk,autoincrement" json:"id"`
	IndicatorID   int             `bun:"indicator_id,notnull" json:"indicator_id"`
	CurrencyCode  string          `bun:"currency_code,notnull" json:"currency_code"`
	ReleaseTime   time.Time       `bun:"release_time,notnull" json:"release_time"`
	PeriodDate    *time.Time      `bun:"period_date,type:date" json:"period_date,omitempty"`
	ActualValue   *float64        `bun:"actual_value" json:"actual_value,omitempty"`
	ForecastValue *float64        `bun:"forecast_value" json:"forecast_value,omitempty"`
	PreviousValue *float64        `bun:"previous_value" json:"previous_value,omitempty"`
	RevisedValue  *float64        `bun:"revised_value" json:"revised_value,omitempty"`
	Surprise      *float64        `bun:"surprise" json:"surprise,omitempty"`
	Source        string          `bun:"source,notnull,default:'biquote'" json:"source"`
	RawPayload    json.RawMessage `bun:"raw_payload,type:jsonb" json:"-"`
	CreatedAt     time.Time       `bun:"created_at,notnull,default:now()" json:"created_at"`
}

type BondYield struct {
	bun.BaseModel `bun:"table:market.bond_yields,alias:y"`

	ID            int64     `bun:"id,pk,autoincrement" json:"id"`
	CurrencyCode  string    `bun:"currency_code,notnull" json:"currency_code"`
	MaturityYears int16     `bun:"maturity_years,notnull" json:"maturity_years"`
	RecordDate    time.Time `bun:"record_date,type:date,notnull" json:"record_date"`
	YieldPercent  float64   `bun:"yield_percent,notnull" json:"yield_percent"`
	Source        string    `bun:"source,notnull,default:'fxmacrodata'" json:"source"`
	CreatedAt     time.Time `bun:"created_at,notnull,default:now()" json:"created_at"`
}

type COTData struct {
	bun.BaseModel `bun:"table:market.cot_data,alias:c"`

	ID             int64           `bun:"id,pk,autoincrement" json:"id"`
	AssetID        string          `bun:"asset_id,notnull" json:"asset_id"`
	ReportDate     time.Time       `bun:"report_date,type:date,notnull" json:"report_date"`
	ReleasedAt     time.Time       `bun:"released_at,type:date,notnull" json:"released_at"`
	LongContracts  int             `bun:"long_contracts,notnull" json:"long_contracts"`
	ShortContracts int             `bun:"short_contracts,notnull" json:"short_contracts"`
	NetPosition    int             `bun:"net_position,notnull" json:"net_position"`
	WeeklyChange   *int            `bun:"weekly_change" json:"weekly_change,omitempty"`
	OpenInterest   *int            `bun:"open_interest" json:"open_interest,omitempty"`
	Source         string          `bun:"source,notnull,default:'cftc'" json:"source"`
	RawPayload     json.RawMessage `bun:"raw_payload,type:jsonb" json:"-"`
	CreatedAt      time.Time       `bun:"created_at,notnull,default:now()" json:"created_at"`
}

type RetailSentiment struct {
	bun.BaseModel `bun:"table:market.retail_sentiment,alias:rs"`

	ID         int64     `bun:"id,pk,autoincrement" json:"id"`
	PairCode   string    `bun:"pair_code,notnull" json:"pair_code"`
	Timeframe  string    `bun:"timeframe,notnull" json:"timeframe"`
	LongPct    float64   `bun:"long_pct,notnull" json:"long_pct"`
	ShortPct   float64   `bun:"short_pct,notnull" json:"short_pct"`
	Source     string    `bun:"source,notnull" json:"source"`
	SnapshotAt time.Time `bun:"snapshot_at,notnull" json:"snapshot_at"`
	CreatedAt  time.Time `bun:"created_at,notnull,default:now()" json:"created_at"`
}

type AssetCategoryWeight struct {
	bun.BaseModel `bun:"table:market.asset_category_weights,alias:acw"`

	AssetID  string          `bun:"asset_id,pk" json:"asset_id"`
	Category string          `bun:"category,pk" json:"category"`
	Weight   float64         `bun:"weight,notnull" json:"weight"`
	Params   json.RawMessage `bun:"params,type:jsonb,default:'{}'::jsonb" json:"params,omitempty"`
}

type AssetIndicatorConfig struct {
	bun.BaseModel `bun:"table:market.asset_indicator_config,alias:aic"`

	AssetID     string  `bun:"asset_id,pk" json:"asset_id"`
	IndicatorID int     `bun:"indicator_id,pk" json:"indicator_id"`
	Category    string  `bun:"category,notnull" json:"category"`
	Direction   int16   `bun:"direction,notnull" json:"direction"`
	Weight      float64 `bun:"weight,notnull" json:"weight"`
}

type AssetScore struct {
	bun.BaseModel `bun:"table:market.asset_scores,alias:s"`

	AssetID        string          `bun:"asset_id,pk" json:"asset_id"`
	ComputedAt     time.Time       `bun:"computed_at,pk" json:"computed_at"`
	CompositeScore float64         `bun:"composite_score,notnull" json:"composite_score"`
	Signal         string          `bun:"signal,notnull" json:"signal"`
	CategoryScores json.RawMessage `bun:"category_scores,type:jsonb,notnull" json:"category_scores"`
	CategoryInputs json.RawMessage `bun:"category_inputs,type:jsonb,notnull" json:"category_inputs"`
}

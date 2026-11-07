package models

import "time"

// ─── Assets & Pairs ────────────────────────────────────────────────────────

// Asset is a tradeable instrument that the scoring engine produces a score
// for. Currency assets represent a single currency (USD, EUR, …); non-currency
// assets (JP225, XAU, …) carry a Currency field linking them to the currency
// whose economy drives them.
type Asset struct {
	ID        int64     `bun:"id,pk,autoincrement"                            json:"id"`
	Code      string    `bun:"code,unique,notnull"                            json:"code"`
	Name      string    `bun:"name,notnull"                                   json:"name"`
	Category  string    `bun:"category,notnull"                               json:"category"`
	Currency  string    `bun:"currency"                                       json:"currency,omitempty"`
	CreatedAt time.Time `bun:"created_at,nullzero,notnull,default:current_timestamp" json:"created_at"`
	UpdatedAt time.Time `bun:"updated_at,nullzero,notnull,default:current_timestamp" json:"updated_at"`
}

// Pair is a currency pair. Symbol is the concatenation (EURUSD); Base/Quote
// are the ISO codes that let the scoring engine compute differential scores.
type Pair struct {
	ID            int64     `bun:"id,pk,autoincrement"                            json:"id"`
	Symbol        string    `bun:"symbol,unique,notnull"                          json:"symbol"`
	BaseCurrency  string    `bun:"base_currency,notnull"                          json:"base_currency"`
	QuoteCurrency string    `bun:"quote_currency,notnull"                         json:"quote_currency"`
	CreatedAt     time.Time `bun:"created_at,nullzero,notnull,default:current_timestamp" json:"created_at"`
	UpdatedAt     time.Time `bun:"updated_at,nullzero,notnull,default:current_timestamp" json:"updated_at"`
}

// ─── Economic Indicators & Data ────────────────────────────────────────────

// EconomicIndicator is the registry of every indicator the system knows about.
// Slug is the wire format (used in snapshot files); TEName is the exact
// TradingEconomics label, kept for audit and for the eventual API swap.
type EconomicIndicator struct {
	ID        int64     `bun:"id,pk,autoincrement"                            json:"id"`
	Slug      string    `bun:"slug,unique,notnull"                            json:"slug"`
	Name      string    `bun:"name,notnull"                                   json:"name"`
	TEName    string    `bun:"te_name,notnull"                                json:"te_name"`
	Category  string    `bun:"category,notnull"                               json:"category"`
	Unit      string    `bun:"unit,notnull"                                   json:"unit"`
	CreatedAt time.Time `bun:"created_at,nullzero,notnull,default:current_timestamp" json:"created_at"`
	UpdatedAt time.Time `bun:"updated_at,nullzero,notnull,default:current_timestamp" json:"updated_at"`
}

// EconomicData is one reading for one indicator on one currency for one period.
// previous_value is the release's "Previous" column — the revised prior reading,
// not the originally reported one. forecast_value is NULL for manual imports and
// will be populated when the TE API lands.
type EconomicData struct {
	ID            int64     `bun:"id,pk,autoincrement"                            json:"id"`
	IndicatorID   int64     `bun:"indicator_id,notnull"                           json:"indicator_id"`
	Currency      string    `bun:"currency,notnull"                               json:"currency"`
	Period        time.Time `bun:"period,notnull"                                 json:"period"`
	Value         float64   `bun:"value,notnull"                                  json:"value"`
	PreviousValue *float64  `bun:"previous_value"                                 json:"previous_value,omitempty"`
	ForecastValue *float64  `bun:"forecast_value"                                 json:"forecast_value,omitempty"`
	ReleaseDate   time.Time `bun:"release_date,notnull"                           json:"release_date"`
	Source        string    `bun:"source,notnull"                                 json:"source"`
	CreatedAt     time.Time `bun:"created_at,nullzero,notnull,default:current_timestamp" json:"created_at"`
	UpdatedAt     time.Time `bun:"updated_at,nullzero,notnull,default:current_timestamp" json:"updated_at"`

	Indicator *EconomicIndicator `bun:"rel:belongs-to,join:indicator_id=id" json:"indicator,omitempty"`
}

// ─── Scoring Configuration ─────────────────────────────────────────────────

// AssetCategoryWeight is the configured weight of a scoring category for one
// asset. The set of weights for an asset must sum to 1.0 (the scoring engine
// renormalizes when categories drop out, but the seed data should be balanced).
type AssetCategoryWeight struct {
	ID        int64     `bun:"id,pk,autoincrement"                            json:"id"`
	AssetID   int64     `bun:"asset_id,notnull"                               json:"asset_id"`
	Category  string    `bun:"category,notnull"                               json:"category"`
	Weight    float64   `bun:"weight,notnull"                                 json:"weight"`
	CreatedAt time.Time `bun:"created_at,nullzero,notnull,default:current_timestamp" json:"created_at"`
	UpdatedAt time.Time `bun:"updated_at,nullzero,notnull,default:current_timestamp" json:"updated_at"`

	Asset *Asset `bun:"rel:belongs-to,join:asset_id=id" json:"asset,omitempty"`
}

// AssetIndicatorConfig is the per-asset configuration of one indicator:
// Direction is +1 when rising values are bullish for the asset, -1 when
// bearish. Weight is the indicator's share within its category.
type AssetIndicatorConfig struct {
	ID          int64     `bun:"id,pk,autoincrement"                            json:"id"`
	AssetID     int64     `bun:"asset_id,notnull"                               json:"asset_id"`
	IndicatorID int64     `bun:"indicator_id,notnull"                           json:"indicator_id"`
	Category    string    `bun:"category,notnull"                               json:"category"` // NEW
	Direction   int       `bun:"direction,notnull"                              json:"direction"`
	Weight      float64   `bun:"weight,notnull"                                 json:"weight"`
	CreatedAt   time.Time `bun:"created_at,nullzero,notnull,default:current_timestamp" json:"created_at"`
	UpdatedAt   time.Time `bun:"updated_at,nullzero,notnull,default:current_timestamp" json:"updated_at"`

	Asset     *Asset             `bun:"rel:belongs-to,join:asset_id=id"     json:"asset,omitempty"`
	Indicator *EconomicIndicator `bun:"rel:belongs-to,join:indicator_id=id" json:"indicator,omitempty"`
}

// ─── Scoring Output ────────────────────────────────────────────────────────

// Bucket is the categorical label derived from CompositeScore. Thresholds:
// ≥ +60 VERY_BULLISH, ≥ 0 BULLISH, > -60 BEARISH, else VERY_BEARISH.
const (
	BucketVeryBullish = "VERY_BULLISH"
	BucketBullish     = "BULLISH"
	BucketBearish     = "BEARISH"
	BucketVeryBearish = "VERY_BEARISH"
)

// AssetScore is a computed score for one asset at one point in time.
// CategoryScores holds the score for each active category (0-100 scale);
// CategoryInputs holds the drill-down — per-indicator raw values, surprises,
// directions, and weights that produced the category score.
type AssetScore struct {
	ID             int64              `bun:"id,pk,autoincrement"                            json:"id"`
	AssetID        int64              `bun:"asset_id,notnull"                               json:"asset_id"`
	ScoredAt       time.Time          `bun:"scored_at,notnull"                              json:"scored_at"`
	CompositeScore float64            `bun:"composite_score,notnull"                        json:"composite_score"`
	Bucket         string             `bun:"bucket,notnull"                                 json:"bucket"`
	CategoryScores map[string]float64 `bun:"category_scores,type:jsonb,notnull"             json:"category_scores"`
	CategoryInputs map[string]any     `bun:"category_inputs,type:jsonb,notnull"             json:"category_inputs"`
	CreatedAt      time.Time          `bun:"created_at,nullzero,notnull,default:current_timestamp" json:"created_at"`

	Asset *Asset `bun:"rel:belongs-to,join:asset_id=id" json:"asset,omitempty"`
}

// ─── COT (populated later via CFTC CSV) ────────────────────────────────────

// COTData is one CFTC commitment-of-traders row per (currency, report_date).
// NoncommercialNet = Long - Short. Populated by a future CSV importer.
type COTData struct {
	ID                 int64     `bun:"id,pk,autoincrement"                            json:"id"`
	Currency           string    `bun:"currency,notnull"                               json:"currency"`
	ReportDate         time.Time `bun:"report_date,notnull"                            json:"report_date"`
	NoncommercialLong  int64     `bun:"noncommercial_long,notnull"                     json:"noncommercial_long"`
	NoncommercialShort int64     `bun:"noncommercial_short,notnull"                    json:"noncommercial_short"`
	NoncommercialNet   int64     `bun:"noncommercial_net,notnull"                      json:"noncommercial_net"`
	Source             string    `bun:"source,notnull"                                 json:"source"`
	CreatedAt          time.Time `bun:"created_at,nullzero,notnull,default:current_timestamp" json:"created_at"`
	UpdatedAt          time.Time `bun:"updated_at,nullzero,notnull,default:current_timestamp" json:"updated_at"`
}

// RetailSentiment is one retail long/short snapshot for a pair. Populated
// manually for now; source-of-truth TBD.
type RetailSentiment struct {
	ID         int64     `bun:"id,pk,autoincrement"                            json:"id"`
	PairID     int64     `bun:"pair_id,notnull"                                json:"pair_id"`
	SnapshotAt time.Time `bun:"snapshot_at,notnull"                            json:"snapshot_at"`
	LongPct    float64   `bun:"long_pct,notnull"                               json:"long_pct"`
	ShortPct   float64   `bun:"short_pct,notnull"                              json:"short_pct"`
	Source     string    `bun:"source,notnull"                                 json:"source"`
	CreatedAt  time.Time `bun:"created_at,nullzero,notnull,default:current_timestamp" json:"created_at"`
	UpdatedAt  time.Time `bun:"updated_at,nullzero,notnull,default:current_timestamp" json:"updated_at"`

	Pair *Pair `bun:"rel:belongs-to,join:pair_id=id" json:"pair,omitempty"`
}

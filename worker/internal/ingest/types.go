package ingest

import "encoding/json"

// ConfigFile is the top-level JSON shape of seed/scoring_config.json.
type ConfigFile struct {
	Version   int                   `json:"version"`
	Templates map[string]Template   `json:"templates"`
	Assets    map[string]AssetEntry `json:"assets"`
}

type Template struct {
	CategoryWeights map[string]CategoryWeight `json:"category_weights"`
	Indicators      map[string][]IndicatorDef `json:"indicators"`
}

type CategoryWeight struct {
	Weight float64         `json:"weight"`
	Params json.RawMessage `json:"params,omitempty"`
}

type IndicatorDef struct {
	Suffix    string  `json:"suffix"`
	Direction int16   `json:"direction"`
	Weight    float64 `json:"weight"`
}

type AssetEntry struct {
	Template string    `json:"template"`
	Currency string    `json:"currency"`
	Override *Template `json:"override,omitempty"`
}

// Plan is the fully-resolved, DB-ready form of the config file.
type Plan struct {
	Assets []AssetPlan
}

type AssetPlan struct {
	AssetID         string
	CategoryWeights []CategoryWeightRow
	Indicators      []IndicatorRow
}

type CategoryWeightRow struct {
	AssetID  string
	Category string
	Weight   float64
	Params   json.RawMessage
}

type IndicatorRow struct {
	AssetID       string
	Category      string
	IndicatorCode string // e.g. "USD_CPI_YOY"
	IndicatorID   int    // resolved from DB during validation
	Direction     int16
	Weight        float64
}

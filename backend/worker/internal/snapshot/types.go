package snapshot

// File is the on-disk wire format for a daily snapshot.
type File struct {
	SnapshotDate string          `json:"snapshot_date"` // YYYY-MM-DD
	EconomicData []EconomicDatum `json:"economic_data"`
}

// EconomicDatum is one indicator reading for one currency on one period.
type EconomicDatum struct {
	Slug          string   `json:"slug"`
	Currency      string   `json:"currency"` // ISO 4217
	Value         float64  `json:"value"`
	PreviousValue *float64 `json:"previous_value"` // nil = first observation
	Unit          string   `json:"unit"`           // asserted against registry
	Period        string   `json:"period"`         // YYYY-MM
}

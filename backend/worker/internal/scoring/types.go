package scoring

import "time"

// Input types ---------------------------------------------------------------

type CategoryWeight struct {
	Category string
	Weight   float64
}

type IndicatorConfig struct {
	IndicatorID int64
	Slug        string
	Category    string
	Direction   int
	Weight      float64
}

// Observation is the latest economic_data row for one (indicator, currency).
type Observation struct {
	Period        time.Time
	Value         float64
	PreviousValue *float64
	ForecastValue *float64
	ReleaseDate   time.Time
	IsCOT         bool
}

// Output types --------------------------------------------------------------

const (
	ModeTrend    = "trend"
	ModeSurprise = "surprise"
	ModeDropped  = "dropped"
	ModeCOT      = "cot"
)

type IndicatorSignal struct {
	Slug          string    `json:"slug"`
	Period        time.Time `json:"period"`
	Value         float64   `json:"value"`
	PreviousValue *float64  `json:"previous_value,omitempty"`
	ForecastValue *float64  `json:"forecast_value,omitempty"`
	RawChange     *float64  `json:"raw_change,omitempty"`
	Direction     int       `json:"direction"`
	Weight        float64   `json:"weight_original"`
	Signal        float64   `json:"signal"` // -100..+100 after direction applied
	Mode          string    `json:"mode"`
	DropReason    string    `json:"drop_reason,omitempty"`
}

type CategoryScore struct {
	Category           string            `json:"category"`
	Score              float64           `json:"score"`
	WeightOriginal     float64           `json:"weight_original"`
	WeightRenormalized float64           `json:"weight_renormalized"`
	ActiveIndicators   int               `json:"active_indicators"`
	TotalIndicators    int               `json:"total_indicators"`
	Indicators         []IndicatorSignal `json:"indicators"`
}

type AssetScore struct {
	AssetCode  string
	Currency   string
	Composite  float64
	Bucket     string
	Categories []CategoryScore
	ScoredAt   time.Time
}

// Buckets per handoff: ±60 for very bullish/bearish, 0 splits bullish/bearish.
const (
	BucketVeryBullish = "VERY_BULLISH"
	BucketBullish     = "BULLISH"
	BucketBearish     = "BEARISH"
	BucketVeryBearish = "VERY_BEARISH"
)

func Bucket(composite float64) string {
	switch {
	case composite >= 60:
		return BucketVeryBullish
	case composite >= 0:
		return BucketBullish
	case composite > -60:
		return BucketBearish
	default:
		return BucketVeryBearish
	}
}

const BucketNeutral = "NEUTRAL"

// BucketPair maps a pair differential to a categorical label.
//
// Unlike Bucket(), this adds a NEUTRAL band between −10 and +10 so that
// near-zero differentials can be filtered out of the Top Setups screen.
// Same thresholds elsewhere:
//   >= +60   VERY_BULLISH
//   > +10    BULLISH
//   >= -10   NEUTRAL
//   > -60    BEARISH
//   <= -60   VERY_BEARISH
func BucketPair(differential float64) string {
	switch {
	case differential >= 60:
		return BucketVeryBullish
	case differential > 10:
		return BucketBullish
	case differential >= -10:
		return BucketNeutral
	case differential > -60:
		return BucketBearish
	default:
		return BucketVeryBearish
	}
}

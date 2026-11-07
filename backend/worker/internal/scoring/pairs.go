package scoring

import "time"

// PairScore is the differential score between a pair's base and quote currency.
// Differential = base_score − quote_score. Positive = buy the pair, negative = sell.
type PairScore struct {
	PairSymbol   string
	BaseCode     string
	QuoteCode    string
	BaseScore    float64
	QuoteScore   float64
	Differential float64
	Bucket       string
	ScoredAt     time.Time
}

// ScorePair computes a pair signal from the two underlying currency scores.
// No renormalization — the differential is a straight subtraction.
func ScorePair(
	pairSymbol, baseCode, quoteCode string,
	baseScore, quoteScore float64,
	scoredAt time.Time,
) PairScore {
	diff := baseScore - quoteScore
	return PairScore{
		PairSymbol:   pairSymbol,
		BaseCode:     baseCode,
		QuoteCode:    quoteCode,
		BaseScore:    baseScore,
		QuoteScore:   quoteScore,
		Differential: diff,
		Bucket:       BucketPair(diff),
		ScoredAt:     scoredAt,
	}
}

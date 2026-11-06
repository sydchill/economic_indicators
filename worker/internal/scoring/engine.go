package scoring

import (
	"math"
	"time"
)

// ComputeSignal produces the score contribution for one indicator observation.
//
// v1 uses trend mode only, because forecast_value is NULL on every row we have
// (manual imports don't carry TE forecasts). Trend mode = sign(change) × 100,
// where change = value − previous_value. This is deliberately coarse: without
// historical std devs we can't compute a defensible z-score, and inventing one
// per-indicator would hide the assumption. RawChange is preserved so the
// drill-down shows exactly what moved.
//
// Surprise mode is stubbed for the TE API swap — once forecast_value is
// populated, (value − forecast) / std(surprise) slots in here.
func ComputeSignal(
	cfg IndicatorConfig,
	obs *Observation,
	cutoff time.Time,
) IndicatorSignal {
	sig := IndicatorSignal{
		Slug:      cfg.Slug,
		Direction: cfg.Direction,
		Weight:    cfg.Weight,
	}

	if obs == nil {
		sig.Mode = ModeDropped
		sig.DropReason = "no observation"
		return sig
	}

	sig.Period = obs.Period
	sig.Value = obs.Value
	sig.PreviousValue = obs.PreviousValue
	sig.ForecastValue = obs.ForecastValue

	if obs.Period.Before(cutoff) {
		sig.Mode = ModeDropped
		sig.DropReason = "period older than lookback cutoff"
		return sig
	}

	// Future: if obs.ForecastValue != nil, use surprise mode.
	// Today all rows have ForecastValue == nil, so we always take trend.

	if obs.PreviousValue == nil {
		sig.Mode = ModeDropped
		sig.DropReason = "previous_value is null (cannot compute trend)"
		return sig
	}

	change := obs.Value - *obs.PreviousValue
	sig.RawChange = &change

	raw := 0.0
	switch {
	case change > 0:
		raw = 100.0
	case change < 0:
		raw = -100.0
	}
	sig.Signal = raw * float64(cfg.Direction)
	sig.Mode = ModeTrend
	return sig
}

// ScoreAsset aggregates per-category scores and the composite score for one asset.
//
// Algorithm:
//  1. Compute signal per indicator (ComputeSignal).
//  2. Per category: drop indicators with Mode="dropped"; renormalize remaining
//     weights to sum 1.0; score = Σ (weight × signal).
//  3. Drop categories with zero active indicators; renormalize remaining
//     category weights to sum 1.0.
//  4. composite = Σ (renormalized category weight × category score).
func ScoreAsset(
	assetCode, currency string,
	catWeights []CategoryWeight,
	indicators []IndicatorConfig, // all indicators for this asset, any category
	observations map[int64]*Observation, // keyed by indicator_id
	lookbackDays int,
	scoredAt time.Time,
) AssetScore {
	cutoff := scoredAt.AddDate(0, 0, -lookbackDays)

	// Group indicators by category for renormalization scope.
	byCategory := map[string][]IndicatorConfig{}
	for _, ic := range indicators {
		byCategory[ic.Category] = append(byCategory[ic.Category], ic)
	}

	var cats []CategoryScore
	activeCategoryWeightSum := 0.0
	for _, cw := range catWeights {
		cfgs := byCategory[cw.Category]

		// Compute signals for every configured indicator in this category.
		signals := make([]IndicatorSignal, 0, len(cfgs))
		activeWeightSum := 0.0
		activeCount := 0
		for _, ic := range cfgs {
			sig := ComputeSignal(ic, observations[ic.IndicatorID], cutoff)
			signals = append(signals, sig)
			if sig.Mode != ModeDropped {
				activeWeightSum += ic.Weight
				activeCount++
			}
		}

		cat := CategoryScore{
			Category:         cw.Category,
			WeightOriginal:   cw.Weight,
			TotalIndicators:  len(cfgs),
			ActiveIndicators: activeCount,
			Indicators:       signals,
		}

		if activeCount == 0 {
			// Category drops entirely — renormalization handled below.
			cat.WeightRenormalized = 0
			cats = append(cats, cat)
			continue
		}

		// Score the category with per-indicator weights renormalized.
		catScore := 0.0
		for _, sig := range signals {
			if sig.Mode == ModeDropped {
				continue
			}
			renormW := sig.Weight / activeWeightSum
			catScore += renormW * sig.Signal
		}
		cat.Score = catScore
		activeCategoryWeightSum += cw.Weight
		cats = append(cats, cat)
	}

	// Renormalize category weights and compute composite.
	composite := 0.0
	if activeCategoryWeightSum > 0 {
		for i := range cats {
			if cats[i].ActiveIndicators == 0 {
				continue
			}
			renormW := cats[i].WeightOriginal / activeCategoryWeightSum
			cats[i].WeightRenormalized = renormW
			composite += renormW * cats[i].Score
		}
	}

	return AssetScore{
		AssetCode:  assetCode,
		Currency:   currency,
		Composite:  math.Round(composite*100) / 100,
		Bucket:     Bucket(composite),
		Categories: cats,
		ScoredAt:   scoredAt,
	}
}

package macro

import (
	"fmt"
	"math"
)

// InstrumentScore InstrumentScore
type InstrumentScore struct {
	Symbol      string             `json:"symbol"`
	AssetType   string             `json:"asset_type"` // "index" or "metal"
	TotalScore  float64            `json:"total_score"`
	Components  map[string]float64 `json:"components"`
	Explanation string             `json:"explanation"`
}

// Hard-coded mapping for now; later you can extend this list.
var instrumentBase = []struct {
	Symbol    string
	AssetType string
	BaseFX    string // which macro country code/currency drives it
}{
	{"US500", "index", "USD"},
	{"US100", "index", "USD"},
	{"JP225", "index", "JPY"}, // Nikkei 225
	{"XAUUSD", "metal", "USD"},
	{"XAGUSD", "metal", "USD"},
}

// BuildInstrumentScores derives instrument scores from currency macro scores.
func BuildInstrumentScores(scoresByCountry map[string]ScoreBreakdown) map[string]InstrumentScore {
	out := make(map[string]InstrumentScore)

	for _, inst := range instrumentBase {
		baseScore, ok := scoresByCountry[inst.BaseFX]
		if !ok {
			continue
		}
		instScore := scoreInstrument(inst.Symbol, inst.AssetType, baseScore)
		out[inst.Symbol] = instScore
	}

	return out
}

// scoring rules per asset type
func scoreInstrument(symbol, assetType string, base ScoreBreakdown) InstrumentScore {
	comps := make(map[string]float64)

	// pull some key components for readability
	gdp := base.Components["gdp_growth"]
	unemp := base.Components["unemployment"]
	infl := base.Components["inflation"]
	rate := base.Components["interest_rate"]
	conf := base.Components["business_confidence"]
	manu := base.Components["manufacturing_pmi"]
	svc := base.Components["services_pmi"]

	switch assetType {
	case "index":
		// Risk assets like US500/US100 like:
		// + GDP growth, + confidence, + PMIs
		// - too high rates, - very high inflation
		comps["growth"] = gdp
		comps["confidence"] = avgNonZero(conf, manu, svc)
		comps["employment"] = unemp
		comps["rates_headwind"] = -rate
		comps["inflation_headwind"] = -math.Abs(infl)

	case "metal":
		// Metals like gold/silver tend to like:
		// + inflation (especially above target)
		// + lower real rates / dovish policy
		// + weaker currency (we invert the base macro score)
		comps["inflation_theme"] = math.Max(0, infl)   // only + if inflation above target
		comps["rates_theme"] = -rate                   // lower rates = positive
		comps["usd_weakness_theme"] = -base.TotalScore // weaker USD boosts XAUUSD/XAGUSD
	default:
		// fallback: just mirror base macro score
		comps["macro"] = base.TotalScore
	}

	// aggregate
	var sum float64
	var count float64
	for _, v := range comps {
		sum += v
		count++
	}
	total := 0.0
	if count > 0 {
		total = sum / count
	}
	total = round(total, 3)
	comps = roundMap(comps, 3)

	explanation := explainInstrument(symbol, assetType, base, comps, total)

	return InstrumentScore{
		Symbol:      symbol,
		AssetType:   assetType,
		TotalScore:  total,
		Components:  comps,
		Explanation: explanation,
	}
}

func avgNonZero(vals ...float64) float64 {
	var sum float64
	var count float64
	for _, v := range vals {
		if v != 0 {
			sum += v
			count++
		}
	}
	if count == 0 {
		return 0
	}
	return sum / count
}

func explainInstrument(symbol, assetType string, base ScoreBreakdown, comps map[string]float64, total float64) string {
	var bias string
	switch {
	case total > 0.3:
		bias = "strong bullish bias"
	case total > 0.1:
		bias = "mild bullish bias"
	case total < -0.3:
		bias = "strong bearish bias"
	case total < -0.1:
		bias = "mild bearish bias"
	default:
		bias = "roughly neutral stance"
	}

	text := fmt.Sprintf("%s currently has a %s (score %.2f) based on %s macro conditions.",
		symbol, bias, total, base.Country)

	switch assetType {
	case "index":
		text += explainIndexFromComponents(base, comps)
	case "metal":
		text += explainMetalFromComponents(base, comps)
	}

	return text
}

func explainIndexFromComponents(base ScoreBreakdown, comps map[string]float64) string {
	var parts []string

	if comps["growth"] > 0.15 {
		parts = append(parts, "solid GDP growth")
	}
	if comps["confidence"] > 0.15 {
		parts = append(parts, "resilient business and services activity")
	}
	if comps["employment"] > 0.15 {
		parts = append(parts, "low unemployment")
	}
	if comps["rates_headwind"] < -0.15 {
		parts = append(parts, "headwinds from high interest rates")
	}
	if comps["inflation_headwind"] < -0.15 {
		parts = append(parts, "concerns around inflation dynamics")
	}

	if len(parts) == 0 {
		return " Price action is mostly reflecting a balanced macro backdrop."
	}

	return " Key drivers are " + joinWithAnd(parts) + "."
}

func explainMetalFromComponents(base ScoreBreakdown, comps map[string]float64) string {
	var supports []string
	var headwinds []string

	if comps["inflation_theme"] > 0.15 {
		supports = append(supports, "elevated inflation")
	}
	if comps["rates_theme"] > 0.15 {
		supports = append(supports, "relatively low or easing interest rates")
	} else if comps["rates_theme"] < -0.15 {
		headwinds = append(headwinds, "higher interest rates")
	}

	if comps["usd_weakness_theme"] > 0.15 {
		supports = append(supports, "a softer US macro backdrop")
	} else if comps["usd_weakness_theme"] < -0.15 {
		headwinds = append(headwinds, "a stronger US macro backdrop")
	}

	text := ""
	if len(supports) > 0 {
		text += " Supportive factors include " + joinWithAnd(supports) + "."
	}
	if len(headwinds) > 0 {
		text += " On the other hand, " + joinWithAnd(headwinds) + " act as headwinds."
	}
	if text == "" {
		text = " Macro signals for this metal are mixed."
	}
	return text
}

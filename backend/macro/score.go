package macro

import (
	"fmt"
	"math"
	"strings"
)

// ScoreBreakdown ScoreBreakdown
type ScoreBreakdown struct {
	Country       string             `json:"country"`
	TotalScore    float64            `json:"total_score"`
	Components    map[string]float64 `json:"components"`
	RawIndicators MacroSnapshot      `json:"raw_indicators"`
	Explanation   string             `json:"explanation"`
}

// ScoreSnapshot ScoreSnapshot
func ScoreSnapshot(m MacroSnapshot) ScoreBreakdown {
	components := make(map[string]float64)

	components["gdp_growth"] = clamp(m.GDPAnnualGrowthRate/4.0, -1, 1)       // -4%→-1, 0→0, 4%→1
	components["unemployment"] = clamp((10.0-m.UnemploymentRate)/6.0, -1, 1) // 4%→~1, 10%→0, >10%→neg
	components["inflation"] = scoreInflation(m.InflationRate)
	components["interest_rate"] = clamp(m.InterestRate/10.0, -1, 1)          // up to 10% considered max positive
	components["current_account"] = clamp(m.CurrentAccount/100_000.0, -1, 1) // scaled down
	components["balance_of_trade"] = clamp(m.BalanceOfTrade/50_000.0, -1, 1)

	components["business_confidence"] = clamp(m.BusinessConfidence/100.0, -1, 1)
	components["manufacturing_pmi"] = scorePMI(m.ManufacturingPMI)

	if spmi := m.ParsedServicesPMI(); spmi != nil {
		components["services_pmi"] = scorePMI(*spmi)
	}

	components["consumer_confidence"] = clamp(m.ConsumerConfidence/100.0, -1, 1)
	components["retail_sales_mom"] = clamp(m.RetailSalesMoM/2.0, -1, 1) // ±2% saturates

	// average all non-zero components
	var sum float64
	var count float64
	for _, v := range components {
		// we include even negative ones, only skip NaNs
		if !math.IsNaN(v) {
			sum += v
			count++
		}
	}

	total := 0.0
	if count > 0 {
		total = sum / count
	}

	total = round(total, 3)
	components = roundMap(components, 3)

	explanation := explainMacroSnapshot(m, components, total)

	return ScoreBreakdown{
		Country:       m.Country,
		TotalScore:    total,
		Components:    components,
		RawIndicators: m,
		Explanation:   explanation,
	}
}

func scoreInflation(inflation float64) float64 {
	// 2% is ideal inflation target; penalize large deviations
	diff := inflation - 2.0
	// ±6% around target → [-1,1]
	return clamp(-diff/6.0, -1, 1)
}

func scorePMI(pmi float64) float64 {
	// PMI 50 = neutral, 60 = strong expansion, 40 = strong contraction
	return clamp((pmi-50.0)/10.0, -1, 1)
}

func clamp(v, min, max float64) float64 {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}

func round(v float64, decimals int) float64 {
	factor := math.Pow(10, float64(decimals))
	return math.Round(v*factor) / factor
}

func roundMap(m map[string]float64, decimals int) map[string]float64 {
	out := make(map[string]float64, len(m))
	for k, v := range m {
		out[k] = round(v, decimals)
	}
	return out
}

func explainMacroSnapshot(m MacroSnapshot, comps map[string]float64, total float64) string {
	// Overall direction
	overall := "roughly neutral"
	if total > 0.3 {
		overall = "overall strong"
	} else if total > 0.1 {
		overall = "slightly positive"
	} else if total < -0.3 {
		overall = "overall weak"
	} else if total < -0.1 {
		overall = "slightly negative"
	}

	// Find a few biggest positive/negative drivers
	type driver struct {
		key   string
		value float64
	}
	var positives, negatives []driver
	for k, v := range comps {
		if v > 0.15 {
			positives = append(positives, driver{k, v})
		} else if v < -0.15 {
			negatives = append(negatives, driver{k, v})
		}
	}

	// Simple name mapping for readability
	label := func(key string) string {
		switch key {
		case "gdp_growth":
			return "GDP growth"
		case "unemployment":
			return "low unemployment"
		case "inflation":
			return "inflation near target"
		case "interest_rate":
			return "interest rate level"
		case "current_account":
			return "current account balance"
		case "balance_of_trade":
			return "balance of trade"
		case "business_confidence":
			return "business confidence"
		case "manufacturing_pmi":
			return "manufacturing PMI"
		case "services_pmi":
			return "services PMI"
		case "consumer_confidence":
			return "consumer confidence"
		case "retail_sales_mom":
			return "retail sales momentum"
		default:
			return key
		}
	}

	// We don’t bother sorting; just take up to 2–3 from each side
	maxDrivers := 3
	var posLabels []string
	for i, d := range positives {
		if i >= maxDrivers {
			break
		}
		posLabels = append(posLabels, label(d.key))
	}
	var negLabels []string
	for i, d := range negatives {
		if i >= maxDrivers {
			break
		}
		negLabels = append(negLabels, label(d.key))
	}

	text := fmt.Sprintf("%s looks %s (score %.2f).", m.Country, overall, total)

	if len(posLabels) > 0 {
		text += " Supportive factors include " + joinWithAnd(posLabels) + "."
	}
	if len(negLabels) > 0 {
		text += " Headwinds come from " + joinWithAnd(negLabels) + "."
	}

	return text
}

func joinWithAnd(items []string) string {
	if len(items) == 0 {
		return ""
	}
	if len(items) == 1 {
		return items[0]
	}
	if len(items) == 2 {
		return items[0] + " and " + items[1]
	}
	// three or more
	last := items[len(items)-1]
	return strings.Join(items[:len(items)-1], ", ") + " and " + last
}

package macro

import "fmt"

// PairSentiment PairSentiment
type PairSentiment struct {
	Base         string         `json:"base"`
	Quote        string         `json:"quote"`
	BaseScore    float64        `json:"base_score"`
	QuoteScore   float64        `json:"quote_score"`
	PairScore    float64        `json:"pair_score"`
	BaseDetails  ScoreBreakdown `json:"base_details"`
	QuoteDetails ScoreBreakdown `json:"quote_details"`
	Explanation  string         `json:"explanation"`
}

// BuildScoresByCountry BuildScoresByCountry
func BuildScoresByCountry(snapshots []MacroSnapshot) map[string]ScoreBreakdown {
	out := make(map[string]ScoreBreakdown, len(snapshots))
	for _, s := range snapshots {
		score := ScoreSnapshot(s)
		out[s.Country] = score
	}
	return out
}

// PairSentimentFromSnapshots PairSentimentFromSnapshots
func PairSentimentFromSnapshots(
	snapshots []MacroSnapshot,
	base string,
	quote string,
) (PairSentiment, error) {
	scores := BuildScoresByCountry(snapshots)

	baseScore, okB := scores[base]
	quoteScore, okQ := scores[quote]
	if !okB || !okQ {
		return PairSentiment{}, fmt.Errorf("missing base or quote score: %v %v", okB, okQ)
	}

	pairScore := round(baseScore.TotalScore-quoteScore.TotalScore, 3)
	explanation := explainPair(baseScore, quoteScore, pairScore)

	return PairSentiment{
		Base:         base,
		Quote:        quote,
		BaseScore:    baseScore.TotalScore,
		QuoteScore:   quoteScore.TotalScore,
		PairScore:    pairScore,
		BaseDetails:  baseScore,
		QuoteDetails: quoteScore,
		Explanation:  explanation,
	}, nil
}

func explainPair(base, quote ScoreBreakdown, pairScore float64) string {
	var bias string
	switch {
	case pairScore > 0.3:
		bias = "much stronger"
	case pairScore > 0.1:
		bias = "stronger"
	case pairScore < -0.3:
		bias = "much weaker"
	case pairScore < -0.1:
		bias = "weaker"
	default:
		bias = "roughly in line"
	}

	text := fmt.Sprintf("%s looks %s than %s on macro fundamentals (pair score %.2f).",
		base.Country, bias, quote.Country, pairScore)

	// Find a couple of key comparative drivers: where base component - quote component is big
	type diff struct {
		name string
		dval float64
	}
	var diffs []diff
	for k, bv := range base.Components {
		if qv, ok := quote.Components[k]; ok {
			diffs = append(diffs, diff{k, bv - qv})
		}
	}

	// pick 2 strongest positive and 2 strongest negative edges
	var posEdges, negEdges []string
	for _, d := range diffs {
		if d.dval > 0.2 {
			posEdges = append(posEdges, explainEdge(kToLabel(d.name), true))
		} else if d.dval < -0.2 {
			negEdges = append(negEdges, explainEdge(kToLabel(d.name), false))
		}
		if len(posEdges) >= 2 && len(negEdges) >= 2 {
			break
		}
	}

	if len(posEdges) > 0 {
		text += " Favouring " + base.Country + " are " + joinWithAnd(posEdges) + "."
	}
	if len(negEdges) > 0 {
		text += " In contrast, " + quote.Country + " looks better in terms of " + joinWithAnd(negEdges) + "."
	}

	return text
}

func kToLabel(key string) string {
	switch key {
	case "gdp_growth":
		return "GDP growth"
	case "unemployment":
		return "unemployment"
	case "inflation":
		return "inflation stability"
	case "interest_rate":
		return "interest rate advantage"
	case "current_account":
		return "current account balance"
	case "balance_of_trade":
		return "trade balance"
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

func explainEdge(label string, favoursBase bool) string {
	if favoursBase {
		// phrase from base perspective
		switch label {
		case "GDP growth":
			return "stronger GDP growth"
		case "unemployment":
			return "lower unemployment"
		case "inflation stability":
			return "more stable inflation"
		case "interest rate advantage":
			return "higher interest rates"
		default:
			return label
		}
	}
	// phrase from quote perspective – we just reuse the label, the sentence will mention quote country
	return label
}

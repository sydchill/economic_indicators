package seed

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/uptrace/bun"

	"github.com/sydchill/economic_indicators/shared/models"
)

func Validate(ctx context.Context, db *bun.DB, plan *Plan) error {
	var dbAssets []string
	if err := db.NewSelect().Model((*models.Asset)(nil)).
		Column("code").Scan(ctx, &dbAssets); err != nil {
		return fmt.Errorf("load assets: %w", err)
	}
	assetSet := make(map[string]bool, len(dbAssets))
	for _, a := range dbAssets {
		assetSet[a] = true
	}

	var indicators []models.EconomicIndicator
	if err := db.NewSelect().Model(&indicators).Scan(ctx); err != nil {
		return fmt.Errorf("load indicators: %w", err)
	}
	indByCode := make(map[string]int, len(indicators))
	for _, i := range indicators {
		indByCode[i.Code] = i.ID
	}

	var errs []string
	for i := range plan.Assets {
		ap := &plan.Assets[i]
		errs = append(errs, validateAsset(ap, assetSet, indByCode)...)
	}

	if len(errs) > 0 {
		sort.Strings(errs)
		return fmt.Errorf("validation failed with %d error(s):\n  %s",
			len(errs), strings.Join(errs, "\n  "))
	}
	return nil
}

func validateAsset(ap *AssetPlan, assetSet map[string]bool, indByCode map[string]int) []string {
	var errs []string

	if !assetSet[ap.AssetID] {
		return []string{fmt.Sprintf("asset %s: not found in market.assets", ap.AssetID)}
	}

	catWeight := make(map[string]float64, len(ap.CategoryWeights))
	catParams := make(map[string]json.RawMessage, len(ap.CategoryWeights))
	var catSum float64

	for _, cw := range ap.CategoryWeights {
		if cw.Weight < 0 || cw.Weight > 1 {
			errs = append(errs, fmt.Sprintf("%s/%s: weight %v out of [0,1]", ap.AssetID, cw.Category, cw.Weight))
		}
		if _, dup := catWeight[cw.Category]; dup {
			errs = append(errs, fmt.Sprintf("%s/%s: duplicate category", ap.AssetID, cw.Category))
		}
		catWeight[cw.Category] = cw.Weight
		catParams[cw.Category] = cw.Params
		catSum += cw.Weight
	}
	if diff := absDiff(catSum, 1.0); diff > 0.001 {
		errs = append(errs, fmt.Sprintf("%s: category weights sum to %.4f, must be 1.0", ap.AssetID, catSum))
	}

	// Every asset must declare cot and rates explicitly.
	for _, required := range []string{"cot", "rates"} {
		if _, ok := catWeight[required]; !ok {
			errs = append(errs, fmt.Sprintf("%s: missing required category %q", ap.AssetID, required))
		}
	}

	// Validate params for special categories.
	for _, cat := range []string{"cot", "rates"} {
		if _, ok := catWeight[cat]; ok {
			errs = append(errs, validateCategoryParams(ap.AssetID, cat, catParams[cat])...)
		}
	}

	// Group indicators by category and resolve indicator IDs.
	indByCat := make(map[string][]IndicatorRow)
	for i := range ap.Indicators {
		row := &ap.Indicators[i]

		if row.Category == "cot" || row.Category == "rates" {
			errs = append(errs, fmt.Sprintf("%s/%s: category must not have indicator entries",
				ap.AssetID, row.Category))
			continue
		}

		id, ok := indByCode[row.IndicatorCode]
		if !ok {
			errs = append(errs, fmt.Sprintf("%s/%s: indicator %q not found",
				ap.AssetID, row.Category, row.IndicatorCode))
			continue
		}
		row.IndicatorID = id

		if row.Direction != -1 && row.Direction != 1 {
			errs = append(errs, fmt.Sprintf("%s/%s/%s: direction must be ±1, got %d",
				ap.AssetID, row.Category, row.IndicatorCode, row.Direction))
		}
		if row.Weight < 0 || row.Weight > 1 {
			errs = append(errs, fmt.Sprintf("%s/%s/%s: weight %v out of [0,1]",
				ap.AssetID, row.Category, row.IndicatorCode, row.Weight))
		}
		indByCat[row.Category] = append(indByCat[row.Category], *row)
	}

	// For each category with non-zero weight, sum indicator weights to 1.
	for cat, rows := range indByCat {
		if catWeight[cat] == 0 {
			continue
		}
		var sum float64
		for _, r := range rows {
			sum += r.Weight
		}
		if diff := absDiff(sum, 1.0); diff > 0.001 {
			errs = append(errs, fmt.Sprintf("%s/%s: indicator weights sum to %.4f, must be 1.0",
				ap.AssetID, cat, sum))
		}
	}

	// Categories with weight but no indicators are an error.
	for cat, w := range catWeight {
		if w == 0 || cat == "cot" || cat == "rates" {
			continue
		}
		if len(indByCat[cat]) == 0 {
			errs = append(errs, fmt.Sprintf("%s/%s: weight %v but no indicators configured",
				ap.AssetID, cat, w))
		}
	}

	return errs
}

func validateCategoryParams(assetID, cat string, raw json.RawMessage) []string {
	var p struct {
		Direction *int `json:"direction,omitempty"`
		Maturity  *int `json:"maturity,omitempty"`
	}
	if err := json.Unmarshal(raw, &p); err != nil {
		return []string{fmt.Sprintf("%s/%s: invalid params: %v", assetID, cat, err)}
	}

	var errs []string
	if p.Direction == nil || (*p.Direction != 1 && *p.Direction != -1) {
		errs = append(errs, fmt.Sprintf("%s/%s: params.direction must be ±1", assetID, cat))
	}
	if cat == "rates" {
		switch {
		case p.Maturity == nil:
			errs = append(errs, fmt.Sprintf("%s/%s: params.maturity required", assetID, cat))
		case *p.Maturity != 2 && *p.Maturity != 5 && *p.Maturity != 10 && *p.Maturity != 30:
			errs = append(errs, fmt.Sprintf("%s/%s: params.maturity must be 2,5,10,30", assetID, cat))
		}
	}
	return errs
}

func absDiff(a, b float64) float64 {
	if a > b {
		return a - b
	}
	return b - a
}

package seed

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
)

func LoadFile(path string) (*ConfigFile, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}

	var cfg ConfigFile
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&cfg); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}

	if cfg.Version != 1 {
		return nil, fmt.Errorf("unsupported config version %d (expected 1)", cfg.Version)
	}
	if len(cfg.Templates) == 0 {
		return nil, fmt.Errorf("%s: no templates defined", path)
	}
	if len(cfg.Assets) == 0 {
		return nil, fmt.Errorf("%s: no assets defined", path)
	}
	return &cfg, nil
}

// Resolve expands templates into per-asset plans, applying overrides.
func Resolve(cfg *ConfigFile) (*Plan, error) {
	plan := &Plan{}

	for assetID, entry := range cfg.Assets {
		base, ok := cfg.Templates[entry.Template]
		if !ok {
			return nil, fmt.Errorf("asset %s: unknown template %q", assetID, entry.Template)
		}
		if entry.Currency == "" {
			return nil, fmt.Errorf("asset %s: currency must be set", assetID)
		}

		effective := base
		if entry.Override != nil {
			effective = mergeTemplates(base, *entry.Override)
		}

		ap := AssetPlan{AssetID: assetID}

		for cat, cw := range effective.CategoryWeights {
			params := cw.Params
			if len(params) == 0 {
				params = json.RawMessage("{}")
			}
			ap.CategoryWeights = append(ap.CategoryWeights, CategoryWeightRow{
				AssetID:  assetID,
				Category: cat,
				Weight:   cw.Weight,
				Params:   params,
			})
		}

		for cat, defs := range effective.Indicators {
			for _, def := range defs {
				ap.Indicators = append(ap.Indicators, IndicatorRow{
					AssetID:       assetID,
					Category:      cat,
					IndicatorCode: entry.Currency + "_" + def.Suffix,
					Direction:     def.Direction,
					Weight:        def.Weight,
				})
			}
		}

		plan.Assets = append(plan.Assets, ap)
	}
	return plan, nil
}

// mergeTemplates applies an override template on top of a base template.
// Category weights merge by key; indicator lists replace per category.
func mergeTemplates(base, override Template) Template {
	out := Template{
		CategoryWeights: make(map[string]CategoryWeight, len(base.CategoryWeights)),
		Indicators:      make(map[string][]IndicatorDef, len(base.Indicators)),
	}
	for k, v := range base.CategoryWeights {
		out.CategoryWeights[k] = v
	}
	for k, v := range override.CategoryWeights {
		out.CategoryWeights[k] = v
	}
	for k, v := range base.Indicators {
		out.Indicators[k] = v
	}
	for k, v := range override.Indicators {
		out.Indicators[k] = v // full category replacement
	}
	return out
}

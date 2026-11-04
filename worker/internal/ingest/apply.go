package ingest

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/uptrace/bun"

	"github.com/sydchill/economic_indicators/shared/models"
)

// Apply replaces each asset's config rows inside a single transaction.
// Pre-existing rows for a given asset are deleted first, so re-running is
// safe and idempotent.
func Apply(ctx context.Context, db *bun.DB, plan *Plan) error {
	return db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		for _, ap := range plan.Assets {
			if _, err := tx.NewDelete().
				Model((*models.AssetCategoryWeight)(nil)).
				Where("asset_id = ?", ap.AssetID).
				Exec(ctx); err != nil {
				return fmt.Errorf("delete weights %s: %w", ap.AssetID, err)
			}
			if _, err := tx.NewDelete().
				Model((*models.AssetIndicatorConfig)(nil)).
				Where("asset_id = ?", ap.AssetID).
				Exec(ctx); err != nil {
				return fmt.Errorf("delete indicators %s: %w", ap.AssetID, err)
			}

			if len(ap.CategoryWeights) > 0 {
				rows := make([]models.AssetCategoryWeight, len(ap.CategoryWeights))
				for i, w := range ap.CategoryWeights {
					params := w.Params
					if len(params) == 0 {
						params = json.RawMessage("{}")
					}
					rows[i] = models.AssetCategoryWeight{
						AssetID:  w.AssetID,
						Category: w.Category,
						Weight:   w.Weight,
						Params:   params,
					}
				}
				if _, err := tx.NewInsert().Model(&rows).Exec(ctx); err != nil {
					return fmt.Errorf("insert weights %s: %w", ap.AssetID, err)
				}
			}

			if len(ap.Indicators) > 0 {
				rows := make([]models.AssetIndicatorConfig, len(ap.Indicators))
				for i, r := range ap.Indicators {
					rows[i] = models.AssetIndicatorConfig{
						AssetID:     r.AssetID,
						IndicatorID: r.IndicatorID,
						Category:    r.Category,
						Direction:   r.Direction,
						Weight:      r.Weight,
					}
				}
				if _, err := tx.NewInsert().Model(&rows).Exec(ctx); err != nil {
					return fmt.Errorf("insert indicators %s: %w", ap.AssetID, err)
				}
			}
		}
		return nil
	})
}

package models_test

import (
	"context"
	"os"
	"testing"

	"github.com/sydchill/economic_indicators/shared/db"
	"github.com/sydchill/economic_indicators/shared/models"
)

func TestMarketSchemaRoundTrip(t *testing.T) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL not set")
	}
	ctx := context.Background()
	bunDB, err := db.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer bunDB.Close()

	// Read: 13 assets, 28 pairs, 104 indicators
	var assets int
	if err := bunDB.NewSelect().Model((*models.Asset)(nil)).ColumnExpr("count(*)").Scan(ctx, &assets); err != nil {
		t.Fatal(err)
	}
	if assets != 13 {
		t.Fatalf("expected 13 assets, got %d", assets)
	}

	var pairs int
	if err := bunDB.NewSelect().Model((*models.Pair)(nil)).ColumnExpr("count(*)").Scan(ctx, &pairs); err != nil {
		t.Fatal(err)
	}
	if pairs != 28 {
		t.Fatalf("expected 28 pairs, got %d", pairs)
	}

	// Read: 7 majors
	majors, err := bunDB.NewSelect().Model((*models.Pair)(nil)).
		Where("p.is_major = TRUE").Count(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if majors != 7 {
		t.Fatalf("expected 7 majors, got %d", majors)
	}

	// Filter by EUR — the query screen 2 will use
	eurPairs, err := bunDB.NewSelect().Model((*models.Pair)(nil)).
		Where("p.base_currency = ? OR p.quote_currency = ?", "EUR", "EUR").
		Count(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if eurPairs != 13 {
		t.Fatalf("expected 13 EUR pairs, got %d", eurPairs)
	}

	// Load a specific asset
	usd := new(models.Asset)
	if err := bunDB.NewSelect().Model(usd).Where("a.code = ?", "USD").Scan(ctx); err != nil {
		t.Fatal(err)
	}
	if usd.AssetClass != "currency" || !usd.IsMajor {
		t.Fatalf("USD misconfigured: class=%s major=%v", usd.AssetClass, usd.IsMajor)
	}

	// Load an index — primary_currency should resolve
	nas := new(models.Asset)
	if err := bunDB.NewSelect().Model(nas).Where("a.code = ?", "NAS100").Scan(ctx); err != nil {
		t.Fatal(err)
	}
	if nas.PrimaryCurrency == nil || *nas.PrimaryCurrency != "USD" {
		t.Fatalf("NAS100 primary_currency wrong: %v", nas.PrimaryCurrency)
	}
}

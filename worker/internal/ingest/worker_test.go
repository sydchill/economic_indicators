package ingest

import (
	"testing"
)

func TestIndicatorCode(t *testing.T) {
	cases := map[[2]string]string{
		{"USD", "CPI YoY"}:              "USD_CPI_YOY",
		{"EUR", "  Non-Farm Payrolls "}: "EUR_NON-FARM_PAYROLLS",
		{"JPY", "GDP QoQ"}:              "JPY_GDP_QOQ",
	}
	for in, want := range cases {
		if got := indicatorCode(in[0], in[1]); got != want {
			t.Errorf("indicatorCode(%q,%q) = %q, want %q", in[0], in[1], got, want)
		}
	}
}

func TestNormalizeImportance(t *testing.T) {
	cases := map[string]string{
		"HIGH": "high", "High": "high", "high": "high",
		"MEDIUM": "medium", "medium": "medium",
		"low": "low", "": "low", "whatever": "low",
	}
	for in, want := range cases {
		if got := normalizeImportance(in); got != want {
			t.Errorf("normalizeImportance(%q) = %q, want %q", in, got, want)
		}
	}
}

package ingest

// MajorCountryCodes is the ISO 3166-1 alpha-2 list for the eight G10
// currencies the ingestion worker tracks by default. Callers may pass a
// different list to Worker.RunIngestOnce to narrow or widen the scope.
var MajorCountryCodes = []string{
	"US", // USD
	"EU", // EUR (euro area — biquote uses EU as the country code)
	"GB", // GBP
	"JP", // JPY
	"AU", // AUD
	"NZ", // NZD
	"CA", // CAD
	"CH", // CHF
}

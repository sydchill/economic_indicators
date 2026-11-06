package snapshot

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"time"
)

var periodRe = regexp.MustCompile(`^\d{4}-(0[1-9]|1[0-2])$`)

// Load reads and validates a snapshot file. Returns the parsed File or an error
// describing the first validation failure (row index included).
func Load(path string) (*File, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var f File
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&f); err != nil {
		return nil, fmt.Errorf("json: %w", err)
	}
	if err := validate(&f); err != nil {
		return nil, err
	}
	return &f, nil
}

func validate(f *File) error {
	if _, err := time.Parse("2006-01-02", f.SnapshotDate); err != nil {
		return fmt.Errorf("snapshot_date: %w", err)
	}
	snapMonth := f.SnapshotDate[:7] // YYYY-MM
	if len(f.EconomicData) == 0 {
		return fmt.Errorf("economic_data: must contain at least one row")
	}
	seen := make(map[string]struct{}, len(f.EconomicData))
	for i, d := range f.EconomicData {
		if d.Slug == "" {
			return fmt.Errorf("row %d: slug is required", i)
		}
		if d.Currency == "" {
			return fmt.Errorf("row %d: currency is required (slug=%s)", i, d.Slug)
		}
		if d.Unit == "" {
			return fmt.Errorf("row %d: unit is required (slug=%s currency=%s)", i, d.Slug, d.Currency)
		}
		if !periodRe.MatchString(d.Period) {
			return fmt.Errorf("row %d: period must be YYYY-MM, got %q", i, d.Period)
		}
		if d.Period > snapMonth {
			return fmt.Errorf("row %d: period %s is after snapshot month %s (skip future-dated rows — do not include them)", i, d.Period, snapMonth)
		}
		key := d.Slug + "|" + d.Currency + "|" + d.Period
		if _, dup := seen[key]; dup {
			return fmt.Errorf("row %d: duplicate (slug=%s currency=%s period=%s)", i, d.Slug, d.Currency, d.Period)
		}
		seen[key] = struct{}{}
	}
	return nil
}

// PeriodDate converts a YYYY-MM string to first-of-month UTC.
func PeriodDate(s string) (time.Time, error) {
	return time.Parse("2006-01", s)
}

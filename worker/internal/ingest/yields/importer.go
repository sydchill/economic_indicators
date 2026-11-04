package yields

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/uptrace/bun"
)

// ImportCSV reads a file with header:
//   currency,maturity_years,record_date,yield_percent
// and upserts each row into market.bond_yields.
func ImportCSV(ctx context.Context, db *bun.DB, path string) (int, error) {
	f, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer f.Close()

	r := csv.NewReader(f)
	r.FieldsPerRecord = 4
	header, err := r.Read()
	if err != nil {
		return 0, fmt.Errorf("read header: %w", err)
	}
	expected := []string{"currency", "maturity_years", "record_date", "yield_percent"}
	for i, h := range header {
		if strings.ToLower(strings.TrimSpace(h)) != expected[i] {
			return 0, fmt.Errorf("header col %d: want %q, got %q", i, expected[i], h)
		}
	}

	var count int
	for {
		row, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return count, err
		}

		ccy := strings.ToUpper(strings.TrimSpace(row[0]))
		maturity, err := strconv.Atoi(strings.TrimSpace(row[1]))
		if err != nil {
			return count, fmt.Errorf("row %d: maturity: %w", count+2, err)
		}
		if maturity != 2 && maturity != 5 && maturity != 10 && maturity != 30 {
			return count, fmt.Errorf("row %d: maturity must be 2,5,10,30", count+2)
		}
		date, err := time.Parse("2006-01-02", strings.TrimSpace(row[2]))
		if err != nil {
			return count, fmt.Errorf("row %d: date: %w", count+2, err)
		}
		y, err := strconv.ParseFloat(strings.TrimSpace(row[3]), 64)
		if err != nil {
			return count, fmt.Errorf("row %d: yield: %w", count+2, err)
		}

		_, err = db.NewRaw(`
			INSERT INTO market.bond_yields
				(currency_code, maturity_years, record_date, yield_percent, source)
			VALUES (?, ?, ?, ?, 'manual_csv')
			ON CONFLICT (currency_code, maturity_years, record_date)
			DO UPDATE SET yield_percent = EXCLUDED.yield_percent,
			              source = EXCLUDED.source
		`, ccy, maturity, date, y).Exec(ctx)
		if err != nil {
			return count, fmt.Errorf("row %d: %w", count+2, err)
		}
		count++
	}
	return count, nil
}

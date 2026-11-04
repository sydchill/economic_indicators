package yields

import (
	"context"
	"encoding/csv"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type TreasuryClient struct {
	http *http.Client
}

func NewTreasuryClient() *TreasuryClient {
	return &TreasuryClient{http: &http.Client{Timeout: 30 * time.Second}}
}

type YieldRow struct {
	CurrencyCode  string
	MaturityYears int16
	RecordDate    time.Time
	YieldPercent  float64
	Source        string
}

func (c *TreasuryClient) Fetch(ctx context.Context, year int) ([]YieldRow, error) {
	url := fmt.Sprintf(
		"https://home.treasury.gov/resource-center/data-chart-center/interest-rates/daily-treasury-rates.csv/%d/all?type=daily_treasury_yield_curve&field_tdr_date_value=%d&page&_format=csv",
		year, year,
	)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "currency-signal/1.0")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("treasury request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("treasury status %d", resp.StatusCode)
	}

	r := csv.NewReader(resp.Body)
	r.FieldsPerRecord = -1
	records, err := r.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("treasury csv: %w", err)
	}
	if len(records) < 2 {
		return nil, fmt.Errorf("treasury csv empty")
	}

	header := records[0]
	col := make(map[string]int, len(header))
	for i, h := range header {
		col[strings.TrimSpace(strings.ToLower(h))] = i
	}

	dateIdx, ok := col["date"]
	if !ok {
		return nil, fmt.Errorf("treasury csv: missing Date column")
	}
	tenorIdx := map[int16]int{}
	for _, t := range []int16{2, 5, 10, 30} {
		key := strings.ToLower(fmt.Sprintf("%d yr", t))
		idx, ok := col[key]
		if !ok {
			return nil, fmt.Errorf("treasury csv: missing %q column", key)
		}
		tenorIdx[t] = idx
	}

	var out []YieldRow
	for _, row := range records[1:] {
		if len(row) <= dateIdx {
			continue
		}
		date, err := time.Parse("01/02/2006", strings.TrimSpace(row[dateIdx]))
		if err != nil {
			continue
		}
		for tenor, idx := range tenorIdx {
			if idx >= len(row) {
				continue
			}
			v := strings.TrimSpace(row[idx])
			if v == "" || v == "N/A" {
				continue
			}
			y, err := strconv.ParseFloat(v, 64)
			if err != nil {
				continue
			}
			out = append(out, YieldRow{
				CurrencyCode:  "USD",
				MaturityYears: tenor,
				RecordDate:    date,
				YieldPercent:  y,
				Source:        "treasury_csv",
			})
		}
	}
	return out, nil
}

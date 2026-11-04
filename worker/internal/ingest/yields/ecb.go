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

type ECBClient struct {
	http *http.Client
}

func NewECBClient() *ECBClient {
	return &ECBClient{http: &http.Client{Timeout: 30 * time.Second}}
}

// ECB spot-rate maturities we care about, mapped to years.
var ecbTenors = map[string]int16{
	"SR_2Y":  2,
	"SR_5Y":  5,
	"SR_10Y": 10,
	"SR_30Y": 30,
}

func (c *ECBClient) Fetch(ctx context.Context, startDate time.Time) ([]YieldRow, error) {
	var out []YieldRow

	for key, years := range ecbTenors {
		rows, err := c.fetchSeries(ctx, key, years, startDate)
		if err != nil {
			return nil, err
		}
		out = append(out, rows...)
	}
	return out, nil
}

func (c *ECBClient) fetchSeries(ctx context.Context, key string, years int16, start time.Time) ([]YieldRow, error) {
	url := fmt.Sprintf(
		"https://data-api.ecb.europa.eu/service/data/YC/B.U2.EUR.4F.G_N_A.SV_C_YM.%s?format=csvdata&startPeriod=%s",
		key, start.Format("2006-01-02"),
	)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "currency-signal/1.0")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ecb %s request: %w", key, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ecb %s status %d", key, resp.StatusCode)
	}

	r := csv.NewReader(resp.Body)
	r.FieldsPerRecord = -1
	records, err := r.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("ecb %s csv: %w", key, err)
	}
	if len(records) < 2 {
		return nil, nil
	}

	header := records[0]
	periodIdx, valueIdx := -1, -1
	for i, h := range header {
		switch strings.TrimSpace(h) {
		case "TIME_PERIOD":
			periodIdx = i
		case "OBS_VALUE":
			valueIdx = i
		}
	}
	if periodIdx < 0 || valueIdx < 0 {
		return nil, fmt.Errorf("ecb %s: missing TIME_PERIOD or OBS_VALUE", key)
	}

	var out []YieldRow
	for _, row := range records[1:] {
		if periodIdx >= len(row) || valueIdx >= len(row) {
			continue
		}
		date, err := time.Parse("2006-01-02", strings.TrimSpace(row[periodIdx]))
		if err != nil {
			continue
		}
		v := strings.TrimSpace(row[valueIdx])
		if v == "" {
			continue
		}
		y, err := strconv.ParseFloat(v, 64)
		if err != nil {
			continue
		}
		out = append(out, YieldRow{
			CurrencyCode:  "EUR",
			MaturityYears: years,
			RecordDate:    date,
			YieldPercent:  y,
			Source:        "ecb_sdmx",
		})
	}
	return out, nil
}

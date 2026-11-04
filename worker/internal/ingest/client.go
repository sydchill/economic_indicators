package ingest

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type BiquoteEvent struct {
	EventID         string    `json:"eventId"`
	Time            time.Time `json:"time"`
	CountryCode     string    `json:"countryCode"`
	Currency        string    `json:"currency"`
	Name            string    `json:"name"`
	Importance      string    `json:"importance"`
	Type            string    `json:"type"`
	Unit            string    `json:"unit"`
	Multiplier      string    `json:"multiplier"`
	Actual          *float64  `json:"actual"`
	Forecast        *float64  `json:"forecast"`
	Previous        *float64  `json:"previous"`
	RevisedPrevious *float64  `json:"revisedPrevious"`
	TimeMode        string    `json:"timeMode"`
}

type BiquoteClient struct {
	baseURL string
	http    *http.Client
}

func NewBiquoteClient(baseURL string) *BiquoteClient {
	if baseURL == "" {
		baseURL = "https://biquote.io"
	}
	return &BiquoteClient{
		baseURL: strings.TrimRight(baseURL, "/"),
		http:    &http.Client{Timeout: 20 * time.Second},
	}
}

func (c *BiquoteClient) FetchCalendar(ctx context.Context, countries []string, importance string) ([]BiquoteEvent, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/api/calendar", nil)
	if err != nil {
		return nil, err
	}

	q := req.URL.Query()
	if len(countries) > 0 {
		q.Set("countries", strings.Join(countries, ","))
	}
	if importance != "" {
		q.Set("importance", importance)
	}
	req.URL.RawQuery = q.Encode()

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("biquote request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("biquote status %d", resp.StatusCode)
	}

	var events []BiquoteEvent
	if err := json.NewDecoder(resp.Body).Decode(&events); err != nil {
		return nil, fmt.Errorf("biquote decode: %w", err)
	}
	return events, nil
}

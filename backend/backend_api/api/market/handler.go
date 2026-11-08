package market

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/uptrace/bun"
)

type Handler struct{ db *bun.DB }

func NewHandler(db *bun.DB) *Handler { return &Handler{db: db} }

func (h *Handler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Get("/assets", h.ListAssets)
	r.Get("/assets/{code}", h.GetAsset)
	r.Get("/pairs", h.ListPairs)
	r.Get("/pairs/{symbol}", h.GetPair)
	return r
}

// ---------------------------------------------------------------
// Asset scores
// ---------------------------------------------------------------

// AssetScoreView carries the aggregate score AND the full drill-down
// (category_scores + category_inputs) so the screen can render
// "why did USD score +45.6?" without an extra request.
type AssetScoreView struct {
	Code           string          `json:"code"`
	Name           string          `json:"name"`
	Category       string          `json:"category"`
	Currency       string          `json:"currency,omitempty"`
	CompositeScore *float64        `json:"composite_score"`
	Bucket         *string         `json:"bucket"`
	ScoredAt       *string         `json:"scored_at"`
	CategoryScores json.RawMessage `json:"category_scores,omitempty"`
	CategoryInputs json.RawMessage `json:"category_inputs,omitempty"`
	HasScore       bool            `json:"has_score"`
}

// GET /api/market/assets
// Every asset, joined with its latest score. Un-scored assets return
// has_score=false with null composite_score/bucket.
func (h *Handler) ListAssets(w http.ResponseWriter, r *http.Request) {
	type row struct {
		Code           string          `bun:"code"`
		Name           string          `bun:"name"`
		Category       string          `bun:"category"`
		Currency       *string         `bun:"currency"`
		CompositeScore *float64        `bun:"composite_score"`
		Bucket         *string         `bun:"bucket"`
		ScoredAt       *time.Time      `bun:"scored_at"`
		CategoryScores json.RawMessage `bun:"category_scores"`
		CategoryInputs json.RawMessage `bun:"category_inputs"`
	}

	var rows []row
	err := h.db.NewRaw(`
		SELECT a.code, a.name, a.category, a.currency,
		       s.composite_score, s.bucket, s.scored_at,
		       s.category_scores, s.category_inputs
		FROM market.assets a
		LEFT JOIN LATERAL (
			SELECT composite_score, bucket, scored_at, category_scores, category_inputs
			FROM market.asset_scores
			WHERE asset_id = a.id
			ORDER BY scored_at DESC
			LIMIT 1
		) s ON TRUE
		ORDER BY COALESCE(s.composite_score, 0) DESC, a.code
	`).Scan(r.Context(), &rows)
	if err != nil {
		writeError(w, 500, err.Error())
		return
	}

	out := make([]AssetScoreView, 0, len(rows))
	for _, rw := range rows {
		v := AssetScoreView{
			Code:           rw.Code,
			Name:           rw.Name,
			Category:       rw.Category,
			CompositeScore: rw.CompositeScore,
			Bucket:         rw.Bucket,
			CategoryScores: rw.CategoryScores,
			CategoryInputs: rw.CategoryInputs,
			HasScore:       rw.CompositeScore != nil,
		}
		if rw.Currency != nil {
			v.Currency = *rw.Currency
		}
		if rw.ScoredAt != nil {
			s := rw.ScoredAt.UTC().Format("2006-01-02")
			v.ScoredAt = &s
		}
		out = append(out, v)
	}
	writeJSON(w, 200, out)
}

// GET /api/market/assets/{code}
func (h *Handler) GetAsset(w http.ResponseWriter, r *http.Request) {
	code := chi.URLParam(r, "code")

	type row struct {
		Code           string          `bun:"code"`
		Name           string          `bun:"name"`
		Category       string          `bun:"category"`
		Currency       *string         `bun:"currency"`
		CompositeScore *float64        `bun:"composite_score"`
		Bucket         *string         `bun:"bucket"`
		ScoredAt       *time.Time      `bun:"scored_at"`
		CategoryScores json.RawMessage `bun:"category_scores"`
		CategoryInputs json.RawMessage `bun:"category_inputs"`
	}

	var rw row
	err := h.db.NewRaw(`
		SELECT a.code, a.name, a.category, a.currency,
		       s.composite_score, s.bucket, s.scored_at,
		       s.category_scores, s.category_inputs
		FROM market.assets a
		LEFT JOIN LATERAL (
			SELECT composite_score, bucket, scored_at, category_scores, category_inputs
			FROM market.asset_scores
			WHERE asset_id = a.id
			ORDER BY scored_at DESC
			LIMIT 1
		) s ON TRUE
		WHERE a.code = ?
	`, code).Scan(r.Context(), &rw)
	if err != nil {
		writeError(w, 404, "asset not found")
		return
	}

	v := AssetScoreView{
		Code:           rw.Code,
		Name:           rw.Name,
		Category:       rw.Category,
		CompositeScore: rw.CompositeScore,
		Bucket:         rw.Bucket,
		CategoryScores: rw.CategoryScores,
		CategoryInputs: rw.CategoryInputs,
		HasScore:       rw.CompositeScore != nil,
	}
	if rw.Currency != nil {
		v.Currency = *rw.Currency
	}
	if rw.ScoredAt != nil {
		s := rw.ScoredAt.UTC().Format("2006-01-02")
		v.ScoredAt = &s
	}
	writeJSON(w, 200, v)
}

// ---------------------------------------------------------------
// Pair scores
// ---------------------------------------------------------------

type PairScoreView struct {
	Symbol        string   `json:"symbol"`
	BaseCurrency  string   `json:"base_currency"`
	QuoteCurrency string   `json:"quote_currency"`
	BaseScore     *float64 `json:"base_score"`
	QuoteScore    *float64 `json:"quote_score"`
	Differential  *float64 `json:"differential"`
	Bucket        *string  `json:"bucket"`
	ScoredAt      *string  `json:"scored_at"`
	HasScore      bool     `json:"has_score"`
}

// GET /api/market/pairs?currency=EUR
// All pairs with their latest differential. Optional ?currency= filter
// narrows to pairs where either side matches.
func (h *Handler) ListPairs(w http.ResponseWriter, r *http.Request) {
	type row struct {
		Symbol        string     `bun:"symbol"`
		BaseCurrency  string     `bun:"base_currency"`
		QuoteCurrency string     `bun:"quote_currency"`
		BaseScore     *float64   `bun:"base_score"`
		QuoteScore    *float64   `bun:"quote_score"`
		Differential  *float64   `bun:"differential"`
		Bucket        *string    `bun:"bucket"`
		ScoredAt      *time.Time `bun:"scored_at"`
	}

	const sqlBase = `
		SELECT p.symbol, p.base_currency, p.quote_currency,
		       ps.base_score, ps.quote_score, ps.differential, ps.bucket, ps.scored_at
		FROM market.pairs p
		LEFT JOIN LATERAL (
			SELECT base_score, quote_score, differential, bucket, scored_at
			FROM market.pair_scores
			WHERE pair_id = p.id
			ORDER BY scored_at DESC
			LIMIT 1
		) ps ON TRUE
	`

	var rows []row
	ccy := r.URL.Query().Get("currency")
	var err error
	if ccy != "" {
		err = h.db.NewRaw(sqlBase+`
			WHERE p.base_currency = ? OR p.quote_currency = ?
			ORDER BY COALESCE(ps.differential, 0) DESC, p.symbol
		`, ccy, ccy).Scan(r.Context(), &rows)
	} else {
		err = h.db.NewRaw(sqlBase+`
			ORDER BY COALESCE(ps.differential, 0) DESC, p.symbol
		`).Scan(r.Context(), &rows)
	}
	if err != nil {
		writeError(w, 500, err.Error())
		return
	}

	out := make([]PairScoreView, 0, len(rows))
	for _, rw := range rows {
		v := PairScoreView{
			Symbol:        rw.Symbol,
			BaseCurrency:  rw.BaseCurrency,
			QuoteCurrency: rw.QuoteCurrency,
			BaseScore:     rw.BaseScore,
			QuoteScore:    rw.QuoteScore,
			Differential:  rw.Differential,
			Bucket:        rw.Bucket,
			HasScore:      rw.Differential != nil,
		}
		if rw.ScoredAt != nil {
			s := rw.ScoredAt.UTC().Format("2006-01-02")
			v.ScoredAt = &s
		}
		out = append(out, v)
	}
	writeJSON(w, 200, out)
}

// GET /api/market/pairs/{symbol}
func (h *Handler) GetPair(w http.ResponseWriter, r *http.Request) {
	symbol := chi.URLParam(r, "symbol")

	type row struct {
		Symbol        string     `bun:"symbol"`
		BaseCurrency  string     `bun:"base_currency"`
		QuoteCurrency string     `bun:"quote_currency"`
		BaseScore     *float64   `bun:"base_score"`
		QuoteScore    *float64   `bun:"quote_score"`
		Differential  *float64   `bun:"differential"`
		Bucket        *string    `bun:"bucket"`
		ScoredAt      *time.Time `bun:"scored_at"`
	}

	var rw row
	err := h.db.NewRaw(`
		SELECT p.symbol, p.base_currency, p.quote_currency,
		       ps.base_score, ps.quote_score, ps.differential, ps.bucket, ps.scored_at
		FROM market.pairs p
		LEFT JOIN LATERAL (
			SELECT base_score, quote_score, differential, bucket, scored_at
			FROM market.pair_scores
			WHERE pair_id = p.id
			ORDER BY scored_at DESC
			LIMIT 1
		) ps ON TRUE
		WHERE p.symbol = ?
	`, symbol).Scan(r.Context(), &rw)
	if err != nil {
		writeError(w, 404, "pair not found")
		return
	}

	v := PairScoreView{
		Symbol:        rw.Symbol,
		BaseCurrency:  rw.BaseCurrency,
		QuoteCurrency: rw.QuoteCurrency,
		BaseScore:     rw.BaseScore,
		QuoteScore:    rw.QuoteScore,
		Differential:  rw.Differential,
		Bucket:        rw.Bucket,
		HasScore:      rw.Differential != nil,
	}
	if rw.ScoredAt != nil {
		s := rw.ScoredAt.UTC().Format("2006-01-02")
		v.ScoredAt = &s
	}
	writeJSON(w, 200, v)
}

// ---------------------------------------------------------------

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/uptrace/bun"
)

// API api
type API struct {
	DB *bun.DB
}

// New new
func New(db *bun.DB) *API {
	return &API{DB: db}
}

func cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// allow your dev frontend
		w.Header().Set("Access-Control-Allow-Origin", "http://localhost:5173")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// Router router
func (a *API) Router() http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(cors)

	r.Get("/api/v1/health", a.HandleHealth)
	r.Get("/api/v1/currencies", a.HandleListCurrencies)

	// new ones:
	r.Get("/api/v1/macro/scores", a.HandleMacroScores)
	r.Get("/api/v1/macro/pair", a.HandleMacroPairSentiment)
	r.Get("/api/v1/instruments/scores", a.HandleInstrumentScores)

	return r
}

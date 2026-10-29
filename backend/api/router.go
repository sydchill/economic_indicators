package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
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
	return cors(chi.NewRouter())
}

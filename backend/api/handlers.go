package api

import (
	"context"
	"economic_indicator/models"
	"encoding/json"
	"net/http"
)

// HandleHealth HandleHealth
func (a *API) HandleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// HandleListCurrencies HandleListCurrencies
func (a *API) HandleListCurrencies(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var currencies []models.Currency

	err := a.DB.NewSelect().
		Model(&currencies).
		Order("code ASC").
		Scan(ctx)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "database error: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"data": currencies})
}

// helpers

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]any{"error": msg})
}

// (optional) convenience for internal use
func ctx(r *http.Request) context.Context {
	return r.Context()
}

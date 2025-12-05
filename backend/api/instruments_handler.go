package api

import (
	"economic_indicator/macro"
	"net/http"
)

// HandleInstrumentScores HandleInstrumentScores
func (a *API) HandleInstrumentScores(w http.ResponseWriter, r *http.Request) {
	snapshots, err := macro.LoadSnapshots("data/macro.json")
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load macro data: "+err.Error())
		return
	}

	// 1) build currency scores
	currencyScores := macro.BuildScoresByCountry(snapshots)
	// 2) derive instrument scores
	instScores := macro.BuildInstrumentScores(currencyScores)

	writeJSON(w, http.StatusOK, map[string]any{
		"data": instScores,
	})
}

package api

import (
	"economic_indicator/macro"
	"net/http"
)

const macroFilePath = "data/macro.json" // adjust if needed
// HandleMacroScores HandleMacroScores
func (a *API) HandleMacroScores(w http.ResponseWriter, r *http.Request) {
	snapshots, err := macro.LoadSnapshots(macroFilePath)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load macro data: "+err.Error())
		return
	}

	scoresMap := macro.BuildScoresByCountry(snapshots)

	writeJSON(w, http.StatusOK, map[string]any{
		"data": scoresMap,
	})
}

// HandleMacroPairSentiment HandleMacroPairSentiment
func (a *API) HandleMacroPairSentiment(w http.ResponseWriter, r *http.Request) {
	base := r.URL.Query().Get("base")
	quote := r.URL.Query().Get("quote")

	if base == "" || quote == "" {
		writeError(w, http.StatusBadRequest, "base and quote are required, e.g. ?base=GBP&quote=USD")
		return
	}

	snapshots, err := macro.LoadSnapshots(macroFilePath)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load macro data: "+err.Error())
		return
	}

	pairSentiment, err := macro.PairSentimentFromSnapshots(snapshots, base, quote)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, pairSentiment)
}

// reuse your existing writeJSON/writeError helpers from handlers.go

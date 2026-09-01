package api

import (
	"net/http"

	"netcompanion/internal/sim"
)

// registerSim expose l'état du simulateur d'équipement à l'UI.
func registerSim(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/sim", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, sim.Current())
	})

	mux.HandleFunc("POST /api/sim/enable", func(w http.ResponseWriter, r *http.Request) {
		info, err := sim.Enable()
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		writeJSON(w, http.StatusOK, info)
	})
}

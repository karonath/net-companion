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
}

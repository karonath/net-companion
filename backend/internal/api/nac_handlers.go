package api

import (
	"errors"
	"net/http"
	"runtime"

	"netcompanion/internal/network/lldp"
	"netcompanion/internal/network/macspoof"
	"netcompanion/internal/network/netinfo"
)

func goosRuntime() string { return runtime.GOOS }

// registerNAC ajoute les routes de contournement NAC sur mux.
func registerNAC(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/nac/lldp", func(w http.ResponseWriter, r *http.Request) {
		iface := ""
		if ifi, err := netinfo.LocalInterface(); err == nil {
			iface = ifi.Name
		}
		writeJSON(w, http.StatusOK, lldp.Capture(iface, 0))
	})

	mux.HandleFunc("POST /api/nac/spoof", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Iface string `json:"iface"`
			MAC   string `json:"mac"`
			Apply bool   `json:"apply"`
		}
		if !decodeJSON(w, r, &body) {
			return
		}
		if body.Iface == "" {
			if ifi, err := netinfo.LocalInterface(); err == nil {
				body.Iface = ifi.Name
			}
		}
		if !body.Apply {
			plan, err := macspoof.BuildPlan(goosRuntime(), body.Iface, body.MAC)
			if err != nil {
				writeError(w, http.StatusBadRequest, err)
				return
			}
			writeJSON(w, http.StatusOK, map[string]any{"dryRun": true, "plan": plan})
			return
		}
		plan, err := macspoof.Apply(body.Iface, body.MAC)
		if err != nil {
			if errors.Is(err, macspoof.ErrElevationRequired) {
				writeJSON(w, http.StatusForbidden, map[string]any{
					"error": err.Error(),
					"plan":  plan,
					"hint":  "relancez Net-Companion en tant qu'administrateur pour appliquer le spoof",
				})
				return
			}
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"applied": true, "plan": plan})
	})
}

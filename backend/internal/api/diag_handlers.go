package api

import (
	"errors"
	"net/http"

	"netcompanion/internal/network/diag"
	"netcompanion/internal/network/netinfo"
)

// registerDiag ajoute les routes de diagnostics de connectivité.
func registerDiag(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/diag", func(w http.ResponseWriter, r *http.Request) {
		gw, _ := netinfo.DefaultGateway()
		writeJSON(w, http.StatusOK, map[string]any{"checks": diag.RunSuite(gw)})
	})

	mux.HandleFunc("POST /api/diag/host", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Host string `json:"host"`
		}
		if !decodeJSON(w, r, &body) {
			return
		}
		if body.Host == "" {
			writeError(w, http.StatusBadRequest, errors.New("host requis"))
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"checks": diag.RunHostSuite(body.Host)})
	})

	mux.HandleFunc("POST /api/diag/port", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Host string `json:"host"`
			Port int    `json:"port"`
		}
		if !decodeJSON(w, r, &body) {
			return
		}
		if body.Host == "" || body.Port <= 0 || body.Port > 65535 {
			writeError(w, http.StatusBadRequest, errors.New("host et port (1-65535) requis"))
			return
		}
		writeJSON(w, http.StatusOK, diag.PortCheck(diag.NetDialer{}, body.Host, body.Port))
	})

	mux.HandleFunc("POST /api/diag/traceroute", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Target string `json:"target"`
		}
		if !decodeJSON(w, r, &body) {
			return
		}
		if body.Target == "" {
			writeError(w, http.StatusBadRequest, errors.New("target requis"))
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"hops": diag.Traceroute(body.Target)})
	})
}

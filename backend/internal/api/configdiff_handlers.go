package api

import (
	"errors"
	"net/http"

	"netcompanion/internal/models"
	"netcompanion/internal/network/configdiff"
	"netcompanion/internal/vault"
)

var errNoSSHCred = errors.New("aucun credential SSH dans le coffre (ajoutez-en un)")

// registerConfigDiff ajoute la route de comparaison de configuration.
func registerConfigDiff(mux *http.ServeMux, v *vault.Vault) {
	mux.HandleFunc("POST /api/configdiff", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			DeviceIP string `json:"deviceIp"`
		}
		if !decodeJSON(w, r, &body) {
			return
		}
		if body.DeviceIP == "" {
			writeError(w, http.StatusBadRequest, errors.New("deviceIp requis"))
			return
		}
		snap, err := v.Snapshot()
		if err != nil {
			writeLocked(w, err)
			return
		}
		if len(snap.SSH) == 0 {
			writeError(w, http.StatusBadRequest, errNoSSHCred)
			return
		}
		lines, running, startup, err := diffViaCredentials(body.DeviceIP, snap.SSH)
		if err != nil {
			writeError(w, http.StatusBadGateway, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"lines": lines, "running": running, "startup": startup,
		})
	})
}

// diffViaCredentials essaie chaque credential SSH jusqu'à récupérer les configs.
func diffViaCredentials(device string, creds []models.SSHCredential) ([]configdiff.DiffLine, string, string, error) {
	var lastErr error
	for _, c := range creds {
		runner, closeFn, err := configdiff.NewSSHRunner(device, c)
		if err != nil {
			lastErr = err
			continue
		}
		running, startup, err := configdiff.Fetch(runner)
		_ = closeFn()
		if err != nil {
			lastErr = err
			continue
		}
		return configdiff.Diff(startup, running), running, startup, nil
	}
	if lastErr == nil {
		lastErr = errors.New("comparaison impossible")
	}
	return nil, "", "", lastErr
}

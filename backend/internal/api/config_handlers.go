package api

import (
	"net/http"
	"os"
	"time"

	"netcompanion/internal/configstore"
	"netcompanion/internal/models"
	"netcompanion/internal/network/configdiff"
	"netcompanion/internal/sim"
	"netcompanion/internal/vault"
)

// registerConfig ajoute les routes de gestion de configuration multi-équipements.
func registerConfig(mux *http.ServeMux, v *vault.Vault) {
	dir := os.Getenv("NC_CONFIG_DIR")
	if dir == "" {
		dir, _ = configstore.DefaultDir()
	}
	store := configstore.NewStore(dir)

	mux.HandleFunc("POST /api/config/backup", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Devices []string `json:"devices"`
		}
		if !decodeJSON(w, r, &body) {
			return
		}
		snap, verr := v.Snapshot() // verr != nil => coffre verrouillé
		type result struct {
			Device     string `json:"device"`
			OK         bool   `json:"ok"`
			Error      string `json:"error,omitempty"`
			DriftCount int    `json:"driftCount"`
		}
		results := make([]result, 0, len(body.Devices))
		for _, device := range body.Devices {
			if device == "" {
				continue
			}
			creds := resolveSSHCreds(device, snap.SSH, verr)
			if len(creds) == 0 {
				results = append(results, result{Device: device, OK: false, Error: "coffre verrouillé ou aucun credential SSH"})
				continue
			}
			running, err := fetchRunning(device, creds)
			if err != nil {
				results = append(results, result{Device: device, OK: false, Error: err.Error()})
				continue
			}
			now := time.Now()
			_ = store.Save(configstore.Snapshot{
				ID: now.Format("20060102-150405.000"), Device: device, Timestamp: now, Running: running,
			})
			drift := 0
			if base, ok := store.Baseline(device); ok {
				for _, l := range configdiff.Diff(base.Running, running) {
					if l.Op != "same" {
						drift++
					}
				}
			}
			results = append(results, result{Device: device, OK: true, DriftCount: drift})
		}
		writeJSON(w, http.StatusOK, map[string]any{"results": results})
	})

	mux.HandleFunc("GET /api/config/devices", func(w http.ResponseWriter, r *http.Request) {
		devs, err := store.ListDevices()
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		writeJSON(w, http.StatusOK, devs)
	})

	mux.HandleFunc("GET /api/config/history", func(w http.ResponseWriter, r *http.Request) {
		device := r.URL.Query().Get("device")
		snaps, _ := store.List(device)
		type meta struct {
			ID        string    `json:"id"`
			Timestamp time.Time `json:"timestamp"`
			Baseline  bool      `json:"baseline"`
		}
		out := make([]meta, 0, len(snaps))
		for _, s := range snaps {
			out = append(out, meta{ID: s.ID, Timestamp: s.Timestamp, Baseline: s.Baseline})
		}
		writeJSON(w, http.StatusOK, out)
	})

	mux.HandleFunc("POST /api/config/baseline", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Device string `json:"device"`
			ID     string `json:"id"`
		}
		if !decodeJSON(w, r, &body) {
			return
		}
		if err := store.SetBaseline(body.Device, body.ID); err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
	})

	mux.HandleFunc("GET /api/config/drift", func(w http.ResponseWriter, r *http.Request) {
		device := r.URL.Query().Get("device")
		base, hasBase := store.Baseline(device)
		latest, hasLatest := store.Latest(device)
		if !hasBase || !hasLatest {
			writeJSON(w, http.StatusOK, map[string]any{"hasBaseline": hasBase, "lines": []configdiff.DiffLine{}})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"hasBaseline": true,
			"lines":       configdiff.Diff(base.Running, latest.Running),
		})
	})
}

// resolveSSHCreds choisit les credentials : démo pour l'adresse du simulateur,
// sinon ceux du coffre (nil si verrouillé).
func resolveSSHCreds(device string, vaultSSH []models.SSHCredential, vaultErr error) []models.SSHCredential {
	if s := sim.Current(); s.Enabled && device == s.SSH {
		return []models.SSHCredential{{Username: sim.DemoSSHUser, Password: sim.DemoSSHPassword}}
	}
	if vaultErr != nil {
		return nil
	}
	return vaultSSH
}

// fetchRunning récupère la running-config via le premier credential SSH qui marche.
func fetchRunning(device string, creds []models.SSHCredential) (string, error) {
	var lastErr error
	for _, c := range creds {
		runner, closeFn, err := configdiff.NewSSHRunner(device, c)
		if err != nil {
			lastErr = err
			continue
		}
		running, _, err := configdiff.Fetch(runner)
		_ = closeFn()
		if err == nil {
			return running, nil
		}
		lastErr = err
	}
	if lastErr == nil {
		lastErr = errNoSSHCred
	}
	return "", lastErr
}

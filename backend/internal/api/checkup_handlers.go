package api

import (
	"encoding/json"
	"net/http"
	"os"
	"sync"
	"time"

	"netcompanion/internal/history"
	"netcompanion/internal/models"
	"netcompanion/internal/network/diag"
	"netcompanion/internal/network/netinfo"
)

// registerCheckup ajoute le check de site 1-clic, l'historique et les rapports.
func registerCheckup(mux *http.ServeMux) {
	dir := os.Getenv("NC_HISTORY_DIR")
	if dir == "" {
		dir, _ = history.DefaultDir()
	}
	store := history.NewStore(dir)

	mux.HandleFunc("POST /api/checkup", func(w http.ResponseWriter, r *http.Request) {
		var meta struct {
			Label string `json:"label"`
			Notes string `json:"notes"`
		}
		if r.Body != nil {
			_ = json.NewDecoder(r.Body).Decode(&meta) // corps optionnel
		}

		ifi, _ := netinfo.LocalInterface()
		gw, _ := netinfo.DefaultGateway()

		var hosts []models.Host
		var checks []diag.Check
		var wg sync.WaitGroup
		wg.Add(2)
		go func() { defer wg.Done(); hosts = runRadar(ifi) }()
		go func() { defer wg.Done(); checks = diag.RunSuite(gw) }()
		wg.Wait()

		now := time.Now()
		snap := history.Snapshot{
			ID: history.NewID(now), Timestamp: now,
			Label: meta.Label, Notes: meta.Notes,
			Interface: ifi, Gateway: gw, Hosts: hosts, Diag: checks,
		}

		prev, hasPrev := store.Latest()
		_ = store.Save(snap)

		var changes *history.Changes
		if hasPrev {
			c := history.Diff(prev, snap)
			changes = &c
		}
		writeJSON(w, http.StatusOK, map[string]any{"snapshot": snap, "changes": changes})
	})

	mux.HandleFunc("GET /api/history", func(w http.ResponseWriter, r *http.Request) {
		metas, err := store.List()
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		writeJSON(w, http.StatusOK, metas)
	})

	mux.HandleFunc("GET /api/history/{id}", func(w http.ResponseWriter, r *http.Request) {
		snap, err := store.Get(r.PathValue("id"))
		if err != nil {
			writeError(w, http.StatusNotFound, err)
			return
		}
		writeJSON(w, http.StatusOK, snap)
	})

	mux.HandleFunc("GET /api/report/{id}", func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		snap, err := store.Get(id)
		if err != nil {
			writeError(w, http.StatusNotFound, err)
			return
		}
		if r.URL.Query().Get("format") == "json" {
			writeJSON(w, http.StatusOK, snap)
			return
		}
		// Calcule les changements vs le snapshot chronologiquement précédent.
		var ch *history.Changes
		if metas, err := store.List(); err == nil {
			for i, m := range metas {
				if m.ID == id && i+1 < len(metas) {
					if prev, err := store.Get(metas[i+1].ID); err == nil {
						c := history.Diff(prev, snap)
						ch = &c
					}
					break
				}
			}
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(history.RenderHTML(snap, ch)))
	})
}

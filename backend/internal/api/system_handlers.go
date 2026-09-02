package api

import (
	"net/http"
	"os"
	"runtime"
	"time"

	"netcompanion/internal/system"
)

// registerSystem expose l'état d'élévation et la relance en administrateur.
func registerSystem(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/system", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{
			"elevated":   system.IsElevated(),
			"os":         runtime.GOOS,
			"canElevate": runtime.GOOS == "windows",
		})
	})

	mux.HandleFunc("POST /api/system/elevate", func(w http.ResponseWriter, r *http.Request) {
		if system.IsElevated() {
			writeJSON(w, http.StatusOK, map[string]any{"already": true})
			return
		}
		// Déclenche l'UAC. Si l'utilisateur annule, une erreur est renvoyée et
		// l'instance courante continue de tourner.
		if err := system.Relaunch(); err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"relaunching": true})
		// Libère le port pour l'instance élevée (qui réessaie de se lier), puis quitte.
		go func() {
			time.Sleep(800 * time.Millisecond)
			os.Exit(0)
		}()
	})
}

// Package server expose le serveur HTTP local de Net-Companion.
package server

import (
	"io/fs"
	"net/http"
	"strconv"
	"strings"

	"netcompanion/internal/api"
	"netcompanion/internal/vault"
)

// Handler sert le frontend embarqué et l'API REST du coffre, protégée par jeton.
func Handler(fsys fs.FS, v *vault.Vault, token string) http.Handler {
	apiMux := http.NewServeMux()
	api.Register(apiMux, v)

	fileServer := http.FileServer(http.FS(fsys))

	top := http.NewServeMux()
	top.Handle("/api/", requireToken(token, apiMux))
	top.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" || r.URL.Path == "/index.html" {
			serveIndex(w, fsys, token)
			return
		}
		fileServer.ServeHTTP(w, r)
	})
	return top
}

// serveIndex sert index.html en y injectant le jeton de session.
func serveIndex(w http.ResponseWriter, fsys fs.FS, token string) {
	data, err := fs.ReadFile(fsys, "index.html")
	if err != nil {
		http.Error(w, "index introuvable", http.StatusInternalServerError)
		return
	}
	html := string(data)
	inject := `<script>window.__NC_TOKEN__=` + strconv.Quote(token) + `</script>`
	if i := strings.Index(html, "</head>"); i >= 0 {
		html = html[:i] + inject + html[i:]
	} else {
		html = inject + html
	}
	// La page ne doit jamais être mise en cache : elle porte le jeton de session
	// et doit toujours refléter la version courante de l'application.
	w.Header().Set("Cache-Control", "no-store, must-revalidate")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(html))
}

// requireToken protège l'API : Origin locale + en-tête X-NC-Token valide.
func requireToken(token string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if o := r.Header.Get("Origin"); o != "" && !allowedOrigin(o) {
			http.Error(w, "origin refusée", http.StatusForbidden)
			return
		}
		// Jeton uniquement via l'en-tête X-NC-Token : jamais dans l'URL (les
		// rapports sont récupérés par fetch puis ouverts via un Blob local).
		provided := r.Header.Get("X-NC-Token")
		if provided != token {
			http.Error(w, "jeton de session invalide", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func allowedOrigin(o string) bool {
	return strings.HasPrefix(o, "http://127.0.0.1:") || strings.HasPrefix(o, "http://localhost:")
}

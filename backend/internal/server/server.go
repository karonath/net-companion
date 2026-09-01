// Package server expose le serveur HTTP local de Net-Companion.
package server

import (
	"io/fs"
	"net/http"
)

// Handler sert le frontend embarqué (SPA) depuis fsys.
func Handler(fsys fs.FS) http.Handler {
	mux := http.NewServeMux()
	mux.Handle("/", http.FileServer(http.FS(fsys)))
	return mux
}

// Package server expose le serveur HTTP local de Net-Companion.
package server

import (
	"io/fs"
	"net/http"

	"netcompanion/internal/api"
	"netcompanion/internal/vault"
)

// Handler sert le frontend embarqué et l'API REST du coffre.
func Handler(fsys fs.FS, v *vault.Vault) http.Handler {
	mux := http.NewServeMux()
	api.Register(mux, v)
	mux.Handle("/", http.FileServer(http.FS(fsys)))
	return mux
}

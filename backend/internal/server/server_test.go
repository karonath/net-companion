package server_test

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"netcompanion/internal/server"
	"netcompanion/internal/vault"
)

func TestHandlerServesIndex(t *testing.T) {
	fsys := fstest.MapFS{
		"index.html": &fstest.MapFile{Data: []byte("<title>Net-Companion</title>")},
	}
	v := vault.New(filepath.Join(t.TempDir(), "vault.dat"))
	h := server.Handler(fsys, v)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("code = %d, want 200", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "Net-Companion") {
		t.Fatalf("body ne contient pas le marqueur: %q", rec.Body.String())
	}
}

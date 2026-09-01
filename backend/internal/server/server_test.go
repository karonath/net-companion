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

const testToken = "test-token-abc"

func newHandler(t *testing.T, index string) http.Handler {
	t.Helper()
	fsys := fstest.MapFS{
		"index.html": &fstest.MapFile{Data: []byte(index)},
	}
	v := vault.New(filepath.Join(t.TempDir(), "vault.dat"))
	return server.Handler(fsys, v, testToken)
}

func TestServesIndexInjectsToken(t *testing.T) {
	h := newHandler(t, "<html><head><title>Net-Companion</title></head><body></body></html>")
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("code = %d, want 200", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "Net-Companion") {
		t.Fatal("index ne contient pas le marqueur")
	}
	if !strings.Contains(body, `window.__NC_TOKEN__="test-token-abc"`) {
		t.Fatalf("jeton non injecté: %q", body)
	}
}

func TestAPIRequiresToken(t *testing.T) {
	h := newHandler(t, "<head></head>")

	// sans jeton → 401
	req := httptest.NewRequest(http.MethodGet, "/api/vault/status", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("sans jeton code = %d, want 401", rec.Code)
	}

	// avec jeton → 200
	req = httptest.NewRequest(http.MethodGet, "/api/vault/status", nil)
	req.Header.Set("X-NC-Token", testToken)
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("avec jeton code = %d, want 200", rec.Code)
	}
}

func TestAPIRejectsForeignOrigin(t *testing.T) {
	h := newHandler(t, "<head></head>")
	req := httptest.NewRequest(http.MethodGet, "/api/vault/status", nil)
	req.Header.Set("X-NC-Token", testToken)
	req.Header.Set("Origin", "http://evil.example.com")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("origin étrangère code = %d, want 403", rec.Code)
	}
}

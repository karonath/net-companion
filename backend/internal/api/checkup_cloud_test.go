package api_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"netcompanion/internal/api"
	"netcompanion/internal/vault"
)

// Vérifie qu'après un checkup, si le cloud est configuré, un POST part vers
// l'endpoint d'ingestion avec la clé d'API, le clientId et un snapshotId non vide.
func TestCheckupPushesToCloudWhenConfigured(t *testing.T) {
	received := make(chan map[string]any, 1)
	ingest := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Api-Key") != "secret" {
			t.Errorf("clé d'API manquante: %q", r.Header.Get("X-Api-Key"))
		}
		b, _ := io.ReadAll(r.Body)
		var m map[string]any
		_ = json.Unmarshal(b, &m)
		w.WriteHeader(http.StatusAccepted)
		received <- m
	}))
	defer ingest.Close()

	t.Setenv("NC_CLOUD_URL", ingest.URL)
	t.Setenv("NC_CLOUD_KEY", "secret")
	t.Setenv("NC_CLIENT_ID", "acme")
	t.Setenv("NC_SITE_ID", "paris-hq")
	t.Setenv("NC_HISTORY_DIR", filepath.Join(t.TempDir(), "history"))

	v := vault.New(filepath.Join(t.TempDir(), "vault.dat"))
	mux := http.NewServeMux()
	api.Register(mux, v)
	srv := httptest.NewServer(mux)
	defer srv.Close()

	resp, err := http.Post(srv.URL+"/api/checkup", "application/json", nil)
	if err != nil {
		t.Fatalf("checkup: %v", err)
	}
	resp.Body.Close()

	select {
	case m := <-received:
		if m["clientId"] != "acme" {
			t.Fatalf("clientId attendu acme: %v", m["clientId"])
		}
		if s, ok := m["snapshotId"].(string); !ok || s == "" {
			t.Fatalf("snapshotId manquant: %v", m)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("aucune remontée cloud reçue dans le délai")
	}
}

// Vérifie que si le cloud est activé (URL+clé) mais que le rattachement
// client/site est incomplet, aucune remontée ne part (le snapshot reste local).
func TestCheckupSkipsCloudWhenIncomplete(t *testing.T) {
	received := make(chan map[string]any, 1)
	ingest := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		var m map[string]any
		_ = json.Unmarshal(b, &m)
		w.WriteHeader(http.StatusAccepted)
		received <- m
	}))
	defer ingest.Close()

	t.Setenv("NC_CLOUD_URL", ingest.URL)
	t.Setenv("NC_CLOUD_KEY", "secret")
	t.Setenv("NC_HISTORY_DIR", filepath.Join(t.TempDir(), "history"))

	v := vault.New(filepath.Join(t.TempDir(), "vault.dat"))
	mux := http.NewServeMux()
	api.Register(mux, v)
	srv := httptest.NewServer(mux)
	defer srv.Close()

	resp, err := http.Post(srv.URL+"/api/checkup", "application/json", nil)
	if err != nil {
		t.Fatalf("checkup: %v", err)
	}
	resp.Body.Close()

	select {
	case <-received:
		t.Fatal("aucune remontée ne devait partir")
	case <-time.After(500 * time.Millisecond):
		// ok : pas de remontée cloud.
	}
}

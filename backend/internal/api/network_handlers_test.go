package api_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"netcompanion/internal/api"
	"netcompanion/internal/vault"
)

func netServer(t *testing.T) (*httptest.Server, *vault.Vault) {
	t.Helper()
	v := vault.New(filepath.Join(t.TempDir(), "vault.dat"))
	mux := http.NewServeMux()
	api.Register(mux, v)
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv, v
}

func TestNetworkInfoReturnsInterface(t *testing.T) {
	srv, _ := netServer(t)
	resp, err := http.Get(srv.URL + "/api/network/info")
	if err != nil {
		t.Fatalf("GET info: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("code = %d, want 200 (machine a une interface active)", resp.StatusCode)
	}
	var body map[string]any
	_ = json.NewDecoder(resp.Body).Decode(&body)
	if _, ok := body["interface"]; !ok {
		t.Fatal("réponse sans champ interface")
	}
}

func TestPortfinderLockedReturns423(t *testing.T) {
	srv, _ := netServer(t)
	// coffre non initialisé => Snapshot renvoie ErrLocked => 423
	resp, err := http.Post(srv.URL+"/api/network/portfinder", "application/json",
		bytes.NewBufferString(`{"deviceIp":"192.0.2.1"}`))
	if err != nil {
		t.Fatalf("POST portfinder: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusLocked {
		t.Fatalf("code = %d, want 423 (coffre verrouillé)", resp.StatusCode)
	}
}

func TestPortfinderUnlockedNoCommunityReturns400(t *testing.T) {
	srv, _ := netServer(t)
	http.Post(srv.URL+"/api/vault/init", "application/json", bytes.NewBufferString(`{"pin":"1234"}`))
	resp, err := http.Post(srv.URL+"/api/network/portfinder", "application/json",
		bytes.NewBufferString(`{"deviceIp":"192.0.2.1"}`))
	if err != nil {
		t.Fatalf("POST portfinder: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("code = %d, want 400 (aucune community)", resp.StatusCode)
	}
}

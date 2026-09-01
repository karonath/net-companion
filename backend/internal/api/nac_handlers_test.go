package api_test

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

func TestLLDPGracefulUnavailable(t *testing.T) {
	srv, _ := netServer(t)
	resp, err := http.Get(srv.URL + "/api/nac/lldp")
	if err != nil {
		t.Fatalf("GET lldp: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("code = %d, want 200 (dégradation propre, pas d'erreur HTTP)", resp.StatusCode)
	}
	var res struct {
		Available bool   `json:"available"`
		Reason    string `json:"reason"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&res)
	if res.Available {
		t.Fatal("sans -tags npcap, LLDP doit être indisponible")
	}
	if res.Reason == "" {
		t.Fatal("raison manquante")
	}
}

func TestSpoofDryRunReturnsPlan(t *testing.T) {
	srv, _ := netServer(t)
	resp, err := http.Post(srv.URL+"/api/nac/spoof", "application/json",
		strings.NewReader(`{"iface":"Wi-Fi","mac":"00:1a:2b:3c:4d:5e","apply":false}`))
	if err != nil {
		t.Fatalf("POST spoof: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("code = %d, want 200", resp.StatusCode)
	}
	var body map[string]any
	_ = json.NewDecoder(resp.Body).Decode(&body)
	if body["dryRun"] != true {
		t.Fatalf("dryRun attendu true : %+v", body)
	}
	if _, ok := body["plan"]; !ok {
		t.Fatal("plan manquant dans la réponse dry-run")
	}
}

func TestSpoofInvalidMAC(t *testing.T) {
	srv, _ := netServer(t)
	resp, err := http.Post(srv.URL+"/api/nac/spoof", "application/json",
		strings.NewReader(`{"iface":"Wi-Fi","mac":"pas-une-mac","apply":false}`))
	if err != nil {
		t.Fatalf("POST spoof: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("code = %d, want 400 (MAC invalide)", resp.StatusCode)
	}
}

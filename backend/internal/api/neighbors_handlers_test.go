package api_test

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

func TestNeighborsDemoReturnsNeighbor(t *testing.T) {
	srv, _ := netServer(t)
	resp, err := http.Post(srv.URL+"/api/network/neighbors", "application/json",
		strings.NewReader(`{"demo":true}`))
	if err != nil {
		t.Fatalf("POST neighbors: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("code = %d, want 200", resp.StatusCode)
	}
	var body struct {
		Neighbors []map[string]any `json:"neighbors"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&body)
	if len(body.Neighbors) == 0 {
		t.Fatal("aucun voisin en démo")
	}
	found := false
	for _, n := range body.Neighbors {
		if n["remoteSysName"] == "SW-CORE-02" {
			found = true
		}
	}
	if !found {
		t.Fatalf("SW-CORE-02 absent: %+v", body.Neighbors)
	}
}

func TestNeighborsLockedReturns423(t *testing.T) {
	srv, _ := netServer(t)
	resp, err := http.Post(srv.URL+"/api/network/neighbors", "application/json",
		strings.NewReader(`{"deviceIp":"192.0.2.1"}`))
	if err != nil {
		t.Fatalf("POST: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusLocked {
		t.Fatalf("code = %d, want 423", resp.StatusCode)
	}
}

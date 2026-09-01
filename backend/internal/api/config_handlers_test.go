package api_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"netcompanion/internal/api"
	"netcompanion/internal/models"
	"netcompanion/internal/sim"
	"netcompanion/internal/vault"
)

func configServer(t *testing.T) (*httptest.Server, *vault.Vault) {
	t.Helper()
	t.Setenv("NC_CONFIG_DIR", filepath.Join(t.TempDir(), "configs"))
	v := vault.New(filepath.Join(t.TempDir(), "vault.dat"))
	mux := http.NewServeMux()
	api.Register(mux, v)
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv, v
}

func TestConfigBackupBaselineDriftViaSimulator(t *testing.T) {
	srv, v := configServer(t)
	if err := v.Init("1234"); err != nil {
		t.Fatalf("init: %v", err)
	}
	if _, err := v.AddSSH(models.SSHCredential{Label: "demo", Username: sim.DemoSSHUser, Password: sim.DemoSSHPassword}); err != nil {
		t.Fatalf("AddSSH: %v", err)
	}
	stop, addr, err := sim.StartSSH("127.0.0.1:0")
	if err != nil {
		t.Fatalf("StartSSH: %v", err)
	}
	defer stop()

	// backup
	resp := postJSON(t, srv.URL+"/api/config/backup", `{"devices":["`+addr+`"]}`)
	var b struct {
		Results []struct {
			Device string `json:"device"`
			OK     bool   `json:"ok"`
			Error  string `json:"error"`
		} `json:"results"`
	}
	json.NewDecoder(resp.Body).Decode(&b)
	resp.Body.Close()
	if len(b.Results) != 1 || !b.Results[0].OK {
		t.Fatalf("backup échoué: %+v", b.Results)
	}

	// devices
	resp, _ = http.Get(srv.URL + "/api/config/devices")
	var devs []map[string]any
	json.NewDecoder(resp.Body).Decode(&devs)
	resp.Body.Close()
	if len(devs) != 1 {
		t.Fatalf("devices = %d, want 1", len(devs))
	}

	// history -> récupère l'id
	resp, _ = http.Get(srv.URL + "/api/config/history?device=" + addr)
	var hist []struct {
		ID string `json:"id"`
	}
	json.NewDecoder(resp.Body).Decode(&hist)
	resp.Body.Close()
	if len(hist) < 1 {
		t.Fatalf("history vide")
	}

	// set baseline
	resp = postJSON(t, srv.URL+"/api/config/baseline", `{"device":"`+addr+`","id":"`+hist[0].ID+`"}`)
	resp.Body.Close()

	// 2e backup : config identique => drift vide
	resp = postJSON(t, srv.URL+"/api/config/backup", `{"devices":["`+addr+`"]}`)
	var b2 struct {
		Results []struct {
			DriftCount int `json:"driftCount"`
		} `json:"results"`
	}
	json.NewDecoder(resp.Body).Decode(&b2)
	resp.Body.Close()
	if b2.Results[0].DriftCount != 0 {
		t.Fatalf("drift = %d, want 0 (config stable)", b2.Results[0].DriftCount)
	}

	// drift endpoint
	resp, _ = http.Get(srv.URL + "/api/config/drift?device=" + addr)
	var d struct {
		HasBaseline bool             `json:"hasBaseline"`
		Lines       []map[string]any `json:"lines"`
	}
	json.NewDecoder(resp.Body).Decode(&d)
	resp.Body.Close()
	if !d.HasBaseline {
		t.Fatal("drift devrait indiquer une baseline")
	}
}

func postJSON(t *testing.T, url, body string) *http.Response {
	t.Helper()
	resp, err := http.Post(url, "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatalf("POST %s: %v", url, err)
	}
	return resp
}

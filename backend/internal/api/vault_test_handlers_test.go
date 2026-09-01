package api_test

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"netcompanion/internal/models"
	"netcompanion/internal/sim"
)

func TestVaultTestLockedReturns423(t *testing.T) {
	srv, _ := netServer(t)
	resp, err := http.Post(srv.URL+"/api/vault/test", "application/json",
		strings.NewReader(`{"type":"ssh","id":"x","deviceIp":"127.0.0.1:22"}`))
	if err != nil {
		t.Fatalf("POST: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusLocked {
		t.Fatalf("code = %d, want 423", resp.StatusCode)
	}
}

func TestVaultTestUnknownIDReturns404(t *testing.T) {
	srv, v := netServer(t)
	if err := v.Init("1234"); err != nil {
		t.Fatalf("init: %v", err)
	}
	resp, err := http.Post(srv.URL+"/api/vault/test", "application/json",
		strings.NewReader(`{"type":"ssh","id":"inconnu","deviceIp":"127.0.0.1:22"}`))
	if err != nil {
		t.Fatalf("POST: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("code = %d, want 404", resp.StatusCode)
	}
}

func TestVaultTestSSHSuccessViaSimulator(t *testing.T) {
	srv, v := netServer(t)
	if err := v.Init("1234"); err != nil {
		t.Fatalf("init: %v", err)
	}
	cred, err := v.AddSSH(models.SSHCredential{Label: "demo", Username: sim.DemoSSHUser, Password: sim.DemoSSHPassword})
	if err != nil {
		t.Fatalf("AddSSH: %v", err)
	}

	stop, addr, err := sim.StartSSH("127.0.0.1:0")
	if err != nil {
		t.Fatalf("StartSSH: %v", err)
	}
	defer stop()

	resp, err := http.Post(srv.URL+"/api/vault/test", "application/json",
		strings.NewReader(`{"type":"ssh","id":"`+cred.ID+`","deviceIp":"`+addr+`"}`))
	if err != nil {
		t.Fatalf("POST: %v", err)
	}
	defer resp.Body.Close()
	var res struct {
		OK     bool   `json:"ok"`
		Detail string `json:"detail"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&res)
	if !res.OK {
		t.Fatalf("test SSH échoué contre le simulateur: %s", res.Detail)
	}
}

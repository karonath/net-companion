package api_test

import (
	"net/http"
	"strings"
	"testing"
)

func TestConfigDiffLockedReturns423(t *testing.T) {
	srv, _ := netServer(t)
	resp, err := http.Post(srv.URL+"/api/configdiff", "application/json",
		strings.NewReader(`{"deviceIp":"192.0.2.1"}`))
	if err != nil {
		t.Fatalf("POST configdiff: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusLocked {
		t.Fatalf("code = %d, want 423 (coffre verrouillé)", resp.StatusCode)
	}
}

func TestConfigDiffUnlockedNoSSHReturns400(t *testing.T) {
	srv, _ := netServer(t)
	http.Post(srv.URL+"/api/vault/init", "application/json", strings.NewReader(`{"pin":"1234"}`))
	resp, err := http.Post(srv.URL+"/api/configdiff", "application/json",
		strings.NewReader(`{"deviceIp":"192.0.2.1"}`))
	if err != nil {
		t.Fatalf("POST configdiff: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("code = %d, want 400 (aucun credential SSH)", resp.StatusCode)
	}
}

func TestConfigDiffMissingDeviceReturns400(t *testing.T) {
	srv, _ := netServer(t)
	http.Post(srv.URL+"/api/vault/init", "application/json", strings.NewReader(`{"pin":"1234"}`))
	resp, err := http.Post(srv.URL+"/api/configdiff", "application/json",
		strings.NewReader(`{}`))
	if err != nil {
		t.Fatalf("POST configdiff: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("code = %d, want 400 (deviceIp requis)", resp.StatusCode)
	}
}

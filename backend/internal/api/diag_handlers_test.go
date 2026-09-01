package api_test

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strings"
	"testing"
)

func TestDiagPortOpenAndClosed(t *testing.T) {
	srv, _ := netServer(t)

	// listener local = port ouvert connu
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer ln.Close()
	openPort := ln.Addr().(*net.TCPAddr).Port

	resp, err := http.Post(srv.URL+"/api/diag/port", "application/json",
		strings.NewReader(fmt.Sprintf(`{"host":"127.0.0.1","port":%d}`, openPort)))
	if err != nil {
		t.Fatalf("POST port: %v", err)
	}
	var c struct{ Status string }
	_ = json.NewDecoder(resp.Body).Decode(&c)
	resp.Body.Close()
	if c.Status != "ok" {
		t.Fatalf("port ouvert status = %q, want ok", c.Status)
	}

	// port très probablement fermé
	resp, _ = http.Post(srv.URL+"/api/diag/port", "application/json",
		strings.NewReader(`{"host":"127.0.0.1","port":1}`))
	_ = json.NewDecoder(resp.Body).Decode(&c)
	resp.Body.Close()
	if c.Status != "fail" {
		t.Fatalf("port fermé status = %q, want fail", c.Status)
	}
}

func TestDiagPortBadRequest(t *testing.T) {
	srv, _ := netServer(t)
	resp, err := http.Post(srv.URL+"/api/diag/port", "application/json",
		strings.NewReader(`{"host":"","port":0}`))
	if err != nil {
		t.Fatalf("POST: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("code = %d, want 400", resp.StatusCode)
	}
}

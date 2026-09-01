package api_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"netcompanion/internal/api"
	"netcompanion/internal/models"
	"netcompanion/internal/vault"
)

func newServer(t *testing.T) *httptest.Server {
	t.Helper()
	v := vault.New(filepath.Join(t.TempDir(), "vault.dat"))
	mux := http.NewServeMux()
	api.Register(mux, v)
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

func post(t *testing.T, url string, body any) *http.Response {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		_ = json.NewEncoder(&buf).Encode(body)
	}
	resp, err := http.Post(url, "application/json", &buf)
	if err != nil {
		t.Fatalf("POST %s: %v", url, err)
	}
	return resp
}

func TestStatusInitUnlockFlow(t *testing.T) {
	srv := newServer(t)

	resp, err := http.Get(srv.URL + "/api/vault/status")
	if err != nil {
		t.Fatalf("GET status: %v", err)
	}
	var st vault.Status
	_ = json.NewDecoder(resp.Body).Decode(&st)
	resp.Body.Close()
	if st.Initialized || st.Unlocked {
		t.Fatalf("status initial = %+v", st)
	}

	resp = post(t, srv.URL+"/api/vault/init", map[string]string{"pin": "1234"})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("init code = %d, want 200", resp.StatusCode)
	}
	resp.Body.Close()

	// 2e init -> 409
	resp = post(t, srv.URL+"/api/vault/init", map[string]string{"pin": "1234"})
	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("init x2 code = %d, want 409", resp.StatusCode)
	}
	resp.Body.Close()
}

func TestUnlockWrongPINReturns401(t *testing.T) {
	srv := newServer(t)
	post(t, srv.URL+"/api/vault/init", map[string]string{"pin": "1234"}).Body.Close()
	post(t, srv.URL+"/api/vault/lock", nil).Body.Close()

	resp := post(t, srv.URL+"/api/vault/unlock", map[string]string{"pin": "0000"})
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("unlock mauvais PIN code = %d, want 401", resp.StatusCode)
	}
	resp.Body.Close()
}

func TestSNMPCrudAndLockGuard(t *testing.T) {
	srv := newServer(t)
	post(t, srv.URL+"/api/vault/init", map[string]string{"pin": "1234"}).Body.Close()

	// Ajout
	resp := post(t, srv.URL+"/api/vault/secrets/snmp", models.SNMPCredential{Label: "prod", Community: "public", Version: "v2c"})
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("add snmp code = %d, want 201", resp.StatusCode)
	}
	var created models.SNMPCredential
	_ = json.NewDecoder(resp.Body).Decode(&created)
	resp.Body.Close()
	if created.ID == "" {
		t.Fatal("id manquant")
	}

	// Liste
	resp, _ = http.Get(srv.URL + "/api/vault/secrets/snmp")
	var list []models.SNMPCredential
	_ = json.NewDecoder(resp.Body).Decode(&list)
	resp.Body.Close()
	if len(list) != 1 {
		t.Fatalf("liste = %d, want 1", len(list))
	}

	// Suppression
	req, _ := http.NewRequest(http.MethodDelete, srv.URL+"/api/vault/secrets/snmp/"+created.ID, nil)
	resp, _ = http.DefaultClient.Do(req)
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("delete code = %d, want 204", resp.StatusCode)
	}
	resp.Body.Close()

	// Après lock -> 423 sur ajout
	post(t, srv.URL+"/api/vault/lock", nil).Body.Close()
	resp = post(t, srv.URL+"/api/vault/secrets/snmp", models.SNMPCredential{Label: "x", Community: "c"})
	if resp.StatusCode != http.StatusLocked {
		t.Fatalf("add verrouillé code = %d, want 423", resp.StatusCode)
	}
	resp.Body.Close()
}

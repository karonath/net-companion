package api_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"netcompanion/internal/api"
	"netcompanion/internal/vault"
)

func checkupServer(t *testing.T) *httptest.Server {
	t.Helper()
	t.Setenv("NC_HISTORY_DIR", filepath.Join(t.TempDir(), "history"))
	v := vault.New(filepath.Join(t.TempDir(), "vault.dat"))
	mux := http.NewServeMux()
	api.Register(mux, v)
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

func TestCheckupAndHistoryAndReport(t *testing.T) {
	srv := checkupServer(t)

	// 1er check
	resp, err := http.Post(srv.URL+"/api/checkup", "application/json", nil)
	if err != nil {
		t.Fatalf("POST checkup: %v", err)
	}
	var body struct {
		Snapshot struct {
			ID   string           `json:"id"`
			Diag []map[string]any `json:"diag"`
		} `json:"snapshot"`
		Changes any `json:"changes"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&body)
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("checkup code = %d", resp.StatusCode)
	}
	if body.Snapshot.ID == "" || len(body.Snapshot.Diag) == 0 {
		t.Fatalf("snapshot incomplet: %+v", body.Snapshot)
	}

	// historique
	resp, _ = http.Get(srv.URL + "/api/history")
	var metas []map[string]any
	_ = json.NewDecoder(resp.Body).Decode(&metas)
	resp.Body.Close()
	if len(metas) < 1 {
		t.Fatalf("history vide")
	}

	// rapport HTML
	resp, _ = http.Get(srv.URL + "/api/report/" + body.Snapshot.ID)
	ct := resp.Header.Get("Content-Type")
	buf := make([]byte, 64)
	n, _ := resp.Body.Read(buf)
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK || !strings.Contains(ct, "text/html") {
		t.Fatalf("report code=%d ct=%q", resp.StatusCode, ct)
	}
	if !strings.Contains(strings.ToLower(string(buf[:n])), "html") {
		t.Fatalf("rapport ne ressemble pas à du HTML: %q", string(buf[:n]))
	}
}

package cloud

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"netcompanion/internal/history"
)

func TestNoopWhenNotComplete(t *testing.T) {
	p := NewPusher(Config{URL: "https://api", Key: "k"}, nil) // pas de client/site
	if err := p.Push(context.Background(), history.Snapshot{ID: "x"}); err != nil {
		t.Fatalf("noop ne doit jamais renvoyer d'erreur: %v", err)
	}
}

func TestHTTPPusherPostsPayload(t *testing.T) {
	var gotKey, gotCT string
	var body Payload
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotKey = r.Header.Get("X-Api-Key")
		gotCT = r.Header.Get("Content-Type")
		b, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(b, &body)
		w.WriteHeader(http.StatusAccepted)
	}))
	defer srv.Close()

	cfg := Config{URL: srv.URL, Key: "secret", ClientID: "acme", SiteID: "hq", TechID: "t1"}
	p := NewPusher(cfg, nil)
	err := p.Push(context.Background(), history.Snapshot{ID: "20260917-140300"})
	if err != nil {
		t.Fatalf("push a échoué: %v", err)
	}
	if gotKey != "secret" {
		t.Fatalf("clé d'API absente/incorrecte: %q", gotKey)
	}
	if gotCT != "application/json" {
		t.Fatalf("content-type: %q", gotCT)
	}
	if body.ClientID != "acme" || body.SnapshotID != "20260917-140300" {
		t.Fatalf("payload mal transmis: %+v", body)
	}
}

func TestHTTPPusherErrorsOn500(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	p := NewPusher(Config{URL: srv.URL, Key: "k", ClientID: "acme", SiteID: "hq"}, nil)
	if err := p.Push(context.Background(), history.Snapshot{ID: "x"}); err == nil {
		t.Fatal("un code 500 doit produire une erreur")
	}
}

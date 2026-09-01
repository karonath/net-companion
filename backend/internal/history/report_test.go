package history

import (
	"strings"
	"testing"
	"time"

	"netcompanion/internal/models"
	"netcompanion/internal/network/diag"
)

func TestRenderHTML(t *testing.T) {
	s := Snapshot{
		ID:        "20260101-120000",
		Timestamp: time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC),
		Interface: models.InterfaceInfo{Name: "Wi-Fi", IPv4: "192.168.1.15"},
		Gateway:   "192.168.1.1",
		Hosts: []models.Host{
			{IP: "192.168.1.1", Vendor: "Ingram", MAC: "aa:bb:cc:dd:ee:ff"},
			{IP: "192.168.1.20", Vendor: "Nintendo"},
		},
		Diag: []diag.Check{{Name: "Accès Internet", Status: "ok", Detail: "connecté"}},
	}
	html := RenderHTML(s, nil)

	if !strings.Contains(html, "<html") {
		t.Fatal("document HTML attendu")
	}
	if !strings.Contains(html, "192.168.1.1") || !strings.Contains(html, "Nintendo") {
		t.Fatal("le rapport doit lister les hôtes")
	}
	if !strings.Contains(html, "Accès Internet") {
		t.Fatal("le rapport doit lister les checks")
	}
	if !strings.Contains(html, "192.168.1.15") {
		t.Fatal("le rapport doit mentionner l'interface locale")
	}
}

func TestRenderHTMLWithChanges(t *testing.T) {
	s := Snapshot{ID: "x", Timestamp: time.Now()}
	ch := &Changes{
		HostsAdded:   []models.Host{{IP: "192.168.1.99"}},
		HostsRemoved: []models.Host{{IP: "192.168.1.50"}},
	}
	html := RenderHTML(s, ch)
	if !strings.Contains(html, "192.168.1.99") || !strings.Contains(html, "192.168.1.50") {
		t.Fatal("la section changements doit apparaître")
	}
}

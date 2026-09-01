package history

import (
	"testing"
	"time"

	"netcompanion/internal/models"
	"netcompanion/internal/network/diag"
)

func snap(id string, ts time.Time, hosts []models.Host, gw string, checks []diag.Check) Snapshot {
	return Snapshot{ID: id, Timestamp: ts, Gateway: gw, Hosts: hosts, Diag: checks}
}

func TestStoreSaveListGet(t *testing.T) {
	st := NewStore(t.TempDir())
	s1 := snap("20260101-100000", time.Now().Add(-time.Hour),
		[]models.Host{{IP: "192.168.1.10"}}, "192.168.1.1", nil)
	s2 := snap("20260101-110000", time.Now(),
		[]models.Host{{IP: "192.168.1.10"}, {IP: "192.168.1.20"}}, "192.168.1.1", nil)

	if err := st.Save(s1); err != nil {
		t.Fatalf("Save s1: %v", err)
	}
	if err := st.Save(s2); err != nil {
		t.Fatalf("Save s2: %v", err)
	}

	metas, err := st.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(metas) != 2 {
		t.Fatalf("List = %d, want 2", len(metas))
	}
	// tri décroissant : le plus récent en premier
	if metas[0].ID != "20260101-110000" {
		t.Fatalf("ordre = %s en premier, want 20260101-110000", metas[0].ID)
	}
	if metas[0].HostCount != 2 {
		t.Fatalf("HostCount = %d, want 2", metas[0].HostCount)
	}

	got, err := st.Get("20260101-100000")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if len(got.Hosts) != 1 || got.Hosts[0].IP != "192.168.1.10" {
		t.Fatalf("Get s1 hosts = %+v", got.Hosts)
	}
}

func TestDiffDetectsChanges(t *testing.T) {
	prev := snap("a", time.Now().Add(-time.Hour),
		[]models.Host{{IP: "192.168.1.10"}, {IP: "192.168.1.20"}},
		"192.168.1.1",
		[]diag.Check{{Name: "Accès Internet", Status: "ok"}})
	cur := snap("b", time.Now(),
		[]models.Host{{IP: "192.168.1.10"}, {IP: "192.168.1.30"}},
		"192.168.1.254",
		[]diag.Check{{Name: "Accès Internet", Status: "fail"}})

	ch := Diff(prev, cur)
	if len(ch.HostsAdded) != 1 || ch.HostsAdded[0].IP != "192.168.1.30" {
		t.Fatalf("HostsAdded = %+v", ch.HostsAdded)
	}
	if len(ch.HostsRemoved) != 1 || ch.HostsRemoved[0].IP != "192.168.1.20" {
		t.Fatalf("HostsRemoved = %+v", ch.HostsRemoved)
	}
	if ch.GatewayFrom != "192.168.1.1" || ch.GatewayTo != "192.168.1.254" {
		t.Fatalf("gateway change = %s -> %s", ch.GatewayFrom, ch.GatewayTo)
	}
	if len(ch.ChecksChanged) != 1 || ch.ChecksChanged[0].To != "fail" {
		t.Fatalf("ChecksChanged = %+v", ch.ChecksChanged)
	}
}

func TestDiffNoChange(t *testing.T) {
	s := snap("a", time.Now(), []models.Host{{IP: "192.168.1.10"}}, "192.168.1.1", nil)
	ch := Diff(s, s)
	if len(ch.HostsAdded) != 0 || len(ch.HostsRemoved) != 0 || ch.GatewayFrom != "" {
		t.Fatalf("changements inattendus: %+v", ch)
	}
}

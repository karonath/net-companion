package configstore

import (
	"testing"
	"time"
)

func TestSaveListLatest(t *testing.T) {
	st := NewStore(t.TempDir())
	dev := "192.168.1.1"
	if err := st.Save(Snapshot{ID: "20260101-100000", Device: dev, Timestamp: time.Now().Add(-time.Hour), Running: "hostname A"}); err != nil {
		t.Fatalf("Save1: %v", err)
	}
	if err := st.Save(Snapshot{ID: "20260101-110000", Device: dev, Timestamp: time.Now(), Running: "hostname B"}); err != nil {
		t.Fatalf("Save2: %v", err)
	}
	snaps, err := st.List(dev)
	if err != nil || len(snaps) != 2 {
		t.Fatalf("List = %d (%v)", len(snaps), err)
	}
	latest, ok := st.Latest(dev)
	if !ok || latest.Running != "hostname B" {
		t.Fatalf("Latest = %+v ok=%v", latest, ok)
	}
}

func TestBaseline(t *testing.T) {
	st := NewStore(t.TempDir())
	dev := "10.0.0.1"
	st.Save(Snapshot{ID: "a", Device: dev, Timestamp: time.Now().Add(-time.Hour), Running: "cfg-a"})
	st.Save(Snapshot{ID: "b", Device: dev, Timestamp: time.Now(), Running: "cfg-b"})

	if _, ok := st.Baseline(dev); ok {
		t.Fatal("aucune baseline attendue au départ")
	}
	if err := st.SetBaseline(dev, "a"); err != nil {
		t.Fatalf("SetBaseline: %v", err)
	}
	base, ok := st.Baseline(dev)
	if !ok || base.ID != "a" || base.Running != "cfg-a" {
		t.Fatalf("Baseline = %+v ok=%v", base, ok)
	}
	// changer de baseline : un seul baseline actif
	if err := st.SetBaseline(dev, "b"); err != nil {
		t.Fatalf("SetBaseline b: %v", err)
	}
	base, _ = st.Baseline(dev)
	if base.ID != "b" {
		t.Fatalf("baseline active = %s, want b", base.ID)
	}
	sa, _ := st.Get(dev, "a")
	if sa.Baseline {
		t.Fatal("l'ancienne baseline 'a' devrait être désactivée")
	}
}

func TestListDevices(t *testing.T) {
	st := NewStore(t.TempDir())
	st.Save(Snapshot{ID: "a", Device: "192.168.1.1", Timestamp: time.Now(), Running: "x"})
	st.Save(Snapshot{ID: "b", Device: "192.168.1.2", Timestamp: time.Now(), Running: "y"})
	st.SetBaseline("192.168.1.1", "a")

	devs, err := st.ListDevices()
	if err != nil || len(devs) != 2 {
		t.Fatalf("ListDevices = %d (%v)", len(devs), err)
	}
	byDev := map[string]DeviceMeta{}
	for _, d := range devs {
		byDev[d.Device] = d
	}
	if !byDev["192.168.1.1"].HasBaseline {
		t.Fatal("192.168.1.1 devrait avoir une baseline")
	}
	if byDev["192.168.1.2"].HasBaseline {
		t.Fatal("192.168.1.2 ne devrait pas avoir de baseline")
	}
}

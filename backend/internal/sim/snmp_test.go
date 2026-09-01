package sim_test

import (
	"testing"

	"netcompanion/internal/network/portfinder"
	"netcompanion/internal/sim"
)

func TestDemoSNMPClientLocates(t *testing.T) {
	loc, err := portfinder.Locate(sim.DemoSNMPClient(), sim.DemoMAC)
	if err != nil {
		t.Fatalf("Locate contre le simulateur: %v", err)
	}
	if loc.PortIfName != "GigabitEthernet1/0/5" {
		t.Fatalf("port = %q, want GigabitEthernet1/0/5", loc.PortIfName)
	}
	if loc.VLAN != 42 {
		t.Fatalf("vlan = %d, want 42", loc.VLAN)
	}
	if loc.Device != "SW-DEMO-01" {
		t.Fatalf("device = %q, want SW-DEMO-01", loc.Device)
	}
	if loc.Sentence == "" {
		t.Fatal("phrase manquante")
	}
}

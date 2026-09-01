package netinfo

import (
	"testing"

	"netcompanion/internal/models"
)

func TestSelectInterfacePrefersUpIPv4NonLoopback(t *testing.T) {
	cands := []models.InterfaceInfo{
		{Name: "lo", MAC: "", IPv4: "127.0.0.1", CIDR: "127.0.0.1/8"},
		{Name: "eth0", MAC: "aa:bb:cc:dd:ee:ff", IPv4: "192.168.1.10", CIDR: "192.168.1.10/24"},
	}
	got, err := selectInterface(cands)
	if err != nil {
		t.Fatalf("selectInterface: %v", err)
	}
	if got.Name != "eth0" {
		t.Fatalf("interface choisie = %q, want eth0", got.Name)
	}
}

func TestSelectInterfaceErrWhenNoCandidate(t *testing.T) {
	if _, err := selectInterface(nil); err == nil {
		t.Fatal("attendu une erreur si aucune interface exploitable")
	}
}

package netinfo

import (
	"testing"

	"netcompanion/internal/models"
)

func TestSelectInterfaceAnchorsOnGateway(t *testing.T) {
	cands := []models.InterfaceInfo{
		{Name: "Bluetooth", MAC: "aa:aa:aa:aa:aa:aa", IPv4: "169.254.55.111", CIDR: "169.254.55.111/16"},
		{Name: "Tailscale", MAC: "bb:bb:bb:bb:bb:bb", IPv4: "100.109.20.3", CIDR: "100.109.20.3/32"},
		{Name: "Wi-Fi", MAC: "cc:cc:cc:cc:cc:cc", IPv4: "192.168.1.15", CIDR: "192.168.1.15/24"},
	}
	got, err := selectInterface(cands, "192.168.1.1")
	if err != nil {
		t.Fatalf("selectInterface: %v", err)
	}
	if got.Name != "Wi-Fi" {
		t.Fatalf("interface = %q, want Wi-Fi (celle qui possède la passerelle)", got.Name)
	}
}

func TestSelectInterfaceSkipsLinkLocalWithoutGateway(t *testing.T) {
	cands := []models.InterfaceInfo{
		{Name: "Bluetooth", MAC: "aa:aa:aa:aa:aa:aa", IPv4: "169.254.55.111", CIDR: "169.254.55.111/16"},
		{Name: "Ethernet", MAC: "cc:cc:cc:cc:cc:cc", IPv4: "10.0.0.5", CIDR: "10.0.0.5/24"},
	}
	got, err := selectInterface(cands, "") // passerelle inconnue
	if err != nil {
		t.Fatalf("selectInterface: %v", err)
	}
	if got.Name != "Ethernet" {
		t.Fatalf("interface = %q, want Ethernet (le link-local doit être écarté)", got.Name)
	}
}

func TestSelectInterfaceErrWhenNoCandidate(t *testing.T) {
	if _, err := selectInterface(nil, ""); err == nil {
		t.Fatal("attendu une erreur si aucune interface exploitable")
	}
}

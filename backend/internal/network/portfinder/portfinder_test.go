package portfinder

import "testing"

func TestMacToOIDSuffix(t *testing.T) {
	got, err := macToOIDSuffix("00:1a:2b:3c:4d:5e")
	if err != nil {
		t.Fatalf("macToOIDSuffix: %v", err)
	}
	if got != "0.26.43.60.77.94" {
		t.Fatalf("suffixe = %q, want 0.26.43.60.77.94", got)
	}
	if _, err := macToOIDSuffix("pas-une-mac"); err == nil {
		t.Fatal("attendu une erreur sur MAC invalide")
	}
}

// mockSNMP simule un switch : MAC 00:1a:2b:3c:4d:5e sur bridge port 7,
// ifIndex 10001, ifName Gi1/0/5, VLAN 42, sysName SW-CORE-01.
type mockSNMP struct{}

func (mockSNMP) Get(oid string) (string, bool) {
	m := map[string]string{
		"1.3.6.1.2.1.17.4.3.1.2.0.26.43.60.77.94": "7",
		"1.3.6.1.2.1.17.1.4.1.2.7":                 "10001",
		"1.3.6.1.2.1.31.1.1.1.1.10001":             "GigabitEthernet1/0/5",
		"1.3.6.1.2.1.1.5.0":                        "SW-CORE-01",
	}
	v, ok := m[oid]
	return v, ok
}

func (mockSNMP) WalkStrings(root string) map[string]string {
	if root == "1.3.6.1.2.1.17.7.1.2.2.1.2" {
		return map[string]string{
			"1.3.6.1.2.1.17.7.1.2.2.1.2.42.0.26.43.60.77.94": "7",
		}
	}
	return nil
}

func TestLocateResolvesPortAndVLAN(t *testing.T) {
	loc, err := Locate(mockSNMP{}, "00:1a:2b:3c:4d:5e")
	if err != nil {
		t.Fatalf("Locate: %v", err)
	}
	if loc.BridgePort != 7 || loc.IfIndex != 10001 {
		t.Fatalf("port/ifindex = %+v", loc)
	}
	if loc.PortIfName != "GigabitEthernet1/0/5" {
		t.Fatalf("ifName = %q", loc.PortIfName)
	}
	if loc.VLAN != 42 {
		t.Fatalf("vlan = %d, want 42", loc.VLAN)
	}
	if loc.Device != "SW-CORE-01" {
		t.Fatalf("device = %q", loc.Device)
	}
	if loc.Sentence == "" {
		t.Fatal("phrase en langage naturel manquante")
	}
}

func TestLocateMACNotFound(t *testing.T) {
	_, err := Locate(mockSNMP{}, "ff:ff:ff:ff:ff:ff")
	if err == nil {
		t.Fatal("attendu une erreur si la MAC est absente de la table")
	}
}

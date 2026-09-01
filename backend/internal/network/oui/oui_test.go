package oui

import "testing"

func TestVendorFromTable(t *testing.T) {
	table := map[string]string{
		"0000C0": "Western Digital",
		"00000C": "Cisco Systems, Inc",
	}
	cases := map[string]string{
		"00:00:0c:12:34:56": "Cisco Systems, Inc",
		"00-00-C0-aa-bb-cc": "Western Digital",
		"de:ad:be:ef:00:01": "", // inconnu
		"invalide":          "",
	}
	for mac, want := range cases {
		if got := vendorFromTable(mac, table); got != want {
			t.Errorf("vendorFromTable(%q) = %q, want %q", mac, got, want)
		}
	}
}

func TestVendorLoadsEmbeddedTable(t *testing.T) {
	// Cisco 00:00:0C est un OUI historique stable présent dans la base IEEE.
	if got := Vendor("00:00:0c:11:22:33"); got == "" {
		t.Fatal("la base OUI embarquée devrait résoudre le préfixe Cisco 00:00:0C")
	}
}

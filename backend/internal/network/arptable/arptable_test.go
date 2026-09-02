package arptable

import "testing"

// fakeSNMP implémente portfinder.SNMPClient pour les tests.
type fakeSNMP struct {
	walk map[string]string
}

func (f fakeSNMP) Get(string) (string, bool)          { return "", false }
func (f fakeSNMP) WalkStrings(string) map[string]string { return f.walk }

func TestIPFromIndex(t *testing.T) {
	cases := map[string]string{
		"3.192.168.1.42": "192.168.1.42", // ifIndex.a.b.c.d
		"10.10.0.0.5":    "10.0.0.5",
		"1.2.3":          "", // moins de 4 composants -> pas d'IP
	}
	for idx, want := range cases {
		if got := ipFromIndex(idx); got != want {
			t.Errorf("ipFromIndex(%q) = %q, want %q", idx, got, want)
		}
	}
}

func TestMacFromBytes(t *testing.T) {
	if got := macFromBytes([]byte{0x10, 0xe9, 0x92, 0x21, 0x22, 0x30}); got != "10:e9:92:21:22:30" {
		t.Errorf("macFromBytes = %q", got)
	}
	if macFromBytes([]byte{1, 2, 3}) != "" {
		t.Error("longueur != 6 doit donner ''")
	}
}

func TestFromSNMP(t *testing.T) {
	mac := string([]byte{0x68, 0x3a, 0x48, 0x95, 0x7d, 0x54})
	c := fakeSNMP{walk: map[string]string{
		oidIPNetToMediaPhys + ".3.192.168.1.10": mac,
	}}
	entries := FromSNMP(c)
	if len(entries) != 1 {
		t.Fatalf("attendu 1 entrée, obtenu %d : %+v", len(entries), entries)
	}
	if entries[0].IP != "192.168.1.10" || entries[0].MAC != "68:3a:48:95:7d:54" {
		t.Errorf("entrée = %+v", entries[0])
	}
}

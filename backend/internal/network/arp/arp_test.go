package arp

import "testing"

func TestParseProcArp(t *testing.T) {
	raw := "IP address       HW type     Flags       HW address            Mask     Device\n" +
		"192.168.1.1      0x1         0x2         10:e9:92:21:22:30     *        eth0\n" +
		"192.168.1.99     0x1         0x0         00:00:00:00:00:00     *        eth0\n" + // incomplet -> ignoré
		"192.168.1.42     0x1         0x2         68:3a:48:95:7d:54     *        eth0\n"
	got := parseProcArp(raw)
	if len(got) != 2 {
		t.Fatalf("got %d voisins, want 2 : %+v", len(got), got)
	}
	if got[0].IP != "192.168.1.1" || got[0].MAC != "10:e9:92:21:22:30" {
		t.Fatalf("voisin[0] = %+v", got[0])
	}
}

func TestParseWindows(t *testing.T) {
	raw := `
Interface : 192.168.1.10 --- 0x5
  Adresse Internet      Adresse physique      Type
  192.168.1.1           aa-bb-cc-dd-ee-ff     dynamique
  192.168.1.42          11-22-33-44-55-66     dynamique
  224.0.0.22            01-00-5e-00-00-16     statique
`
	got := parseWindows(raw)
	if len(got) != 3 {
		t.Fatalf("got %d voisins, want 3", len(got))
	}
	if got[0].IP != "192.168.1.1" || got[0].MAC != "aa:bb:cc:dd:ee:ff" {
		t.Fatalf("voisin[0] = %+v", got[0])
	}
}

func TestParseLinux(t *testing.T) {
	raw := "192.168.1.1 dev eth0 lladdr aa:bb:cc:dd:ee:ff REACHABLE\n" +
		"192.168.1.42 dev eth0 lladdr 11:22:33:44:55:66 STALE\n" +
		"192.168.1.99 dev eth0 FAILED\n"
	got := parseLinux(raw)
	if len(got) != 2 {
		t.Fatalf("got %d voisins, want 2 (FAILED sans MAC ignoré)", len(got))
	}
	if got[1].IP != "192.168.1.42" || got[1].MAC != "11:22:33:44:55:66" {
		t.Fatalf("voisin[1] = %+v", got[1])
	}
}

package neighbors

import "testing"

// mock implémente portfinder.SNMPClient avec des walks canned.
type mock struct {
	walks map[string]map[string]string
}

func (m mock) Get(string) (string, bool) { return "", false }
func (m mock) WalkStrings(root string) map[string]string {
	if w, ok := m.walks[root]; ok {
		return w
	}
	return nil
}

func TestFromSNMP_LLDP(t *testing.T) {
	// index composite LLDP : <timeMark>.<localPort>.<remIndex> = 0.5.1
	m := mock{walks: map[string]map[string]string{
		oidLldpRemSysName:   {oidLldpRemSysName + ".0.5.1": "SW-CORE-02"},
		oidLldpRemPortID:    {oidLldpRemPortID + ".0.5.1": "GigabitEthernet1/0/24"},
		oidLldpRemChassisID: {oidLldpRemChassisID + ".0.5.1": "aa:bb:cc:dd:ee:01"},
	}}
	ns := FromSNMP(m)
	if len(ns) != 1 {
		t.Fatalf("got %d voisins, want 1", len(ns))
	}
	n := ns[0]
	if n.RemoteSysName != "SW-CORE-02" || n.RemotePortID != "GigabitEthernet1/0/24" {
		t.Fatalf("voisin = %+v", n)
	}
	if n.LocalPort != "5" {
		t.Fatalf("localPort = %q, want 5", n.LocalPort)
	}
	if n.Source != "lldp" {
		t.Fatalf("source = %q, want lldp", n.Source)
	}
}

func TestFromSNMP_CDP(t *testing.T) {
	// index CDP : <ifIndex>.<devIndex> = 7.1
	m := mock{walks: map[string]map[string]string{
		oidCdpDeviceID:   {oidCdpDeviceID + ".7.1": "SW-ACCESS-09"},
		oidCdpDevicePort: {oidCdpDevicePort + ".7.1": "FastEthernet0/1"},
	}}
	ns := FromSNMP(m)
	if len(ns) != 1 || ns[0].RemoteSysName != "SW-ACCESS-09" || ns[0].Source != "cdp" {
		t.Fatalf("voisins CDP = %+v", ns)
	}
	if ns[0].LocalPort != "7" {
		t.Fatalf("localPort CDP = %q, want 7", ns[0].LocalPort)
	}
}

func TestFromSNMP_Empty(t *testing.T) {
	if ns := FromSNMP(mock{walks: map[string]map[string]string{}}); len(ns) != 0 {
		t.Fatalf("attendu aucun voisin, got %+v", ns)
	}
}

package sysinfo

import "testing"

func TestClassify(t *testing.T) {
	cases := []struct {
		descr, services, want string
	}{
		{"Cisco IOS Software, Catalyst 2960 Software", "6", "switch"},
		{"Cisco Adaptive Security Appliance (ASA)", "4", "pare-feu"},
		{"FortiGate-60F", "78", "pare-feu"},
		{"HP ETHERNET MULTI-ENVIRONMENT, JETDIRECT", "", "imprimante"},
		{"Aruba AP-315 (Rev A)", "", "point d'accès"},
		{"VMware ESXi 7.0.3 build-19193900", "72", "hyperviseur"},
		{"Synology DiskStation DS920+", "72", "NAS / stockage"},
		{"Linux server 5.15.0 x86_64", "72", "serveur"},
		{"MikroTik RouterOS RB4011", "78", "routeur / box"},
		{"APC Web/SNMP Management Card", "72", "onduleur"},
		{"Polycom SoundPoint IP 650", "", "téléphone VoIP"},
		{"Siemens SIMATIC S7-1200", "", "automate / OT"},
		{"quelque chose d'inconnu", "", ""},
	}
	for _, c := range cases {
		if got := Classify(c.descr, c.services); got != c.want {
			t.Errorf("Classify(%q, %q) = %q, want %q", c.descr, c.services, got, c.want)
		}
	}
}

func TestRefineL2L3(t *testing.T) {
	if refineL2L3("2", "") != "switch" { // L2 seul
		t.Error("sysServices=2 doit donner switch")
	}
	if refineL2L3("4", "") != "routeur / box" { // L3
		t.Error("sysServices=4 doit donner routeur / box")
	}
	if refineL2L3("72", "") != "" { // L4+L7 sans L2/L3 -> fallback
		t.Error("sysServices=72 sans L2/L3 doit donner le fallback")
	}
	if refineL2L3("abc", "serveur") != "serveur" { // invalide -> fallback
		t.Error("sysServices invalide doit donner le fallback")
	}
}

// fakeSNMP implémente portfinder.SNMPClient.
type fakeSNMP struct{ vals map[string]string }

func (f fakeSNMP) Get(oid string) (string, bool) { v, ok := f.vals[oid]; return v, ok }
func (f fakeSNMP) WalkStrings(string) map[string]string { return nil }

func TestFromSNMP(t *testing.T) {
	c := fakeSNMP{vals: map[string]string{
		oidSysDescr:    "Cisco IOS Software, Catalyst 2960",
		oidSysName:     "SW-CORE-01",
		oidSysServices: "6",
	}}
	info, ok := FromSNMP(c)
	if !ok || info.SysName != "SW-CORE-01" || info.DeviceType != "switch" {
		t.Fatalf("FromSNMP = %+v (ok=%v)", info, ok)
	}
	// Pas de sysDescr -> pas SNMP.
	if _, ok := FromSNMP(fakeSNMP{vals: map[string]string{}}); ok {
		t.Error("sans sysDescr, FromSNMP doit renvoyer false")
	}
}

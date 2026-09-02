package diag

import "testing"

func TestScanHostPorts(t *testing.T) {
	d := fakeDialer{ok: map[string]bool{
		"192.168.1.10:80":  true,
		"192.168.1.10:443": true,
	}}
	open := scanHostPorts(d, "192.168.1.10", []int{22, 80, 443, 445})
	if len(open) != 2 || open[0] != 80 || open[1] != 443 {
		t.Fatalf("ports ouverts = %v, want [80 443]", open)
	}
}

func TestPortList(t *testing.T) {
	got := portList([]int{22, 80, 8009})
	want := "22 (SSH), 80 (HTTP), 8009 (Chromecast)"
	if got != want {
		t.Fatalf("portList = %q, want %q", got, want)
	}
}

package portfinder

import "testing"

func TestSplitTargetPort(t *testing.T) {
	cases := []struct {
		in       string
		wantHost string
		wantPort uint16
	}{
		{"192.168.1.1", "192.168.1.1", 161},
		{"127.0.0.1:1161", "127.0.0.1", 1161},
		{"10.0.0.1:0", "10.0.0.1", 161},   // port invalide -> défaut
		{"10.0.0.1:abc", "10.0.0.1", 161}, // port non numérique -> défaut
	}
	for _, c := range cases {
		h, p := splitTargetPort(c.in)
		if h != c.wantHost || p != c.wantPort {
			t.Errorf("splitTargetPort(%q) = %q,%d ; want %q,%d", c.in, h, p, c.wantHost, c.wantPort)
		}
	}
}

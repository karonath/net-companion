package radar

import (
	"sort"
	"sync"
	"testing"
)

func TestHostsInCIDR24(t *testing.T) {
	hosts, err := HostsInCIDR("192.168.1.10/24", 1000)
	if err != nil {
		t.Fatalf("HostsInCIDR: %v", err)
	}
	// /24 => 254 hôtes utilisables (exclut .0 réseau et .255 broadcast).
	if len(hosts) != 254 {
		t.Fatalf("got %d hôtes, want 254", len(hosts))
	}
	if hosts[0] != "192.168.1.1" || hosts[253] != "192.168.1.254" {
		t.Fatalf("bornes = %s..%s", hosts[0], hosts[len(hosts)-1])
	}
}

func TestHostsInCIDRRespectsMax(t *testing.T) {
	hosts, err := HostsInCIDR("10.0.0.0/8", 500)
	if err != nil {
		t.Fatalf("HostsInCIDR: %v", err)
	}
	if len(hosts) != 500 {
		t.Fatalf("got %d, want cap 500", len(hosts))
	}
}

// fakeProber déclare vivants les IP d'un ensemble donné.
type fakeProber struct {
	alive map[string]bool
	mu    sync.Mutex
	calls int
}

func (f *fakeProber) Probe(ip string) bool {
	f.mu.Lock()
	f.calls++
	f.mu.Unlock()
	return f.alive[ip]
}

func TestSweepReturnsAliveHosts(t *testing.T) {
	hosts := []string{"192.168.1.1", "192.168.1.2", "192.168.1.3", "192.168.1.4"}
	fp := &fakeProber{alive: map[string]bool{"192.168.1.1": true, "192.168.1.3": true}}
	got := Sweep(hosts, fp, 8)
	sort.Strings(got)
	if len(got) != 2 || got[0] != "192.168.1.1" || got[1] != "192.168.1.3" {
		t.Fatalf("vivants = %v, want [192.168.1.1 192.168.1.3]", got)
	}
	if fp.calls != 4 {
		t.Fatalf("probes = %d, want 4 (tous testés)", fp.calls)
	}
}

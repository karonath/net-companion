package diag

import (
	"errors"
	"net"
	"testing"
	"time"
)

// fakeDialer : réussit pour les adresses dans ok, échoue sinon.
type fakeDialer struct{ ok map[string]bool }

func (f fakeDialer) Dial(network, addr string, _ time.Duration) (net.Conn, error) {
	if f.ok[addr] {
		c1, c2 := net.Pipe()
		_ = c2.Close()
		return c1, nil
	}
	return nil, errors.New("refused")
}

type fakeResolver struct {
	addrs []string
	err   error
}

func (f fakeResolver) LookupHost(string) ([]string, error) { return f.addrs, f.err }

func TestCheckGatewayReachable(t *testing.T) {
	d := fakeDialer{ok: map[string]bool{"192.168.1.1:53": true}}
	c := CheckGateway(d, "192.168.1.1")
	if c.Status != StatusOK {
		t.Fatalf("status = %q, want ok (%s)", c.Status, c.Detail)
	}
}

func TestCheckGatewayUnreachable(t *testing.T) {
	d := fakeDialer{ok: map[string]bool{}}
	c := CheckGateway(d, "192.168.1.1")
	if c.Status != StatusFail {
		t.Fatalf("status = %q, want fail", c.Status)
	}
}

func TestCheckDNS(t *testing.T) {
	ok := CheckDNS(fakeResolver{addrs: []string{"93.184.216.34"}}, "example.com")
	if ok.Status != StatusOK {
		t.Fatalf("dns ok status = %q", ok.Status)
	}
	bad := CheckDNS(fakeResolver{err: errors.New("no such host")}, "example.com")
	if bad.Status != StatusFail {
		t.Fatalf("dns fail status = %q", bad.Status)
	}
}

func TestCheckInternet(t *testing.T) {
	d := fakeDialer{ok: map[string]bool{"1.1.1.1:443": true}}
	c := CheckInternet(d, []string{"1.1.1.1:443", "8.8.8.8:443"})
	if c.Status != StatusOK {
		t.Fatalf("internet status = %q", c.Status)
	}
	down := CheckInternet(fakeDialer{ok: map[string]bool{}}, []string{"1.1.1.1:443"})
	if down.Status != StatusFail {
		t.Fatalf("internet down status = %q", down.Status)
	}
}

func TestMeasureLatency(t *testing.T) {
	d := fakeDialer{ok: map[string]bool{"192.168.1.1:443": true}}
	c := MeasureLatency(d, "192.168.1.1:443", 3)
	if c.Status != StatusOK {
		t.Fatalf("latency status = %q (%s)", c.Status, c.Detail)
	}
}

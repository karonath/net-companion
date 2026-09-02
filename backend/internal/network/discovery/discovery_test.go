package discovery

import (
	"net"
	"testing"

	"golang.org/x/net/dns/dnsmessage"
)

func TestParseSSDP(t *testing.T) {
	raw := "HTTP/1.1 200 OK\r\n" +
		"CACHE-CONTROL: max-age=1800\r\n" +
		"LOCATION: http://192.168.1.1:5000/desc.xml\r\n" +
		"SERVER: Linux/3.14 UPnP/1.0 Synology/DSM\r\n" +
		"ST: upnp:rootdevice\r\n\r\n"
	if got := parseSSDP(raw); got != "Linux/3.14 UPnP/1.0 Synology/DSM" {
		t.Fatalf("SERVER = %q", got)
	}
}

// construit une réponse mDNS avec un enregistrement A
func buildMDNSResponse(t *testing.T, host, ip string) []byte {
	t.Helper()
	b := dnsmessage.NewBuilder(nil, dnsmessage.Header{Response: true, Authoritative: true})
	if err := b.StartAnswers(); err != nil {
		t.Fatal(err)
	}
	name := dnsmessage.MustNewName(host)
	var a [4]byte
	copy(a[:], net.ParseIP(ip).To4())
	if err := b.AResource(
		dnsmessage.ResourceHeader{Name: name, Type: dnsmessage.TypeA, Class: dnsmessage.ClassINET, TTL: 120},
		dnsmessage.AResource{A: a},
	); err != nil {
		t.Fatal(err)
	}
	buf, err := b.Finish()
	if err != nil {
		t.Fatal(err)
	}
	return buf
}

func TestParseMDNS(t *testing.T) {
	data := buildMDNSResponse(t, "MacBook-de-Thomas.local.", "192.168.1.42")
	out := map[string]string{}
	parseMDNS(data, out)
	if out["192.168.1.42"] != "MacBook-de-Thomas" {
		t.Fatalf("mDNS = %+v, want 192.168.1.42 -> MacBook-de-Thomas", out)
	}
}

func TestMdnsQueryBuilds(t *testing.T) {
	q, err := mdnsQuery()
	if err != nil || len(q) == 0 {
		t.Fatalf("mdnsQuery: %v (len %d)", err, len(q))
	}
	// doit être un message DNS parsable
	var p dnsmessage.Parser
	if _, err := p.Start(q); err != nil {
		t.Fatalf("requête mDNS non parsable: %v", err)
	}
}

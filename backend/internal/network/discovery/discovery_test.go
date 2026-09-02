package discovery

import (
	"encoding/binary"
	"net"
	"testing"

	"golang.org/x/net/dns/dnsmessage"
)

func TestParseSSDPHeaders(t *testing.T) {
	raw := "HTTP/1.1 200 OK\r\n" +
		"CACHE-CONTROL: max-age=1800\r\n" +
		"LOCATION: http://192.168.1.1:5000/desc.xml\r\n" +
		"SERVER: Linux/3.14 UPnP/1.0 Synology/DSM\r\n" +
		"ST: upnp:rootdevice\r\n\r\n"
	server, location := parseSSDPHeaders(raw)
	if server != "Linux/3.14 UPnP/1.0 Synology/DSM" {
		t.Fatalf("SERVER = %q", server)
	}
	if location != "http://192.168.1.1:5000/desc.xml" {
		t.Fatalf("LOCATION = %q", location)
	}
}

func TestParseUPnPXML(t *testing.T) {
	xml := `<?xml version="1.0"?><root><device>
		<friendlyName>Salon TV</friendlyName>
		<manufacturer>Samsung Electronics</manufacturer>
		<modelName>QE55Q80</modelName>
		<modelNumber>2021</modelNumber>
		<deviceType>urn:schemas-upnp-org:device:MediaRenderer:1</deviceType>
	</device></root>`
	f := parseUPnPXML(xml)
	if f["friendlyName"] != "Salon TV" || f["manufacturer"] != "Samsung Electronics" || f["modelName"] != "QE55Q80" {
		t.Fatalf("champs UPnP = %+v", f)
	}
	d := deviceFromUPnP("192.168.1.20", "", f)
	if d.Name != "Salon TV" || d.Manufacturer != "Samsung Electronics" || d.Model != "QE55Q80 2021" {
		t.Fatalf("device UPnP = %+v", d)
	}
}

// buildMDNSChromecast fabrique une réponse mDNS complète (PTR+SRV+TXT+A).
func buildMDNSChromecast(t *testing.T) []byte {
	t.Helper()
	b := dnsmessage.NewBuilder(nil, dnsmessage.Header{Response: true, Authoritative: true})
	if err := b.StartAnswers(); err != nil {
		t.Fatal(err)
	}
	inst := dnsmessage.MustNewName("Living-Room._googlecast._tcp.local.")
	svc := dnsmessage.MustNewName("_googlecast._tcp.local.")
	host := dnsmessage.MustNewName("abcd.local.")

	must := func(err error) {
		if err != nil {
			t.Fatal(err)
		}
	}
	must(b.PTRResource(
		dnsmessage.ResourceHeader{Name: svc, Type: dnsmessage.TypePTR, Class: dnsmessage.ClassINET, TTL: 120},
		dnsmessage.PTRResource{PTR: inst},
	))
	must(b.SRVResource(
		dnsmessage.ResourceHeader{Name: inst, Type: dnsmessage.TypeSRV, Class: dnsmessage.ClassINET, TTL: 120},
		dnsmessage.SRVResource{Priority: 0, Weight: 0, Port: 8009, Target: host},
	))
	must(b.TXTResource(
		dnsmessage.ResourceHeader{Name: inst, Type: dnsmessage.TypeTXT, Class: dnsmessage.ClassINET, TTL: 120},
		dnsmessage.TXTResource{TXT: []string{"md=Chromecast Ultra", "fn=Living Room TV"}},
	))
	var a [4]byte
	copy(a[:], net.ParseIP("192.168.1.50").To4())
	must(b.AResource(
		dnsmessage.ResourceHeader{Name: host, Type: dnsmessage.TypeA, Class: dnsmessage.ClassINET, TTL: 120},
		dnsmessage.AResource{A: a},
	))
	buf, err := b.Finish()
	if err != nil {
		t.Fatal(err)
	}
	return buf
}

func TestMDNSCorrelation(t *testing.T) {
	acc := newMDNSAcc()
	acc.parse(buildMDNSChromecast(t))
	devs := acc.devices()
	var d *Device
	for i := range devs {
		if devs[i].IP == "192.168.1.50" {
			d = &devs[i]
		}
	}
	if d == nil {
		t.Fatalf("appareil 192.168.1.50 absent: %+v", devs)
	}
	if d.Name != "Living Room TV" {
		t.Errorf("Name = %q, want Living Room TV", d.Name)
	}
	if d.Model != "Chromecast Ultra" {
		t.Errorf("Model = %q, want Chromecast Ultra", d.Model)
	}
	if d.DeviceType != "TV / média" {
		t.Errorf("DeviceType = %q, want TV / média", d.DeviceType)
	}
	if d.Hostname != "abcd" {
		t.Errorf("Hostname = %q, want abcd", d.Hostname)
	}
}

func TestMdnsQueryBuilds(t *testing.T) {
	q, err := mdnsQuery()
	if err != nil || len(q) == 0 {
		t.Fatalf("mdnsQuery: %v (len %d)", err, len(q))
	}
	var p dnsmessage.Parser
	if _, err := p.Start(q); err != nil {
		t.Fatalf("requête mDNS non parsable: %v", err)
	}
}

// nbnsEntry encode une entrée de nom NetBIOS (15 octets + suffixe + flags).
func nbnsEntry(name string, suffix byte, group bool) []byte {
	var raw [15]byte
	copy(raw[:], name)
	for i := len(name); i < 15; i++ {
		raw[i] = ' '
	}
	e := append([]byte{}, raw[:]...)
	e = append(e, suffix)
	var flags uint16
	if group {
		flags |= 0x8000
	}
	var fb [2]byte
	binary.BigEndian.PutUint16(fb[:], flags)
	return append(e, fb[:]...)
}

func buildNBNSResponse() []byte {
	pkt := []byte{0x00, 0x00, 0x84, 0x00, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00}
	pkt = append(pkt, encodeNetBIOSName("*")...)
	pkt = append(pkt, 0x00, 0x21, 0x00, 0x01) // type NBSTAT, class IN
	pkt = append(pkt, 0x00, 0x00, 0x00, 0x00) // TTL
	rdata := []byte{0x02}                      // 2 noms
	rdata = append(rdata, nbnsEntry("MYPC", 0x00, false)...)
	rdata = append(rdata, nbnsEntry("WORKGROUP", 0x00, true)...)
	var rl [2]byte
	binary.BigEndian.PutUint16(rl[:], uint16(len(rdata)))
	pkt = append(pkt, rl[:]...)
	pkt = append(pkt, rdata...)
	return pkt
}

func TestParseNBNS(t *testing.T) {
	host, group := parseNBNS(buildNBNSResponse())
	if host != "MYPC" {
		t.Errorf("host = %q, want MYPC", host)
	}
	if group != "WORKGROUP" {
		t.Errorf("group = %q, want WORKGROUP", group)
	}
}

func TestParseHTTPHead(t *testing.T) {
	resp := "HTTP/1.1 200 OK\r\nServer: lighttpd/1.4.35\r\nContent-Type: text/html\r\n\r\n" +
		"<html><head><title>FRITZ!Box 7590</title></head></html>"
	server, title := parseHTTPHead(resp)
	if server != "lighttpd/1.4.35" {
		t.Errorf("server = %q", server)
	}
	if title != "FRITZ!Box 7590" {
		t.Errorf("title = %q", title)
	}
}

func TestParseSSHBanner(t *testing.T) {
	if v := parseSSHBanner("SSH-2.0-OpenSSH_8.9p1 Ubuntu-3ubuntu0.1"); v != "OpenSSH_8.9p1 Ubuntu-3ubuntu0.1" {
		t.Errorf("ssh = %q", v)
	}
}

func TestMerge(t *testing.T) {
	in := []Device{
		{IP: "192.168.1.5", Name: "printer", Sources: []string{"mdns"}, Services: []string{"_ipp._tcp"}},
		{IP: "192.168.1.5", Manufacturer: "HP", Sources: []string{"banner"}, Services: []string{"tcp/9100"}},
	}
	out := Merge(in)
	if len(out) != 1 {
		t.Fatalf("attendu 1 appareil fusionné, obtenu %d", len(out))
	}
	d := out[0]
	if d.Name != "printer" || d.Manufacturer != "HP" {
		t.Errorf("fusion champs = %+v", d)
	}
	if len(d.Sources) != 2 || len(d.Services) != 2 {
		t.Errorf("fusion listes = %+v", d)
	}
}

func TestInferType(t *testing.T) {
	cases := []struct{ hint, want string }{
		{"samsung-washer.home", "électroménager"},
		{"Nintendo Co.,Ltd", "console de jeu"},
		{"livebox.home Sagemcom", "routeur / box"},
		{"HP LaserJet Pro", "imprimante"},
		{"Chromecast Ultra", "TV / média"},
		{"iPhone de Thomas", "smartphone / tablette"},
		{"quelque-chose-inconnu", ""},
	}
	for _, c := range cases {
		if got := InferType(c.hint); got != c.want {
			t.Errorf("InferType(%q) = %q, want %q", c.hint, got, c.want)
		}
	}
}

func TestClassifyService(t *testing.T) {
	cases := map[string]string{
		"_googlecast._tcp": "TV / média",
		"tcp/9100":         "imprimante",
		"_ssh._tcp":        "ordinateur",
		"tcp/445":          "ordinateur",
		"_hap._tcp":        "objet connecté",
	}
	for svc, want := range cases {
		if got := classifyService(svc); got != want {
			t.Errorf("classifyService(%q) = %q, want %q", svc, got, want)
		}
	}
}

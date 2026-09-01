package lldp

import "testing"

// tlv encode un TLV LLDP (type 7 bits, longueur 9 bits) + valeur.
func tlv(typ int, val []byte) []byte {
	length := len(val)
	b := []byte{byte(typ<<1) | byte(length>>8), byte(length & 0xFF)}
	return append(b, val...)
}

func buildFrame() []byte {
	dst := []byte{0x01, 0x80, 0xc2, 0x00, 0x00, 0x0e}
	src := []byte{0x00, 0x00, 0x0c, 0x11, 0x22, 0x33}
	eth := append(append(dst, src...), 0x88, 0xcc)

	chassis := tlv(1, append([]byte{4}, 0x00, 0x00, 0x0c, 0xaa, 0xbb, 0xcc)) // subtype 4 = MAC
	port := tlv(2, append([]byte{5}, []byte("Gi1/0/5")...))                  // subtype 5 = ifname
	ttl := tlv(3, []byte{0x00, 0x78})
	sysname := tlv(5, []byte("SW-CORE-01"))
	end := tlv(0, nil)

	lldpu := append(append(append(append(chassis, port...), ttl...), sysname...), end...)
	return append(eth, lldpu...)
}

func TestParseEthernetLLDP(t *testing.T) {
	n, err := ParseEthernet(buildFrame())
	if err != nil {
		t.Fatalf("ParseEthernet: %v", err)
	}
	if n.SystemName != "SW-CORE-01" {
		t.Fatalf("systemName = %q", n.SystemName)
	}
	if n.PortID != "Gi1/0/5" {
		t.Fatalf("portID = %q", n.PortID)
	}
	if n.ChassisID != "00:00:0c:aa:bb:cc" {
		t.Fatalf("chassisID = %q", n.ChassisID)
	}
}

func TestParseEthernetRejectsNonLLDP(t *testing.T) {
	frame := make([]byte, 20)
	frame[12], frame[13] = 0x08, 0x00 // IPv4, pas LLDP
	if _, err := ParseEthernet(frame); err == nil {
		t.Fatal("attendu une erreur pour ethertype non-LLDP")
	}
}

func TestCaptureStubUnavailable(t *testing.T) {
	res := Capture("eth0", 0)
	if res.Available {
		t.Fatal("le stub par défaut doit être indisponible")
	}
	if res.Reason == "" {
		t.Fatal("le stub doit fournir une raison")
	}
}

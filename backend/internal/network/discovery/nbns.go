package discovery

import (
	"encoding/binary"
	"net"
	"strings"
	"time"
)

// nbstatQuery construit une requête NetBIOS « node status » pour le nom "*",
// qui demande à l'hôte de lister ses noms NetBIOS (nom machine + groupe).
func nbstatQuery() []byte {
	pkt := []byte{
		0x00, 0x00, // ID transaction
		0x00, 0x00, // flags (query)
		0x00, 0x01, // QDCOUNT = 1
		0x00, 0x00, // ANCOUNT
		0x00, 0x00, // NSCOUNT
		0x00, 0x00, // ARCOUNT
	}
	pkt = append(pkt, encodeNetBIOSName("*")...)
	pkt = append(pkt,
		0x00, 0x21, // type = NBSTAT
		0x00, 0x01, // class = IN
	)
	return pkt
}

// encodeNetBIOSName applique l'encodage « premier niveau » : le nom (16 octets,
// complété par des nuls) est découpé en demi-octets, chacun décalé de 'A'.
func encodeNetBIOSName(name string) []byte {
	var raw [16]byte
	copy(raw[:], name)
	if name == "*" {
		raw[0] = '*' // le reste reste nul
	}
	out := make([]byte, 0, 34)
	out = append(out, 0x20) // longueur = 32 caractères encodés
	for _, b := range raw {
		out = append(out, 'A'+(b>>4), 'A'+(b&0x0F))
	}
	out = append(out, 0x00) // terminateur
	return out
}

// skipDNSName saute un nom (labels ou pointeur de compression) et renvoie l'offset suivant.
func skipDNSName(data []byte, off int) int {
	for off < len(data) {
		l := data[off]
		if l == 0 {
			return off + 1
		}
		if l&0xC0 == 0xC0 {
			return off + 2
		}
		off += 1 + int(l)
	}
	return off
}

// parseNBNS extrait le nom de machine et le groupe/domaine d'une réponse NBSTAT.
func parseNBNS(data []byte) (host, group string) {
	if len(data) < 12 {
		return "", ""
	}
	qd := int(binary.BigEndian.Uint16(data[4:6]))
	an := int(binary.BigEndian.Uint16(data[6:8]))
	off := 12
	for i := 0; i < qd; i++ {
		off = skipDNSName(data, off) + 4 // + type/class
	}
	for i := 0; i < an && off+10 <= len(data); i++ {
		off = skipDNSName(data, off)
		if off+10 > len(data) {
			return host, group
		}
		typ := binary.BigEndian.Uint16(data[off : off+2])
		rdlen := int(binary.BigEndian.Uint16(data[off+8 : off+10]))
		off += 10
		if off+rdlen > len(data) {
			return host, group
		}
		if typ == 0x0021 { // NBSTAT
			host, group = parseNodeNames(data[off : off+rdlen])
			return host, group
		}
		off += rdlen
	}
	return host, group
}

// parseNodeNames lit la liste de noms NetBIOS du RDATA d'une réponse NBSTAT.
func parseNodeNames(rdata []byte) (host, group string) {
	if len(rdata) < 1 {
		return "", ""
	}
	num := int(rdata[0])
	p := 1
	for i := 0; i < num && p+18 <= len(rdata); i++ {
		name := strings.TrimRight(string(rdata[p:p+15]), " \x00")
		suffix := rdata[p+15]
		flags := binary.BigEndian.Uint16(rdata[p+16 : p+18])
		p += 18
		isGroup := flags&0x8000 != 0
		if strings.HasPrefix(name, "\x01\x02") || name == "" {
			continue // __MSBROWSE__ et bruit
		}
		switch {
		case suffix == 0x00 && !isGroup && host == "":
			host = name
		case suffix == 0x00 && isGroup && group == "":
			group = name
		}
	}
	return host, group
}

// NBNS interroge chaque hôte en NetBIOS (UDP 137) et renvoie ceux qui exposent
// un nom (typiquement des machines Windows / partages SMB).
func NBNS(timeout time.Duration, srcIP string, hosts []string) []Device {
	if len(hosts) == 0 {
		return nil
	}
	conn, err := net.ListenUDP("udp4", &net.UDPAddr{IP: bindIP(srcIP), Port: 0})
	if err != nil {
		return nil
	}
	defer conn.Close()

	q := nbstatQuery()
	for _, ip := range hosts {
		if addr := net.ParseIP(ip); addr != nil {
			_, _ = conn.WriteToUDP(q, &net.UDPAddr{IP: addr, Port: 137})
		}
	}

	_ = conn.SetReadDeadline(time.Now().Add(timeout))
	byIP := map[string]Device{}
	buf := make([]byte, 2048)
	for {
		n, src, err := conn.ReadFromUDP(buf)
		if err != nil {
			break
		}
		ip := src.IP.String()
		if _, ok := byIP[ip]; ok {
			continue
		}
		host, group := parseNBNS(buf[:n])
		if host == "" {
			continue
		}
		d := Device{IP: ip, Name: host, Hostname: host, Sources: []string{"nbns"}, DeviceType: "ordinateur"}
		if group != "" {
			d.Services = []string{"workgroup:" + group}
		}
		byIP[ip] = d
	}

	out := make([]Device, 0, len(byIP))
	for _, d := range byIP {
		out = append(out, d)
	}
	return out
}

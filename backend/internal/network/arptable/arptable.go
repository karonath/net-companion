// Package arptable lit la table ARP d'un équipement réseau via SNMP
// (ipNetToMediaTable). C'est l'inventaire exhaustif côté passerelle/switch :
// tout appareil auquel l'équipement a parlé, y compris ceux que le PC local ne
// peut pas joindre directement (utile en entreprise).
package arptable

import (
	"fmt"
	"net"
	"strings"

	"netcompanion/internal/network/portfinder"
)

// oidIPNetToMediaPhys = ipNetToMediaPhysAddress : IP -> adresse physique (MAC).
// L'index de la table est ifIndex.a.b.c.d ; la valeur est la MAC (6 octets).
const oidIPNetToMediaPhys = "1.3.6.1.2.1.4.22.1.2"

// Entry est une association IP -> MAC vue par l'équipement interrogé.
type Entry struct {
	IP  string `json:"ip"`
	MAC string `json:"mac"`
}

// FromSNMP marche la table ipNetToMediaTable et renvoie les couples IP/MAC.
func FromSNMP(c portfinder.SNMPClient) []Entry {
	vals := c.WalkStrings(oidIPNetToMediaPhys)
	var out []Entry
	for oid, v := range vals {
		idx := strings.TrimPrefix(oid, oidIPNetToMediaPhys+".")
		ip := ipFromIndex(idx)
		mac := macFromBytes([]byte(v))
		if ip == "" || mac == "" {
			continue
		}
		out = append(out, Entry{IP: ip, MAC: mac})
	}
	return out
}

// ipFromIndex extrait l'IPv4 des 4 derniers composants de l'index SNMP.
func ipFromIndex(idx string) string {
	parts := strings.Split(idx, ".")
	if len(parts) < 4 {
		return ""
	}
	ip := net.ParseIP(strings.Join(parts[len(parts)-4:], "."))
	if ip == nil || ip.To4() == nil {
		return ""
	}
	return ip.String()
}

// macFromBytes formate 6 octets en MAC "aa:bb:cc:dd:ee:ff".
func macFromBytes(b []byte) string {
	if len(b) != 6 {
		return ""
	}
	return fmt.Sprintf("%02x:%02x:%02x:%02x:%02x:%02x", b[0], b[1], b[2], b[3], b[4], b[5])
}

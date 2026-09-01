// Package neighbors découvre le voisinage d'un équipement via LLDP-MIB / CDP-MIB (SNMP).
package neighbors

import (
	"sort"
	"strings"

	"netcompanion/internal/network/portfinder"
)

// OID des tables de voisinage.
const (
	oidLldpRemChassisID = "1.0.8802.1.1.2.1.4.1.1.5"
	oidLldpRemPortID    = "1.0.8802.1.1.2.1.4.1.1.7"
	oidLldpRemSysName   = "1.0.8802.1.1.2.1.4.1.1.9"

	oidCdpDeviceID   = "1.3.6.1.4.1.9.9.23.1.2.1.1.6"
	oidCdpDevicePort = "1.3.6.1.4.1.9.9.23.1.2.1.1.7"
)

// Neighbor est un voisin découvert (équipement adjacent).
type Neighbor struct {
	LocalPort       string `json:"localPort"`
	RemoteSysName   string `json:"remoteSysName"`
	RemoteChassisID string `json:"remoteChassisId"`
	RemotePortID    string `json:"remotePortId"`
	Source          string `json:"source"` // "lldp" | "cdp"
}

// FromSNMP interroge les tables LLDP puis CDP et renvoie les voisins.
func FromSNMP(c portfinder.SNMPClient) []Neighbor {
	var out []Neighbor
	out = append(out, fromLLDP(c)...)
	out = append(out, fromCDP(c)...)
	return out
}

// suffix renvoie l'index (partie après "<root>.") d'un OID complet.
func suffix(oid, root string) string {
	return strings.TrimPrefix(oid, root+".")
}

// nth renvoie le n-ième composant (0-based) d'un index pointé, ou "".
func nth(index string, n int) string {
	parts := strings.Split(index, ".")
	if n < 0 || n >= len(parts) {
		return ""
	}
	return parts[n]
}

func fromLLDP(c portfinder.SNMPClient) []Neighbor {
	sysNames := c.WalkStrings(oidLldpRemSysName)
	portIDs := c.WalkStrings(oidLldpRemPortID)
	chassis := c.WalkStrings(oidLldpRemChassisID)

	var idx []string
	for oid := range sysNames {
		idx = append(idx, suffix(oid, oidLldpRemSysName))
	}
	sort.Strings(idx)

	var out []Neighbor
	for _, i := range idx {
		out = append(out, Neighbor{
			// index LLDP = <timeMark>.<localPort>.<remIndex> → localPort = composant 1
			LocalPort:       nth(i, 1),
			RemoteSysName:   sysNames[oidLldpRemSysName+"."+i],
			RemotePortID:    portIDs[oidLldpRemPortID+"."+i],
			RemoteChassisID: chassis[oidLldpRemChassisID+"."+i],
			Source:          "lldp",
		})
	}
	return out
}

func fromCDP(c portfinder.SNMPClient) []Neighbor {
	devIDs := c.WalkStrings(oidCdpDeviceID)
	devPorts := c.WalkStrings(oidCdpDevicePort)

	var idx []string
	for oid := range devIDs {
		idx = append(idx, suffix(oid, oidCdpDeviceID))
	}
	sort.Strings(idx)

	var out []Neighbor
	for _, i := range idx {
		out = append(out, Neighbor{
			// index CDP = <ifIndex>.<devIndex> → localPort = composant 0
			LocalPort:     nth(i, 0),
			RemoteSysName: devIDs[oidCdpDeviceID+"."+i],
			RemotePortID:  devPorts[oidCdpDevicePort+"."+i],
			Source:        "cdp",
		})
	}
	return out
}

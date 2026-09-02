// Package sysinfo identifie précisément un équipement via SNMP (MIB-II system) :
// sysDescr + sysServices donnent le type réel (switch, routeur, pare-feu, point
// d'accès, imprimante, serveur…). C'est la source la plus fiable en entreprise.
package sysinfo

import (
	"strconv"
	"strings"

	"netcompanion/internal/network/portfinder"
)

// OID MIB-II (RFC 1213).
const (
	oidSysDescr    = "1.3.6.1.2.1.1.1.0"
	oidSysName     = "1.3.6.1.2.1.1.5.0"
	oidSysServices = "1.3.6.1.2.1.1.7.0"
)

// Info regroupe l'identité SNMP d'un équipement.
type Info struct {
	SysName    string
	SysDescr   string
	DeviceType string
}

// FromSNMP lit sysDescr/sysName/sysServices et en déduit le type. Le second
// retour est false si l'équipement ne répond pas au SNMP (pas de sysDescr).
func FromSNMP(c portfinder.SNMPClient) (Info, bool) {
	descr, ok := c.Get(oidSysDescr)
	if !ok || strings.TrimSpace(descr) == "" {
		return Info{}, false
	}
	name, _ := c.Get(oidSysName)
	services, _ := c.Get(oidSysServices)
	return Info{
		SysName:    strings.TrimSpace(name),
		SysDescr:   strings.TrimSpace(descr),
		DeviceType: Classify(descr, services),
	}, true
}

// Classify devine le type d'équipement depuis sysDescr (mots-clés constructeurs)
// puis affine avec sysServices (couches OSI actives). Renvoie "" si indéterminé.
func Classify(sysDescr, sysServices string) string {
	d := strings.ToLower(sysDescr)
	switch {
	case has(d, "firewall", "fortigate", "fortios", "palo alto", "pan-os", "sonicwall",
		"checkpoint", "check point", "pfsense", "opnsense", "adaptive security", "cisco asa"):
		return "pare-feu"
	case has(d, "access point", "wireless lan", "wlan controller", "aironet", "air-cap",
		"unifi ap", "instant ap", "aruba ap", "meraki mr", "lightweight ap"):
		return "point d'accès"
	case has(d, "voip", "ip phone", "polycom", "yealink", "grandstream", "snom",
		"cisco unified", "sip phone"):
		return "téléphone VoIP"
	case has(d, "jetdirect", "laserjet", "officejet", "deskjet", "printer", "kyocera",
		"lexmark", "xerox", "brother", "zebra", "epson stylus", "imagerunner"):
		return "imprimante"
	case has(d, "esxi", "vmware esx", "proxmox", "hyper-v", "xenserver", "vsphere"):
		return "hyperviseur"
	case has(d, "synology", "qnap", "truenas", "freenas", "netapp", "diskstation",
		"unraid", "storage array", "san "):
		return "NAS / stockage"
	case has(d, "ups", "smart-ups", "apc web", "eaton", "riello", "powerware", "galaxy vs"):
		return "onduleur"
	case has(d, "s7-", "simatic", "siemens s7", "modbus", "scada", "rockwell",
		"allen-bradley", "schneider electric", " plc"):
		return "automate / OT"
	case has(d, "camera", "ipcam", "hikvision", "dahua", "axis ", "reolink", "network camera"):
		return "caméra"
	case has(d, "catalyst", "nexus", "procurve", "provision", "powerconnect", "qfx",
		"ex2200", "ex3300", "ex4300", "switch", "dgs-", "dell emc networking"):
		return refineL2L3(sysServices, "switch")
	case has(d, "ios-xe", "ios ", "cisco ios", "routeros", "mikrotik", "edgerouter",
		"vyos", "router", "versatile routing", "junos"):
		return refineL2L3(sysServices, "routeur / box")
	case has(d, "windows server", "server 20", "ubuntu", "debian", "centos", "red hat",
		"rhel", "freebsd", "linux", "windows"):
		return "serveur"
	}
	// Aucun mot-clé : on se rabat sur les couches OSI actives.
	return refineL2L3(sysServices, "")
}

// refineL2L3 utilise sysServices (somme de bits par couche OSI) pour distinguer
// un switch (L2, bit 2) d'un routeur (L3, bit 4). fallback si indéterminable.
func refineL2L3(sysServices, fallback string) string {
	n, err := strconv.Atoi(strings.TrimSpace(sysServices))
	if err != nil {
		return fallback
	}
	l2 := n&0x02 != 0
	l3 := n&0x04 != 0
	switch {
	case l3:
		if fallback == "switch" {
			return "switch" // switch L3 : on garde switch
		}
		return "routeur / box"
	case l2:
		return "switch"
	default:
		return fallback
	}
}

func has(s string, subs ...string) bool {
	for _, sub := range subs {
		if strings.Contains(s, sub) {
			return true
		}
	}
	return false
}

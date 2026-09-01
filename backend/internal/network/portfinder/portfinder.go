// Package portfinder localise physiquement un hôte via SNMP (BRIDGE-MIB).
package portfinder

import (
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"

	"netcompanion/internal/models"
)

// OID de base (BRIDGE-MIB / Q-BRIDGE-MIB / IF-MIB).
const (
	oidFdbPort    = "1.3.6.1.2.1.17.4.3.1.2"     // dot1dTpFdbPort.<mac>
	oidBasePortIf = "1.3.6.1.2.1.17.1.4.1.2"     // dot1dBasePortIfIndex.<port>
	oidIfName     = "1.3.6.1.2.1.31.1.1.1.1"     // ifName.<ifIndex>
	oidSysName    = "1.3.6.1.2.1.1.5.0"          // sysName.0
	oidQBridgeFdb = "1.3.6.1.2.1.17.7.1.2.2.1.2" // dot1qTpFdbPort walk (index: vlan.mac)
)

// SNMPClient est l'accès SNMP minimal nécessaire (mockable en test).
type SNMPClient interface {
	Get(oid string) (string, bool)
	WalkStrings(root string) map[string]string
}

// macToOIDSuffix convertit une MAC en suffixe OID décimal pointé.
func macToOIDSuffix(mac string) (string, error) {
	hw, err := net.ParseMAC(mac)
	if err != nil {
		return "", err
	}
	parts := make([]string, len(hw))
	for i, b := range hw {
		parts[i] = strconv.Itoa(int(b))
	}
	return strings.Join(parts, "."), nil
}

// Locate résout le port physique et le VLAN de targetMAC via le client SNMP.
func Locate(c SNMPClient, targetMAC string) (models.PortLocation, error) {
	suffix, err := macToOIDSuffix(targetMAC)
	if err != nil {
		return models.PortLocation{}, err
	}

	portStr, ok := c.Get(oidFdbPort + "." + suffix)
	if !ok || portStr == "" {
		return models.PortLocation{}, errors.New("MAC absente de la table de commutation (dot1dTpFdbPort)")
	}
	bridgePort, err := strconv.Atoi(strings.TrimSpace(portStr))
	if err != nil {
		return models.PortLocation{}, fmt.Errorf("bridge port illisible: %v", err)
	}

	loc := models.PortLocation{BridgePort: bridgePort}

	if ifIdxStr, ok := c.Get(oidBasePortIf + "." + strconv.Itoa(bridgePort)); ok {
		if ifIdx, err := strconv.Atoi(strings.TrimSpace(ifIdxStr)); err == nil {
			loc.IfIndex = ifIdx
			if name, ok := c.Get(oidIfName + "." + strconv.Itoa(ifIdx)); ok {
				loc.PortIfName = name
			}
		}
	}

	loc.VLAN = findVLAN(c, suffix)

	if dev, ok := c.Get(oidSysName); ok {
		loc.Device = dev
	}

	loc.Sentence = sentence(loc)
	return loc, nil
}

// findVLAN parcourt la Q-BRIDGE FDB et renvoie le VLAN indexant targetMAC (0 si inconnu).
func findVLAN(c SNMPClient, macSuffix string) int {
	walk := c.WalkStrings(oidQBridgeFdb)
	for oid := range walk {
		rest := strings.TrimPrefix(oid, oidQBridgeFdb+".")
		if rest == oid {
			continue
		}
		// rest = <vlan>.<mac 6 octets décimaux>
		if !strings.HasSuffix(rest, macSuffix) {
			continue
		}
		vlanPart := strings.TrimSuffix(rest, "."+macSuffix)
		if v, err := strconv.Atoi(vlanPart); err == nil {
			return v
		}
	}
	return 0
}

func sentence(loc models.PortLocation) string {
	dev := loc.Device
	if dev == "" {
		dev = "l'équipement"
	}
	port := loc.PortIfName
	if port == "" {
		port = "port bridge " + strconv.Itoa(loc.BridgePort)
	}
	if loc.VLAN > 0 {
		return fmt.Sprintf("Vous êtes branché sur %s, port %s, VLAN %d.", dev, port, loc.VLAN)
	}
	return fmt.Sprintf("Vous êtes branché sur %s, port %s.", dev, port)
}

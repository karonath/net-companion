// Package netinfo découvre l'interface réseau active et la passerelle.
package netinfo

import (
	"errors"
	"net"
	"strconv"
	"strings"

	"github.com/jackpal/gateway"

	"netcompanion/internal/models"
)

// selectInterface choisit l'interface à utiliser :
//  1. celle dont le sous-réseau contient la passerelle par défaut (routable) ;
//  2. sinon la première interface non link-local (169.254.x écartée) ;
//  3. en dernier recours, la première interface disponible.
func selectInterface(cands []models.InterfaceInfo, gatewayIP string) (models.InterfaceInfo, error) {
	if len(cands) == 0 {
		return models.InterfaceInfo{}, errors.New("aucune interface réseau active avec IPv4 trouvée")
	}

	// 1) l'interface qui possède la passerelle par défaut
	if gw := net.ParseIP(gatewayIP); gw != nil {
		for _, c := range cands {
			if _, ipnet, err := net.ParseCIDR(c.CIDR); err == nil && ipnet.Contains(gw) {
				return c, nil
			}
		}
	}

	// 2) première interface routable (on écarte le link-local APIPA 169.254.x)
	for _, c := range cands {
		if ip := net.ParseIP(c.IPv4); ip != nil && !ip.IsLinkLocalUnicast() {
			return c, nil
		}
	}

	// 3) dernier recours
	return cands[0], nil
}

// enumerate liste les interfaces up, non-loopback, avec MAC et IPv4.
func enumerate() []models.InterfaceInfo {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil
	}
	var out []models.InterfaceInfo
	for _, ifi := range ifaces {
		if ifi.Flags&net.FlagUp == 0 || ifi.Flags&net.FlagLoopback != 0 {
			continue
		}
		if ifi.HardwareAddr.String() == "" {
			continue
		}
		addrs, err := ifi.Addrs()
		if err != nil {
			continue
		}
		for _, a := range addrs {
			ipnet, ok := a.(*net.IPNet)
			if !ok || ipnet.IP.To4() == nil || ipnet.IP.IsLoopback() {
				continue
			}
			ones, _ := ipnet.Mask.Size()
			out = append(out, models.InterfaceInfo{
				Name: ifi.Name,
				MAC:  ifi.HardwareAddr.String(),
				IPv4: ipnet.IP.String(),
				CIDR: ipnet.IP.String() + "/" + strconv.Itoa(ones),
			})
		}
	}
	return out
}

// LocalInterface renvoie l'interface active (auto-détection ancrée sur la passerelle).
func LocalInterface() (models.InterfaceInfo, error) {
	gw, _ := DefaultGateway()
	return selectInterface(enumerate(), gw)
}

// ListInterfaces renvoie toutes les interfaces exploitables (pour le choix manuel).
func ListInterfaces() []models.InterfaceInfo {
	cands := enumerate()
	if cands == nil {
		return []models.InterfaceInfo{}
	}
	return cands
}

// InterfaceByName renvoie l'interface portant ce nom, si elle existe.
func InterfaceByName(name string) (models.InterfaceInfo, bool) {
	for _, c := range enumerate() {
		if c.Name == name {
			return c, true
		}
	}
	return models.InterfaceInfo{}, false
}

// DefaultGateway renvoie l'IP de la passerelle par défaut.
func DefaultGateway() (string, error) {
	ip, err := gateway.DiscoverGateway()
	if err != nil {
		return "", err
	}
	return ip.String(), nil
}

// ReverseDNS renvoie le nom d'hôte (PTR) associé à ip, ou "" si aucun.
func ReverseDNS(ip string) string {
	names, err := net.LookupAddr(ip)
	if err != nil || len(names) == 0 {
		return ""
	}
	return strings.TrimSuffix(names[0], ".")
}

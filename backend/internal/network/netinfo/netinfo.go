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

// selectInterface choisit la première interface exploitable (IPv4, non loopback).
func selectInterface(cands []models.InterfaceInfo) (models.InterfaceInfo, error) {
	for _, c := range cands {
		if c.MAC == "" {
			continue
		}
		if c.IPv4 == "" || strings.HasPrefix(c.IPv4, "127.") {
			continue
		}
		return c, nil
	}
	return models.InterfaceInfo{}, errors.New("aucune interface réseau active avec IPv4 trouvée")
}

// LocalInterface énumère les interfaces système et renvoie l'active.
func LocalInterface() (models.InterfaceInfo, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return models.InterfaceInfo{}, err
	}
	var cands []models.InterfaceInfo
	for _, ifi := range ifaces {
		if ifi.Flags&net.FlagUp == 0 || ifi.Flags&net.FlagLoopback != 0 {
			continue
		}
		addrs, err := ifi.Addrs()
		if err != nil {
			continue
		}
		for _, a := range addrs {
			ipnet, ok := a.(*net.IPNet)
			if !ok || ipnet.IP.To4() == nil {
				continue
			}
			ones, _ := ipnet.Mask.Size()
			cands = append(cands, models.InterfaceInfo{
				Name: ifi.Name,
				MAC:  ifi.HardwareAddr.String(),
				IPv4: ipnet.IP.String(),
				CIDR: ipnet.IP.String() + "/" + strconv.Itoa(ones),
			})
		}
	}
	return selectInterface(cands)
}

// ReverseDNS renvoie le nom d'hôte (PTR) associé à ip, ou "" si aucun.
func ReverseDNS(ip string) string {
	names, err := net.LookupAddr(ip)
	if err != nil || len(names) == 0 {
		return ""
	}
	return strings.TrimSuffix(names[0], ".")
}

// DefaultGateway renvoie l'IP de la passerelle par défaut.
func DefaultGateway() (string, error) {
	ip, err := gateway.DiscoverGateway()
	if err != nil {
		return "", err
	}
	return ip.String(), nil
}

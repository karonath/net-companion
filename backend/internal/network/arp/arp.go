// Package arp lit et parse la table ARP locale (voisins niveau 2).
package arp

import (
	"net"
	"os/exec"
	"regexp"
	"runtime"
	"strings"
)

// Neighbor est une entrée (IP, MAC) de la table ARP.
type Neighbor struct {
	IP  string `json:"ip"`
	MAC string `json:"mac"`
}

var (
	reIPv4    = regexp.MustCompile(`\b(\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3})\b`)
	reMACWin  = regexp.MustCompile(`\b([0-9a-fA-F]{2}(?:-[0-9a-fA-F]{2}){5})\b`)
	reMACUnix = regexp.MustCompile(`\b([0-9a-fA-F]{1,2}(?::[0-9a-fA-F]{1,2}){5})\b`)
)

// parseWindows parse la sortie de `arp -a` (Windows).
func parseWindows(raw string) []Neighbor {
	var out []Neighbor
	for _, line := range strings.Split(raw, "\n") {
		ip := reIPv4.FindString(line)
		mac := reMACWin.FindString(line)
		if ip == "" || mac == "" {
			continue
		}
		out = append(out, Neighbor{IP: ip, MAC: normalizeMAC(mac)})
	}
	return out
}

// parseLinux parse la sortie de `ip neigh` (Linux) ; ignore les entrées sans MAC.
func parseLinux(raw string) []Neighbor {
	var out []Neighbor
	for _, line := range strings.Split(raw, "\n") {
		if !strings.Contains(line, "lladdr") {
			continue
		}
		ip := reIPv4.FindString(line)
		mac := reMACUnix.FindString(line)
		if ip == "" || mac == "" {
			continue
		}
		out = append(out, Neighbor{IP: ip, MAC: normalizeMAC(mac)})
	}
	return out
}

// Read lit la table ARP du système courant.
func Read() ([]Neighbor, error) {
	if runtime.GOOS == "windows" {
		raw, err := exec.Command("arp", "-a").Output()
		if err != nil {
			return nil, err
		}
		return parseWindows(string(raw)), nil
	}
	raw, err := exec.Command("ip", "neigh").Output()
	if err != nil {
		return nil, err
	}
	return parseLinux(string(raw)), nil
}

// normalizeMAC renvoie une MAC en minuscules, séparée par ':' et zéro-paddée.
func normalizeMAC(s string) string {
	s = strings.ReplaceAll(s, "-", ":")
	hw, err := net.ParseMAC(s)
	if err != nil {
		return strings.ToLower(s)
	}
	return hw.String()
}

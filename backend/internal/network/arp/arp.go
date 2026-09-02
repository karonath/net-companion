// Package arp lit et parse la table ARP locale (voisins niveau 2).
package arp

import (
	"net"
	"os"
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

// parseProcArp parse /etc/… /proc/net/arp (Linux) ; ignore les entrées incomplètes.
func parseProcArp(raw string) []Neighbor {
	var out []Neighbor
	for i, line := range strings.Split(raw, "\n") {
		if i == 0 {
			continue // en-tête
		}
		f := strings.Fields(line)
		if len(f) < 4 {
			continue
		}
		ip, flags, mac := f[0], f[2], f[3]
		if flags == "0x0" || mac == "00:00:00:00:00:00" {
			continue // résolution incomplète
		}
		if reIPv4.FindString(ip) == "" {
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
	// Linux : /proc/net/arp est toujours disponible (pas de sous-processus ni root).
	if data, err := os.ReadFile("/proc/net/arp"); err == nil {
		if ns := parseProcArp(string(data)); len(ns) > 0 {
			return ns, nil
		}
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

//go:build !windows

package arp

// ScanIPs : le balayage ARP actif natif n'est disponible que sous Windows
// (API SendARP sans privilège). Ailleurs, on s'appuie sur le balayage TCP qui
// réchauffe le cache ARP du système, puis sur la lecture de Read().
func ScanIPs(ips []string) []Neighbor { return nil }

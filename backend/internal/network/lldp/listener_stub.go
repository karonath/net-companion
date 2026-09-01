//go:build !npcap

package lldp

import "time"

// Capture (build par défaut) : capture indisponible sans Npcap.
func Capture(iface string, timeout time.Duration) Result {
	return Result{
		Available: false,
		Reason:    "capture LLDP indisponible : nécessite Npcap et une compilation avec -tags npcap",
		Neighbors: []Neighbor{},
	}
}

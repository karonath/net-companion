// Package history persiste les snapshots d'intervention et détecte les changements.
package history

import (
	"time"

	"netcompanion/internal/models"
	"netcompanion/internal/network/diag"
	"netcompanion/internal/network/health"
)

// Snapshot est un cliché d'intervention (inventaire + diagnostics horodatés).
type Snapshot struct {
	ID        string               `json:"id"`
	Timestamp time.Time            `json:"timestamp"`
	Label     string               `json:"label,omitempty"`
	Notes     string               `json:"notes,omitempty"`
	Interface models.InterfaceInfo `json:"interface"`
	Gateway   string               `json:"gateway"`
	Hosts     []models.Host        `json:"hosts"`
	Diag      []diag.Check         `json:"diag"`
	Health    *health.Report       `json:"health,omitempty"`
}

// Meta est le résumé d'un snapshot pour la liste d'historique.
type Meta struct {
	ID        string    `json:"id"`
	Timestamp time.Time `json:"timestamp"`
	HostCount int       `json:"hostCount"`
}

// CheckChange décrit un check dont le statut a changé entre deux passages.
type CheckChange struct {
	Name string `json:"name"`
	From string `json:"from"`
	To   string `json:"to"`
}

// Changes agrège les différences entre deux snapshots.
type Changes struct {
	HostsAdded    []models.Host `json:"hostsAdded"`
	HostsRemoved  []models.Host `json:"hostsRemoved"`
	GatewayFrom   string        `json:"gatewayFrom,omitempty"`
	GatewayTo     string        `json:"gatewayTo,omitempty"`
	ChecksChanged []CheckChange `json:"checksChanged"`
}

// NewID génère un identifiant horodaté triable pour un snapshot.
func NewID(t time.Time) string { return t.Format("20060102-150405") }

// Diff compare prev et cur et renvoie les changements (hôtes par IP, checks par nom).
func Diff(prev, cur Snapshot) Changes {
	var ch Changes

	prevIPs := map[string]bool{}
	for _, h := range prev.Hosts {
		prevIPs[h.IP] = true
	}
	curIPs := map[string]bool{}
	for _, h := range cur.Hosts {
		curIPs[h.IP] = true
	}
	for _, h := range cur.Hosts {
		if !prevIPs[h.IP] {
			ch.HostsAdded = append(ch.HostsAdded, h)
		}
	}
	for _, h := range prev.Hosts {
		if !curIPs[h.IP] {
			ch.HostsRemoved = append(ch.HostsRemoved, h)
		}
	}

	if prev.Gateway != cur.Gateway {
		ch.GatewayFrom = prev.Gateway
		ch.GatewayTo = cur.Gateway
	}

	prevChecks := map[string]string{}
	for _, c := range prev.Diag {
		prevChecks[c.Name] = c.Status
	}
	for _, c := range cur.Diag {
		if from, ok := prevChecks[c.Name]; ok && from != c.Status {
			ch.ChecksChanged = append(ch.ChecksChanged, CheckChange{Name: c.Name, From: from, To: c.Status})
		}
	}
	return ch
}

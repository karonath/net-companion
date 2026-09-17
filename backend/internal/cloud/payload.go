package cloud

import (
	"time"

	"netcompanion/internal/history"
	"netcompanion/internal/network/health"
)

// Summary reprend les seuls champs indexés côté cloud (DynamoDB).
type Summary struct {
	HealthScore    int    `json:"healthScore"`
	HealthGrade    string `json:"healthGrade"`
	HostCount      int    `json:"hostCount"`
	CriticalIssues int    `json:"criticalIssues"`
}

// Payload est le corps POST envoyé à l'ingestion (contrat partagé avec le Lambda).
type Payload struct {
	ClientID     string           `json:"clientId"`
	SiteID       string           `json:"siteId"`
	TechnicianID string           `json:"technicianId"`
	SnapshotID   string           `json:"snapshotId"`
	Timestamp    time.Time        `json:"timestamp"`
	Summary      Summary          `json:"summary"`
	Snapshot     history.Snapshot `json:"snapshot"`
}

// BuildPayload construit le payload d'ingestion à partir de la config et du snapshot.
func BuildPayload(c Config, snap history.Snapshot) Payload {
	var s Summary
	s.HostCount = len(snap.Hosts)
	if snap.Health != nil {
		s.HealthScore = snap.Health.Score
		s.HealthGrade = snap.Health.Grade
		for _, iss := range snap.Health.Issues {
			if iss.Severity == health.SevCritical {
				s.CriticalIssues++
			}
		}
	}
	return Payload{
		ClientID:     c.ClientID,
		SiteID:       c.SiteID,
		TechnicianID: c.TechID,
		SnapshotID:   snap.ID,
		Timestamp:    snap.Timestamp,
		Summary:      s,
		Snapshot:     snap,
	}
}

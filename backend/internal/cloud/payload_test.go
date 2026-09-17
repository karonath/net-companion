package cloud

import (
	"testing"
	"time"

	"netcompanion/internal/history"
	"netcompanion/internal/network/health"
)

func TestBuildPayloadMapsSummary(t *testing.T) {
	snap := history.Snapshot{
		ID:        "20260917-140300",
		Timestamp: time.Date(2026, 9, 17, 14, 3, 0, 0, time.UTC),
		Health: &health.Report{
			Score: 92, Grade: "A",
			Issues: []health.Issue{
				{Severity: health.SevCritical, Title: "x"},
				{Severity: health.SevWarning, Title: "y"},
			},
		},
	}
	cfg := Config{URL: "https://api", Key: "k", ClientID: "acme", SiteID: "paris-hq", TechID: "tech-charlys"}

	p := BuildPayload(cfg, snap)

	if p.ClientID != "acme" || p.SiteID != "paris-hq" || p.TechnicianID != "tech-charlys" {
		t.Fatalf("identité mal mappée: %+v", p)
	}
	if p.SnapshotID != "20260917-140300" {
		t.Fatalf("snapshotId=%q", p.SnapshotID)
	}
	if p.Summary.HealthScore != 92 || p.Summary.HealthGrade != "A" {
		t.Fatalf("score/grade: %+v", p.Summary)
	}
	if p.Summary.CriticalIssues != 1 {
		t.Fatalf("criticalIssues attendu 1, obtenu %d", p.Summary.CriticalIssues)
	}
}

func TestConfigEnabledAndComplete(t *testing.T) {
	if (Config{}).Enabled() {
		t.Fatal("config vide ne doit pas être Enabled")
	}
	c := Config{URL: "https://api", Key: "k"}
	if !c.Enabled() {
		t.Fatal("URL+Key doit être Enabled")
	}
	if c.Complete() {
		t.Fatal("sans ClientID/SiteID ne doit pas être Complete")
	}
	c.ClientID, c.SiteID = "acme", "hq"
	if !c.Complete() {
		t.Fatal("doit être Complete")
	}
}

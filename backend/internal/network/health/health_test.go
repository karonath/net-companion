package health

import (
	"testing"

	"netcompanion/internal/models"
	"netcompanion/internal/network/diag"
)

func TestAnalyzeHealthy(t *testing.T) {
	hosts := []models.Host{
		{IP: "192.168.1.1", MAC: "aa:bb:cc:00:00:01", DeviceType: "routeur / box"},
		{IP: "192.168.1.2", MAC: "aa:bb:cc:00:00:02", DeviceType: "ordinateur"},
	}
	checks := []diag.Check{{Name: "Accès Internet", Status: diag.StatusOK}}
	r := Analyze(hosts, checks, "192.168.1.1")
	if r.Score != 100 || r.Grade != "A" || len(r.Issues) != 0 {
		t.Fatalf("réseau sain attendu, obtenu %+v", r)
	}
}

func TestAnalyzeAnomalies(t *testing.T) {
	hosts := []models.Host{
		{IP: "192.168.1.10", MAC: "aa:bb:cc:00:00:10", DeviceType: "ordinateur"},
		{IP: "192.168.1.11", MAC: "aa:bb:cc:00:00:10"}, // MAC dupliquée + non identifié
		{IP: "169.254.5.5", MAC: "aa:bb:cc:00:00:12"},  // APIPA + non identifié
	}
	checks := []diag.Check{
		{Name: "Accès Internet", Status: diag.StatusFail, Detail: "hors ligne"},
		{Name: "Résolution DNS", Status: diag.StatusWarn, Detail: "DNS lent"},
	}
	r := Analyze(hosts, checks, "192.168.1.1")

	if r.Score >= 100 {
		t.Errorf("le score doit être dégradé, obtenu %d", r.Score)
	}
	// critical (internet) doit passer en premier
	if len(r.Issues) == 0 || r.Issues[0].Severity != SevCritical {
		t.Fatalf("anomalie critique en tête attendue, obtenu %+v", r.Issues)
	}
	// doit contenir MAC dupliquée + APIPA
	var hasDup, hasApipa bool
	for _, is := range r.Issues {
		if is.Title == "MAC dupliquée" {
			hasDup = true
		}
		if is.Title == "Adresse APIPA" {
			hasApipa = true
		}
	}
	if !hasDup || !hasApipa {
		t.Errorf("MAC dupliquée et APIPA attendues : %+v", r.Issues)
	}
}

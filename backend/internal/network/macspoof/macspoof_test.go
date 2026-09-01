package macspoof

import (
	"strings"
	"testing"
)

func TestNormalizeMAC(t *testing.T) {
	colon, bare, err := NormalizeMAC("00-1a-2b-3C-4d-5e")
	if err != nil {
		t.Fatalf("NormalizeMAC: %v", err)
	}
	if colon != "00:1a:2b:3c:4d:5e" {
		t.Fatalf("colon = %q", colon)
	}
	if bare != "001A2B3C4D5E" {
		t.Fatalf("bare = %q", bare)
	}
	if _, _, err := NormalizeMAC("pas-mac"); err == nil {
		t.Fatal("attendu une erreur sur MAC invalide")
	}
}

func TestBuildPlanWindows(t *testing.T) {
	p, err := BuildPlan("windows", "Wi-Fi", "00:1a:2b:3c:4d:5e")
	if err != nil {
		t.Fatalf("BuildPlan: %v", err)
	}
	if p.OS != "windows" || len(p.Steps) == 0 {
		t.Fatalf("plan = %+v", p)
	}
	joined := ""
	for _, s := range p.Steps {
		joined += s.Command + " " + strings.Join(s.Args, " ") + "\n"
	}
	if !strings.Contains(joined, "001A2B3C4D5E") {
		t.Fatalf("le plan Windows doit contenir la MAC nue :\n%s", joined)
	}
	if !strings.Contains(joined, "Wi-Fi") {
		t.Fatalf("le plan doit référencer l'interface :\n%s", joined)
	}
}

func TestBuildPlanLinux(t *testing.T) {
	p, err := BuildPlan("linux", "eth0", "00:1a:2b:3c:4d:5e")
	if err != nil {
		t.Fatalf("BuildPlan: %v", err)
	}
	joined := ""
	for _, s := range p.Steps {
		joined += s.Command + " " + strings.Join(s.Args, " ") + "\n"
	}
	if !strings.Contains(joined, "00:1a:2b:3c:4d:5e") || !strings.Contains(joined, "eth0") {
		t.Fatalf("plan linux incomplet :\n%s", joined)
	}
}

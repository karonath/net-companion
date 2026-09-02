// Package health calcule un score de santé réseau et détecte des anomalies à
// partir de l'inventaire (hôtes) et des diagnostics de connectivité.
package health

import (
	"fmt"
	"sort"
	"strings"

	"netcompanion/internal/models"
	"netcompanion/internal/network/diag"
)

// Sévérités d'anomalie.
const (
	SevCritical = "critical"
	SevWarning  = "warning"
	SevInfo     = "info"
)

// Issue est une anomalie détectée.
type Issue struct {
	Severity string `json:"severity"`
	Title    string `json:"title"`
	Detail   string `json:"detail"`
}

// Report agrège le score, la note et les anomalies.
type Report struct {
	Score   int     `json:"score"`
	Grade   string  `json:"grade"`
	Summary string  `json:"summary"`
	Issues  []Issue `json:"issues"`
}

// Analyze produit un bilan de santé à partir des hôtes et des contrôles.
func Analyze(hosts []models.Host, checks []diag.Check, gateway string) Report {
	issues := []Issue{} // jamais nil (JSON = [] même sans anomalie)

	// 1) Contrôles de connectivité en échec/alerte.
	for _, c := range checks {
		switch c.Status {
		case diag.StatusFail:
			issues = append(issues, Issue{SevCritical, c.Name, c.Detail})
		case diag.StatusWarn:
			issues = append(issues, Issue{SevWarning, c.Name, c.Detail})
		}
	}

	// 2) MAC dupliquée = conflit d'IP ou usurpation possible.
	macIPs := map[string][]string{}
	for _, h := range hosts {
		if h.MAC == "" {
			continue
		}
		m := strings.ToLower(h.MAC)
		macIPs[m] = append(macIPs[m], h.IP)
	}
	var dupMacs []string
	for m := range macIPs {
		if len(macIPs[m]) > 1 {
			dupMacs = append(dupMacs, m)
		}
	}
	sort.Strings(dupMacs)
	for _, m := range dupMacs {
		ips := macIPs[m]
		sort.Strings(ips)
		issues = append(issues, Issue{SevWarning, "MAC dupliquée",
			fmt.Sprintf("%s partagée par %s (conflit d'IP ou usurpation ?)", m, strings.Join(ips, ", "))})
	}

	// 3) Adresses APIPA (169.254.x) = pas de bail DHCP.
	for _, h := range hosts {
		if strings.HasPrefix(h.IP, "169.254.") {
			issues = append(issues, Issue{SevWarning, "Adresse APIPA",
				fmt.Sprintf("%s sans bail DHCP (169.254.x)", h.IP)})
		}
	}

	// 4) Appareils non identifiés (information).
	unknown := 0
	for _, h := range hosts {
		if h.DeviceType == "" {
			unknown++
		}
	}
	if unknown > 0 {
		issues = append(issues, Issue{SevInfo, "Appareils non identifiés",
			fmt.Sprintf("%d appareil(s) sans type déterminé", unknown)})
	}

	// Score : 100 moins un poids par anomalie selon la gravité.
	score := 100
	for _, is := range issues {
		switch is.Severity {
		case SevCritical:
			score -= 20
		case SevWarning:
			score -= 8
		case SevInfo:
			score -= 2
		}
	}
	if score < 0 {
		score = 0
	}

	sort.SliceStable(issues, func(i, j int) bool { return sevRank(issues[i].Severity) < sevRank(issues[j].Severity) })
	return Report{Score: score, Grade: gradeFor(score), Summary: summaryFor(score, issues), Issues: issues}
}

func sevRank(s string) int {
	switch s {
	case SevCritical:
		return 0
	case SevWarning:
		return 1
	default:
		return 2
	}
}

func gradeFor(score int) string {
	switch {
	case score >= 90:
		return "A"
	case score >= 75:
		return "B"
	case score >= 50:
		return "C"
	case score >= 25:
		return "D"
	default:
		return "E"
	}
}

func summaryFor(score int, issues []Issue) string {
	if len(issues) == 0 {
		return "Réseau sain — aucun problème détecté."
	}
	for _, is := range issues {
		if is.Severity == SevCritical {
			return "Problèmes critiques détectés — action recommandée."
		}
	}
	for _, is := range issues {
		if is.Severity == SevWarning {
			return "Quelques points d'attention."
		}
	}
	return "Réseau sain (remarques mineures)."
}

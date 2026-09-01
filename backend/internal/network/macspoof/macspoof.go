// Package macspoof usurpe l'adresse MAC de l'interface (parade NAC 802.1x).
package macspoof

import (
	"errors"
	"fmt"
	"net"
	"strings"
)

// ErrElevationRequired est renvoyée si l'application requiert des droits admin/root absents.
var ErrElevationRequired = errors.New("élévation de privilèges requise (admin/root)")

// Step est une commande système unitaire du plan de spoofing.
type Step struct {
	Description string   `json:"description"`
	Command     string   `json:"command"`
	Args        []string `json:"args"`
}

// Plan est la séquence de commandes qui usurperait la MAC (affichable en dry-run).
type Plan struct {
	OS    string `json:"os"`
	MAC   string `json:"mac"`
	Steps []Step `json:"steps"`
}

// NormalizeMAC renvoie la MAC en forme ':' minuscule et en forme nue MAJUSCULE.
func NormalizeMAC(mac string) (colon string, bare string, err error) {
	hw, err := net.ParseMAC(strings.ReplaceAll(mac, "-", ":"))
	if err != nil {
		return "", "", err
	}
	if len(hw) != 6 {
		return "", "", fmt.Errorf("adresse MAC EUI-48 attendue, obtenu %d octets", len(hw))
	}
	colon = hw.String()
	bare = strings.ToUpper(strings.ReplaceAll(colon, ":", ""))
	return colon, bare, nil
}

// BuildPlan construit le plan de commandes pour l'OS cible.
func BuildPlan(goos, iface, mac string) (Plan, error) {
	colon, bare, err := NormalizeMAC(mac)
	if err != nil {
		return Plan{}, err
	}
	switch goos {
	case "windows":
		return Plan{OS: goos, MAC: colon, Steps: []Step{
			{
				Description: "Écrit la MAC dans la propriété avancée NetworkAddress de l'adaptateur",
				Command:     "powershell",
				Args:        []string{"-NoProfile", "-Command", fmt.Sprintf("Set-NetAdapterAdvancedProperty -Name '%s' -RegistryKeyword 'NetworkAddress' -RegistryValue '%s' -NoRestart", iface, bare)},
			},
			{
				Description: "Redémarre l'adaptateur pour appliquer la nouvelle MAC",
				Command:     "powershell",
				Args:        []string{"-NoProfile", "-Command", fmt.Sprintf("Restart-NetAdapter -Name '%s'", iface)},
			},
			{
				Description: "Renouvelle le bail DHCP",
				Command:     "ipconfig",
				Args:        []string{"/renew"},
			},
		}}, nil
	case "linux":
		return Plan{OS: goos, MAC: colon, Steps: []Step{
			{Description: "Descend l'interface", Command: "ip", Args: []string{"link", "set", "dev", iface, "down"}},
			{Description: "Change la MAC", Command: "ip", Args: []string{"link", "set", "dev", iface, "address", colon}},
			{Description: "Remonte l'interface", Command: "ip", Args: []string{"link", "set", "dev", iface, "up"}},
			{Description: "Renouvelle le bail DHCP", Command: "dhclient", Args: []string{iface}},
		}}, nil
	case "darwin":
		return Plan{OS: goos, MAC: colon, Steps: []Step{
			{Description: "Change la MAC", Command: "ifconfig", Args: []string{iface, "ether", colon}},
		}}, nil
	default:
		return Plan{}, fmt.Errorf("OS non supporté pour le MAC spoofing : %s", goos)
	}
}

// run exécute les étapes du plan (utilisé par Apply après vérification admin).
func run(p Plan) error {
	for _, s := range p.Steps {
		if err := execStep(s); err != nil {
			return fmt.Errorf("étape %q: %w", s.Description, err)
		}
	}
	return nil
}

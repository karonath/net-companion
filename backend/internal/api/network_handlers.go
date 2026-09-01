package api

import (
	"errors"
	"net/http"
	"strings"

	"netcompanion/internal/models"
	"netcompanion/internal/network/arp"
	"netcompanion/internal/network/netinfo"
	"netcompanion/internal/network/oui"
	"netcompanion/internal/network/portfinder"
	"netcompanion/internal/network/radar"
	"netcompanion/internal/vault"
)

var errNoCommunity = errors.New("aucune community SNMP dans le coffre (ajoutez-en une)")

// registerNetwork ajoute les routes réseau sur mux.
func registerNetwork(mux *http.ServeMux, v *vault.Vault) {
	mux.HandleFunc("GET /api/network/info", func(w http.ResponseWriter, r *http.Request) {
		ifi, err := netinfo.LocalInterface()
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		gw, _ := netinfo.DefaultGateway()
		writeJSON(w, http.StatusOK, map[string]any{"interface": ifi, "gateway": gw})
	})

	mux.HandleFunc("GET /api/network/radar", func(w http.ResponseWriter, r *http.Request) {
		ifi, err := netinfo.LocalInterface()
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		writeJSON(w, http.StatusOK, models.RadarResult{Interface: ifi, Hosts: runRadar(ifi)})
	})

	mux.HandleFunc("POST /api/network/portfinder", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			DeviceIP  string `json:"deviceIp"`
			TargetMAC string `json:"targetMac"`
		}
		if !decodeJSON(w, r, &body) {
			return
		}
		snap, err := v.Snapshot()
		if err != nil {
			writeLocked(w, err)
			return
		}
		if len(snap.SNMP) == 0 {
			writeError(w, http.StatusBadRequest, errNoCommunity)
			return
		}
		ifi, _ := netinfo.LocalInterface()
		device := body.DeviceIP
		if device == "" {
			device, _ = netinfo.DefaultGateway()
		}
		target := body.TargetMAC
		if target == "" {
			target = ifi.MAC
		}
		loc, err := locateViaCommunities(device, target, snap.SNMP)
		if err != nil {
			writeError(w, http.StatusBadGateway, err)
			return
		}
		writeJSON(w, http.StatusOK, loc)
	})
}

// runRadar construit la topologie L2 à partir de la table ARP (vérité niveau 2).
// Le sweep du sous-réseau sert uniquement à réchauffer le cache ARP : les hôtes
// réels répondent en L2 et y apparaissent. Les résultats du sweep ne créent
// PAS de nœuds (certains points d'accès acceptent le TCP pour toutes les IP,
// ce qui produirait des hôtes fantômes).
func runRadar(ifi models.InterfaceInfo) []models.Host {
	if ifi.CIDR != "" {
		if hosts, err := radar.HostsInCIDR(ifi.CIDR, 1024); err == nil {
			radar.Sweep(hosts, radar.NewProber(), 64) // effet de bord : peuple l'ARP
		}
	}

	out := []models.Host{}
	for _, n := range readARP() {
		if isMulticastOrBroadcastMAC(n.MAC) {
			continue // ignore les entrées multicast/broadcast (224.x, 239.x, 255.x…)
		}
		out = append(out, models.Host{
			IP: n.IP, MAC: n.MAC, Vendor: oui.Vendor(n.MAC), Alive: true, Source: "arp",
		})
	}
	return out
}

// isMulticastOrBroadcastMAC repère les adresses L2 non-unicast à exclure de la
// topologie : multicast IPv4 (01:00:5e), IPv6 (33:33), broadcast (ff:ff:…).
func isMulticastOrBroadcastMAC(mac string) bool {
	return strings.HasPrefix(mac, "01:00:5e") ||
		strings.HasPrefix(mac, "33:33") ||
		strings.HasPrefix(mac, "ff:ff:ff:ff:ff:ff")
}

func readARP() []arp.Neighbor {
	n, err := arp.Read()
	if err != nil {
		return nil
	}
	return n
}

// locateViaCommunities essaie chaque community jusqu'à localiser la MAC.
func locateViaCommunities(device, targetMAC string, comms []models.SNMPCredential) (models.PortLocation, error) {
	var lastErr error
	for _, c := range comms {
		client, closeFn, err := portfinder.NewGoSNMP(device, c.Community)
		if err != nil {
			lastErr = err
			continue
		}
		loc, err := portfinder.Locate(client, targetMAC)
		_ = closeFn()
		if err == nil {
			return loc, nil
		}
		lastErr = err
	}
	if lastErr == nil {
		lastErr = errors.New("localisation impossible")
	}
	return models.PortLocation{}, lastErr
}

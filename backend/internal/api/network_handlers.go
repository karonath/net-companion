package api

import (
	"errors"
	"net/http"

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

// runRadar combine la table ARP (avec fabricant) et un sweep du sous-réseau.
func runRadar(ifi models.InterfaceInfo) []models.Host {
	seen := map[string]*models.Host{}

	for _, n := range readARP() {
		seen[n.IP] = &models.Host{IP: n.IP, MAC: n.MAC, Vendor: oui.Vendor(n.MAC), Alive: true, Source: "arp"}
	}

	if ifi.CIDR != "" {
		if hosts, err := radar.HostsInCIDR(ifi.CIDR, 1024); err == nil {
			for _, ip := range radar.Sweep(hosts, radar.NewProber(), 64) {
				if h, ok := seen[ip]; ok {
					h.Alive = true
				} else {
					seen[ip] = &models.Host{IP: ip, Alive: true, Source: "sweep"}
				}
			}
		}
	}

	out := make([]models.Host, 0, len(seen))
	for _, h := range seen {
		out = append(out, *h)
	}
	return out
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

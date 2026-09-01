package api

import (
	"errors"
	"net"
	"net/http"
	"strings"
	"time"

	"netcompanion/internal/models"
	"netcompanion/internal/network/arp"
	"netcompanion/internal/network/netinfo"
	"netcompanion/internal/network/neighbors"
	"netcompanion/internal/network/oui"
	"netcompanion/internal/network/portfinder"
	"netcompanion/internal/network/radar"
	"netcompanion/internal/sim"
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

	mux.HandleFunc("GET /api/network/host", func(w http.ResponseWriter, r *http.Request) {
		ip := r.URL.Query().Get("ip")
		if ip == "" {
			writeError(w, http.StatusBadRequest, errors.New("paramètre ip requis"))
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"ip":        ip,
			"hostname":  netinfo.ReverseDNS(ip),
			"latencyMs": quickRTTms(ip),
		})
	})

	mux.HandleFunc("POST /api/network/neighbors", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			DeviceIP string `json:"deviceIp"`
			Demo     bool   `json:"demo"`
		}
		if !decodeJSON(w, r, &body) {
			return
		}
		if body.Demo {
			writeJSON(w, http.StatusOK, map[string]any{"neighbors": neighbors.FromSNMP(sim.DemoSNMPClient())})
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
		device := body.DeviceIP
		if device == "" {
			device, _ = netinfo.DefaultGateway()
		}
		ns, err := neighborsViaCredentials(device, snap.SNMP)
		if err != nil {
			writeError(w, http.StatusBadGateway, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"neighbors": ns})
	})

	mux.HandleFunc("POST /api/network/portfinder", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			DeviceIP  string `json:"deviceIp"`
			TargetMAC string `json:"targetMac"`
			Demo      bool   `json:"demo"`
		}
		if !decodeJSON(w, r, &body) {
			return
		}
		// Mode démo : interroge le switch simulé (client SNMP simulé), sans coffre.
		if body.Demo {
			target := body.TargetMAC
			if target == "" {
				target = sim.DemoMAC
			}
			loc, err := portfinder.Locate(sim.DemoSNMPClient(), target)
			if err != nil {
				writeError(w, http.StatusBadGateway, err)
				return
			}
			writeJSON(w, http.StatusOK, loc)
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

// quickRTTms mesure un RTT TCP best-effort vers l'hôte (ms), -1 si injoignable.
func quickRTTms(ip string) int {
	for _, port := range []string{"443", "80", "22", "53"} {
		start := time.Now()
		conn, err := net.DialTimeout("tcp", net.JoinHostPort(ip, port), 700*time.Millisecond)
		if err == nil {
			_ = conn.Close()
			return int(time.Since(start).Milliseconds())
		}
	}
	return -1
}

func readARP() []arp.Neighbor {
	n, err := arp.Read()
	if err != nil {
		return nil
	}
	return n
}

// neighborsViaCredentials essaie chaque credential SNMP jusqu'à obtenir des voisins.
func neighborsViaCredentials(device string, comms []models.SNMPCredential) ([]neighbors.Neighbor, error) {
	var lastErr error
	for _, c := range comms {
		client, closeFn, err := portfinder.NewGoSNMP(device, c)
		if err != nil {
			lastErr = err
			continue
		}
		ns := neighbors.FromSNMP(client)
		_ = closeFn()
		return ns, nil
	}
	if lastErr == nil {
		lastErr = errors.New("découverte des voisins impossible")
	}
	return nil, lastErr
}

// locateViaCommunities essaie chaque community jusqu'à localiser la MAC.
func locateViaCommunities(device, targetMAC string, comms []models.SNMPCredential) (models.PortLocation, error) {
	var lastErr error
	for _, c := range comms {
		client, closeFn, err := portfinder.NewGoSNMP(device, c)
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

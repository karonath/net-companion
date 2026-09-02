package api

import (
	"errors"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"netcompanion/internal/models"
	"netcompanion/internal/network/arp"
	"netcompanion/internal/network/arptable"
	"netcompanion/internal/network/discovery"
	"netcompanion/internal/network/netinfo"
	"netcompanion/internal/network/neighbors"
	"netcompanion/internal/network/oui"
	"netcompanion/internal/network/portfinder"
	"netcompanion/internal/network/radar"
	"netcompanion/internal/network/sysinfo"
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

	mux.HandleFunc("GET /api/network/interfaces", func(w http.ResponseWriter, r *http.Request) {
		auto := ""
		if ifi, err := netinfo.LocalInterface(); err == nil {
			auto = ifi.Name
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"interfaces": netinfo.ListInterfaces(),
			"auto":       auto,
		})
	})

	mux.HandleFunc("GET /api/network/radar", func(w http.ResponseWriter, r *http.Request) {
		// Mode démo : topologie d'entreprise simulée (tout le logiciel bascule dessus).
		if sim.Current().Enabled {
			writeJSON(w, http.StatusOK, sim.DemoRadar())
			return
		}
		ifi, err := resolveInterface(r.URL.Query().Get("iface"))
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		writeJSON(w, http.StatusOK, models.RadarResult{Interface: ifi, Hosts: runRadar(ifi, v)})
	})

	mux.HandleFunc("GET /api/network/publicip", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"ip": publicIP()})
	})

	mux.HandleFunc("GET /api/network/host", func(w http.ResponseWriter, r *http.Request) {
		ip := r.URL.Query().Get("ip")
		if ip == "" {
			writeError(w, http.StatusBadRequest, errors.New("paramètre ip requis"))
			return
		}
		// Mode démo : infos simulées pour les hôtes du parc de démonstration.
		if sim.Current().Enabled {
			if hostname, latency, ok := sim.DemoHostInfo(ip); ok {
				writeJSON(w, http.StatusOK, map[string]any{"ip": ip, "hostname": hostname, "latencyMs": latency})
				return
			}
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
		if body.Demo || sim.Current().Enabled {
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
		if body.Demo || sim.Current().Enabled {
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

// runRadar cartographie le réseau de la façon la plus complète possible :
//  1. balayage ARP actif (SendARP, Windows) : capte tout ce qui répond à l'ARP,
//     même sans port ouvert (objets, imprimantes, caméras réveillés) ;
//  2. table ARP de l'OS (ce que le système a déjà résolu) ;
//  3. inventaire SNMP de la passerelle (entreprise) : table ARP du routeur/switch,
//     incluant les appareils que le PC ne peut pas joindre directement ;
//  4. identification précise (mDNS/SSDP/NetBIOS/bannières) pour nommer et typer.
// Un appareil totalement muet (téléphone en veille profonde qui ignore l'ARP)
// n'est visible que par l'étape 3 (la passerelle le connaît).
func runRadar(ifi models.InterfaceInfo, v *vault.Vault) []models.Host {
	byIP := map[string]*models.Host{}
	var order []string
	// add insère ou complète un hôte (dédup par IP), en écartant multicast/broadcast.
	add := func(ip, mac, source string) *models.Host {
		if mac != "" && isMulticastOrBroadcastMAC(mac) {
			return nil
		}
		h, ok := byIP[ip]
		if !ok {
			h = &models.Host{IP: ip, Alive: true, Source: source}
			byIP[ip] = h
			order = append(order, ip)
		}
		if h.MAC == "" && mac != "" {
			h.MAC = mac
			h.Vendor = oui.Vendor(mac)
		}
		h.Sources = mergeStrings(h.Sources, []string{source})
		return h
	}

	// Liste des IP du sous-réseau à sonder.
	var subnetIPs []string
	if ifi.CIDR != "" {
		if hosts, err := radar.HostsInCIDR(ifi.CIDR, 1024); err == nil {
			subnetIPs = hosts
		}
	}

	// 1) Balayage ARP actif (autorité L2 directe : IP + MAC). Sous Windows,
	//    SendARP résout l'ARP lui-même : pas besoin du balayage TCP.
	scanned := arp.ScanIPs(subnetIPs)
	for _, n := range scanned {
		add(n.IP, n.MAC, "arp")
	}
	if len(scanned) == 0 && len(subnetIPs) > 0 {
		// Repli (non-Windows) : le balayage TCP réchauffe le cache ARP de l'OS.
		radar.Sweep(subnetIPs, radar.NewProber(), 64)
	}
	// 2) Table ARP du système.
	for _, n := range readARP() {
		add(n.IP, n.MAC, "arp")
	}
	// 3) Inventaire SNMP de la passerelle (silencieux si coffre verrouillé/pas de SNMP).
	for _, e := range gatewayARPViaSNMP(v) {
		add(e.IP, e.MAC, "snmp")
	}

	// 4) Identification précise sur toutes les IP découvertes.
	targets := append([]string{}, order...)
	for _, d := range discovery.Discover(3*time.Second, ifi.IPv4, targets) {
		h, ok := byIP[d.IP]
		if !ok {
			h = &models.Host{IP: d.IP, Alive: true, Source: primarySource(d.Sources)}
			byIP[d.IP] = h
			order = append(order, d.IP)
		}
		fillHostString(&h.Name, d.Name)
		fillHostString(&h.Hostname, d.Hostname)
		fillHostString(&h.Model, d.Model)
		fillHostString(&h.Manufacturer, d.Manufacturer)
		fillHostString(&h.DeviceType, d.DeviceType)
		h.Services = mergeStrings(h.Services, d.Services)
		h.Sources = mergeStrings(h.Sources, d.Sources)
	}

	// 5) Classification SNMP (entreprise) : type fiable via sysDescr/sysServices
	//    sur les équipements qui répondent au SNMP. Silencieux sinon.
	classifyViaSNMP(byIP, v)

	// Reverse-DNS best-effort + inférence de type en dernier recours.
	for _, h := range byIP {
		if h.Hostname == "" {
			if rev := netinfo.ReverseDNS(h.IP); rev != "" {
				h.Hostname = rev
			}
		}
		if h.DeviceType == "" {
			h.DeviceType = discovery.InferType(h.Name, h.Hostname, h.Model, h.Manufacturer, h.Vendor)
		}
	}

	out := make([]models.Host, 0, len(order))
	for _, ip := range order {
		out = append(out, *byIP[ip])
	}
	return out
}

func fillHostString(dst *string, v string) {
	if *dst == "" && v != "" {
		*dst = v
	}
}

// mergeStrings unionne deux listes en éliminant les doublons (ordre préservé).
func mergeStrings(a, b []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, s := range append(append([]string{}, a...), b...) {
		if s == "" || seen[s] {
			continue
		}
		seen[s] = true
		out = append(out, s)
	}
	return out
}

// primarySource choisit la source la plus « parlante » comme origine principale.
func primarySource(sources []string) string {
	for _, pref := range []string{"mdns", "ssdp", "nbns", "banner"} {
		for _, s := range sources {
			if s == pref {
				return pref
			}
		}
	}
	if len(sources) > 0 {
		return sources[0]
	}
	return "discovery"
}

// isMulticastOrBroadcastMAC repère les adresses L2 non-unicast à exclure de la
// topologie : multicast IPv4 (01:00:5e), IPv6 (33:33), broadcast (ff:ff:…).
func isMulticastOrBroadcastMAC(mac string) bool {
	return strings.HasPrefix(mac, "01:00:5e") ||
		strings.HasPrefix(mac, "33:33") ||
		strings.HasPrefix(mac, "ff:ff:ff:ff:ff:ff")
}

// publicIP récupère l'IP publique en best-effort (timeout court), "" si échec.
func publicIP() string {
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get("https://api.ipify.org")
	if err != nil {
		return ""
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(io.LimitReader(resp.Body, 64))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(b))
}

// resolveInterface renvoie l'interface demandée par nom (Mode Universel), ou
// l'auto-détection si le nom est vide/introuvable.
func resolveInterface(name string) (models.InterfaceInfo, error) {
	if name != "" {
		if ifi, ok := netinfo.InterfaceByName(name); ok {
			return ifi, nil
		}
	}
	return netinfo.LocalInterface()
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

// classifyViaSNMP interroge chaque hôte en SNMP (sysDescr/sysServices) pour un
// typage fiable en entreprise (switch/routeur/pare-feu/serveur…). Le type SNMP
// fait autorité (écrase le type deviné). Ne fait rien si le coffre est verrouillé
// ou sans community. Session courte, sans réessai, sondage borné en parallèle.
func classifyViaSNMP(byIP map[string]*models.Host, v *vault.Vault) {
	snap, err := v.Snapshot()
	if err != nil || len(snap.SNMP) == 0 {
		return
	}
	sem := make(chan struct{}, 48)
	var wg sync.WaitGroup
	var mu sync.Mutex
	for ip, h := range byIP {
		wg.Add(1)
		go func(ip string, h *models.Host) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			for _, c := range snap.SNMP {
				client, closeFn, err := portfinder.NewGoSNMPFast(ip, c, 700*time.Millisecond)
				if err != nil {
					continue
				}
				info, ok := sysinfo.FromSNMP(client)
				_ = closeFn()
				if !ok {
					continue
				}
				mu.Lock()
				if info.DeviceType != "" {
					h.DeviceType = info.DeviceType // SNMP fait autorité
				}
				if h.Name == "" && info.SysName != "" {
					h.Name = info.SysName
				}
				h.Sources = mergeStrings(h.Sources, []string{"snmp"})
				mu.Unlock()
				return
			}
		}(ip, h)
	}
	wg.Wait()
}

// gatewayARPViaSNMP lit la table ARP de la passerelle par SNMP (inventaire
// exhaustif entreprise). Renvoie nil en silence si le coffre est verrouillé,
// s'il n'y a aucune community, ou si la passerelle ne répond pas au SNMP.
func gatewayARPViaSNMP(v *vault.Vault) []arptable.Entry {
	snap, err := v.Snapshot()
	if err != nil || len(snap.SNMP) == 0 {
		return nil
	}
	device, err := netinfo.DefaultGateway()
	if err != nil || device == "" {
		return nil
	}
	for _, c := range snap.SNMP {
		client, closeFn, err := portfinder.NewGoSNMP(device, c)
		if err != nil {
			continue
		}
		entries := arptable.FromSNMP(client)
		_ = closeFn()
		if len(entries) > 0 {
			return entries
		}
	}
	return nil
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

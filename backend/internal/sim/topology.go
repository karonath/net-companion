package sim

import (
	"strings"

	"netcompanion/internal/models"
	"netcompanion/internal/network/diag"
)

// DemoGateway est la passerelle (pare-feu) du réseau d'entreprise simulé.
const DemoGateway = "10.10.0.1"

// DemoInterface est l'interface présentée à l'UI en mode démo.
func DemoInterface() models.InterfaceInfo {
	return models.InterfaceInfo{
		Name: "eth0 (démo)", MAC: "02:00:00:00:00:64",
		IPv4: "10.10.0.100", CIDR: "10.10.0.100/24",
	}
}

// DemoHosts renvoie un parc d'entreprise réaliste et varié pour la démonstration,
// avec sa hiérarchie L2 (uplink) : FW → SW-CORE → {serveurs, SW-DIST} → postes.
func DemoHosts() []models.Host {
	h := func(ip, mac, vendor, name, model, typ, uplink string, services ...string) models.Host {
		return models.Host{
			IP: ip, MAC: mac, Vendor: vendor, Name: name, Hostname: name,
			Model: model, DeviceType: typ, Uplink: uplink, Services: services,
			Alive: true, Source: "démo", Sources: []string{"démo"},
		}
	}
	const (
		fw   = DemoGateway // 10.10.0.1
		core = "10.10.0.2"
		dist = "10.10.0.3"
	)
	return []models.Host{
		h(fw, "10:e9:92:00:00:01", "Fortinet", "FW-EDGE-01", "FortiGate-60F", "pare-feu", "", "tcp/443", "tcp/22"),
		h(core, "00:1a:2b:00:00:02", "Cisco", "SW-CORE-01", "Catalyst 9300", "switch", fw, "tcp/22", "tcp/443"),
		h(dist, "00:1a:2b:00:00:03", "Cisco", "SW-DIST-02", "Catalyst 2960-X", "switch", core, "tcp/22"),
		h("10.10.0.10", "00:50:56:00:00:10", "VMware", "ESXI-01", "VMware ESXi 7.0", "hyperviseur", core, "tcp/443", "tcp/902"),
		h("10.10.0.11", "00:15:5d:00:00:11", "Microsoft", "SRV-DC01", "Windows Server 2022", "serveur", core, "tcp/445", "tcp/3389", "tcp/53"),
		h("10.10.0.12", "00:15:5d:00:00:12", "Dell", "SRV-APP02", "Ubuntu Server 22.04", "serveur", core, "tcp/22", "tcp/443"),
		h("10.10.0.20", "00:11:32:00:00:20", "Synology", "NAS-01", "DiskStation DS1821+", "NAS / stockage", core, "tcp/445", "tcp/5001"),
		h("10.10.0.80", "00:c0:b7:00:00:80", "APC", "UPS-SALLE-SERVEUR", "Smart-UPS 1500", "onduleur", core, "tcp/161"),
		h("10.10.0.30", "98:aa:fc:00:00:30", "Aruba", "AP-FLOOR1", "Aruba AP-315", "point d'accès", dist, "tcp/443"),
		h("10.10.0.40", "00:1b:78:00:00:40", "HP", "PRT-COMPTA", "HP LaserJet M609", "imprimante", dist, "tcp/9100", "tcp/631"),
		h("10.10.0.50", "00:04:f2:00:00:50", "Polycom", "VOIP-101", "Polycom VVX 411", "téléphone VoIP", dist, "tcp/5060"),
		h("10.10.0.60", DemoMAC, "Dell", "PC-DUPONT", "OptiPlex 7090", "ordinateur", dist, "tcp/445", "tcp/3389"),
		h("10.10.0.61", "b8:ca:3a:00:00:61", "Dell", "PC-MARTIN", "Latitude 5540", "ordinateur", dist, "tcp/445"),
		h("10.10.0.70", "00:40:8c:00:00:70", "Axis", "CAM-LOBBY", "Axis P3245-LVE", "caméra", dist, "tcp/554", "tcp/80"),
	}
}

// DemoRadar renvoie le résultat radar complet du réseau d'entreprise simulé.
func DemoRadar() models.RadarResult {
	return models.RadarResult{Interface: DemoInterface(), Hosts: DemoHosts()}
}

// demoHost renvoie l'hôte simulé d'IP donnée, s'il existe.
func demoHost(ip string) (models.Host, bool) {
	for _, hst := range DemoHosts() {
		if hst.IP == ip {
			return hst, true
		}
	}
	return models.Host{}, false
}

// DemoHostInfo renvoie le nom et une latence simulée pour un hôte de démo.
func DemoHostInfo(ip string) (hostname string, latencyMs int, ok bool) {
	if hst, found := demoHost(ip); found {
		return hst.Hostname, 3, true
	}
	return "", -1, false
}

// DemoHostDiag renvoie un diagnostic ciblé simulé pour un hôte de démo.
func DemoHostDiag(ip string) []diag.Check {
	hst, found := demoHost(ip)
	if !found {
		return nil
	}
	return []diag.Check{
		{Name: "Hôte joignable", Status: diag.StatusOK, Detail: ip + " répond (port " + firstPort(hst.Services) + ") [démo]"},
		{Name: "Latence vers l'hôte", Status: diag.StatusOK, Detail: ip + " : 3 ms (jitter 1 ms, perte 0%) [démo]"},
		{Name: "Ports ouverts", Status: diag.StatusOK, Detail: strings.Join(hst.Services, ", ") + " [démo]"},
		{Name: "Nom d'hôte", Status: diag.StatusOK, Detail: hst.Hostname},
	}
}

// DemoDiagSuite renvoie un bilan de connectivité complet simulé (tout au vert).
func DemoDiagSuite() []diag.Check {
	return []diag.Check{
		{Name: "Interface locale", Status: diag.StatusOK, Detail: "10.10.0.100 sur eth0 (démo) (10.10.0.100/24), MTU 1500"},
		{Name: "Serveurs DNS", Status: diag.StatusOK, Detail: "10.10.0.11 (2 ms), 1.1.1.1 (10 ms) [démo]"},
		{Name: "Passerelle joignable", Status: diag.StatusOK, Detail: DemoGateway + " joignable (port 443) [démo]"},
		{Name: "Latence passerelle", Status: diag.StatusOK, Detail: DemoGateway + ":443 : 1 ms (jitter 0 ms, perte 0%) [démo]"},
		{Name: "Résolution DNS", Status: diag.StatusOK, Detail: "example.com → 93.184.216.34 (IPv4 + IPv6) en 12 ms [démo]"},
		{Name: "Accès Internet", Status: diag.StatusOK, Detail: "connecté à 1.1.1.1:443 [démo]"},
		{Name: "Latence Internet", Status: diag.StatusOK, Detail: "1.1.1.1:443 : 9 ms (jitter 2 ms, perte 0%) [démo]"},
		{Name: "Perte de paquets", Status: diag.StatusOK, Detail: "0% de perte vers 1.1.1.1 (5 paquets ICMP) [démo]"},
		{Name: "Ports sortants", Status: diag.StatusOK, Detail: "53, 80, 443 ouverts en sortie [démo]"},
		{Name: "IPv6", Status: diag.StatusOK, Detail: "actif (adresse globale + sortie Internet) [démo]"},
		{Name: "Portail captif", Status: diag.StatusOK, Detail: "aucun (accès Internet direct) [démo]"},
		{Name: "IP publique (WAN)", Status: diag.StatusOK, Detail: "203.0.113.42 [démo]"},
		{Name: "Débit descendant (estimation)", Status: diag.StatusOK, Detail: "≈ 940 Mbps (2.0 Mo en 0.0s) [démo]"},
	}
}

func firstPort(services []string) string {
	if len(services) == 0 {
		return "?"
	}
	return strings.TrimPrefix(services[0], "tcp/")
}

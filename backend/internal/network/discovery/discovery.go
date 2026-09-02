// Package discovery identifie précisément les appareils d'un réseau (domestique
// ou entreprise) en combinant cinq sources actives, sans dépendance externe ni
// privilège : mDNS/Bonjour (5353), SSDP/UPnP (1900 + fiche XML), NetBIOS (137),
// bannières de services TCP, et l'OUI/reverse-DNS côté appelant.
package discovery

import (
	"net"
	"sort"
	"strings"
	"sync"
	"time"
)

// Device est un appareil identifié. Les champs sont fusionnés depuis toutes les
// sources qui ont vu l'IP ; Sources liste les techniques qui ont contribué.
type Device struct {
	IP           string   `json:"ip"`
	Name         string   `json:"name,omitempty"`         // nom convivial (mDNS fn/instance, UPnP friendlyName, NetBIOS)
	Hostname     string   `json:"hostname,omitempty"`     // nom d'hôte technique (mDNS A, reverse-DNS)
	Model        string   `json:"model,omitempty"`        // modèle matériel (mDNS TXT, UPnP modelName)
	Manufacturer string   `json:"manufacturer,omitempty"` // constructeur (UPnP manufacturer, bannière)
	DeviceType   string   `json:"deviceType,omitempty"`   // catégorie devinée (ordinateur, imprimante, TV…)
	Services     []string `json:"services,omitempty"`     // services annoncés / ports ouverts
	Sources      []string `json:"sources,omitempty"`      // "mdns" | "ssdp" | "nbns" | "banner"
}

// Discover lance les quatre sondes réseau en parallèle (liées à l'interface
// active srcIP) et fusionne les résultats par IP. hosts (issus d'ARP) est la
// liste des IP à sonder activement en NetBIOS/bannières ; peut être nil.
func Discover(timeout time.Duration, srcIP string, hosts []string) []Device {
	var wg sync.WaitGroup
	var mdns, ssdp, nbns, banners []Device
	wg.Add(4)
	go func() { defer wg.Done(); mdns = MDNS(timeout, srcIP) }()
	go func() { defer wg.Done(); ssdp = SSDP(timeout, srcIP) }()
	go func() { defer wg.Done(); nbns = NBNS(timeout, srcIP, hosts) }()
	go func() { defer wg.Done(); banners = Banners(timeout, hosts) }()
	wg.Wait()

	all := make([]Device, 0, len(mdns)+len(ssdp)+len(nbns)+len(banners))
	all = append(all, mdns...)
	all = append(all, ssdp...)
	all = append(all, nbns...)
	all = append(all, banners...)
	return Merge(all)
}

// Merge fusionne des Device par IP : chaque champ texte prend la première valeur
// non vide, les Services et Sources sont unionnés puis triés.
func Merge(in []Device) []Device {
	byIP := map[string]*Device{}
	var order []string
	for _, d := range in {
		if d.IP == "" {
			continue
		}
		cur, ok := byIP[d.IP]
		if !ok {
			cp := d
			cp.Services = uniqueSorted(cp.Services)
			cp.Sources = uniqueSorted(cp.Sources)
			byIP[d.IP] = &cp
			order = append(order, d.IP)
			continue
		}
		fillString(&cur.Name, d.Name)
		fillString(&cur.Hostname, d.Hostname)
		fillString(&cur.Model, d.Model)
		fillString(&cur.Manufacturer, d.Manufacturer)
		fillString(&cur.DeviceType, d.DeviceType)
		cur.Services = uniqueSorted(append(cur.Services, d.Services...))
		cur.Sources = uniqueSorted(append(cur.Sources, d.Sources...))
	}
	out := make([]Device, 0, len(order))
	for _, ip := range order {
		out = append(out, *byIP[ip])
	}
	return out
}

func fillString(dst *string, v string) {
	if *dst == "" && v != "" {
		*dst = v
	}
}

func uniqueSorted(in []string) []string {
	if len(in) == 0 {
		return nil
	}
	seen := map[string]bool{}
	var out []string
	for _, s := range in {
		s = strings.TrimSpace(s)
		if s == "" || seen[s] {
			continue
		}
		seen[s] = true
		out = append(out, s)
	}
	sort.Strings(out)
	return out
}

// classifyService devine une catégorie d'appareil depuis un identifiant de
// service (mDNS "_googlecast._tcp", port "tcp/9100", etc.). Renvoie "" si inconnu.
func classifyService(svc string) string {
	s := strings.ToLower(svc)
	switch {
	case has(s, "_googlecast", "_airplay", "_raop", "_spotify-connect", "_sonos", "roku", "_androidtvremote"):
		return "TV / média"
	case has(s, "_ipp", "_ipps", "_printer", "_pdl-datastream", "tcp/9100", "tcp/515", "tcp/631"):
		return "imprimante"
	case has(s, "_scanner", "_uscan"):
		return "scanner"
	case has(s, "_afpovertcp", "_nfs", "_adisk", "nas"):
		return "NAS / stockage"
	case has(s, "_smb", "tcp/445", "tcp/139", "_ssh", "_sftp-ssh", "tcp/22"):
		return "ordinateur"
	case has(s, "_homekit", "_hap", "_hue", "_matter", "_miio"):
		return "objet connecté"
	case has(s, "_apple-mobdev", "_companion-link", "iphone", "ipad", "android"):
		return "smartphone / tablette"
	case has(s, "internetgatewaydevice", "gateway", "router", "_ubnt"):
		return "routeur / box"
	}
	return ""
}

// InferType devine une catégorie d'appareil à partir de tout indice textuel
// collecté (nom, hôte, modèle, constructeur, fabricant OUI). Dernier recours
// quand aucun service n'a livré de type. Renvoie "" si rien ne correspond.
func InferType(hints ...string) string {
	s := strings.ToLower(strings.Join(hints, " "))
	switch {
	// --- Équipements d'infrastructure (entreprise) : préfixes de nom + marques ---
	case has(s, "fw-", "-fw", "firewall", "fortinet", "fortigate", "palo alto", "sonicwall", "pfsense"):
		return "pare-feu"
	case has(s, "ap-", "-ap", "access-point", "accesspoint", "aruba", "unifi", "meraki mr", "aironet"):
		return "point d'accès"
	case has(s, "sw-", "-sw", "switch", "catalyst", "nexus", "procurve"):
		return "switch"
	case has(s, "esxi", "vmware", "proxmox", "hyper-v", "vsphere"):
		return "hyperviseur"
	case has(s, "srv-", "-srv", "server", "serveur"):
		return "serveur"
	case has(s, "voip", "phone-", "-phone", "polycom", "yealink", "sip"):
		return "téléphone VoIP"
	case has(s, "ups-", "-ups", "onduleur", "smart-ups", "apc ", "eaton"):
		return "onduleur"
	case has(s, "plc", "siemens", "simatic", "modbus", "scada", "rockwell", "allen-bradley"):
		return "automate / OT"
	// --- Grand public / mixte ---
	case has(s, "nintendo", "playstation", "sony interactive", "xbox", "steam"):
		return "console de jeu"
	case has(s, "washer", "fridge", "refriger", "dishwash", "oven", "microwave", "cooktop", "laundry", "vacuum", "roborock"):
		return "électroménager"
	case has(s, "prt-", "-prt", "printer", "laserjet", "officejet", "deskjet", "ecotank", "brother", "canon", "lexmark", "kyocera", "jetdirect", "zebra"):
		return "imprimante"
	case has(s, "chromecast", "roku", "appletv", "apple tv", "bravia", "firetv", "shield", "smart-tv", "smarttv", " tv", "-tv", "webos", "tizen"):
		return "TV / média"
	case has(s, "iphone", "ipad", "android", "pixel", "galaxy", "oneplus", "xiaomi", "huawei", "phone"):
		return "smartphone / tablette"
	case has(s, "camera", "webcam", "ipcam", "hikvision", "dahua", "axis", "reolink", "nest cam", "ring"):
		return "caméra"
	case has(s, "livebox", "freebox", "bbox", "fritz", "rtr-", "-rtr", "router", "routeur", "gateway", "gw-", "sagemcom", "netgear", "tp-link", "mikrotik"):
		return "routeur / box"
	case has(s, "synology", "qnap", "nas", "truenas", "diskstation"):
		return "NAS / stockage"
	case has(s, "raspberr", "esp32", "esp8266", "tasmota", "shelly", "sonoff", "tuya", "smartthings", "samjin"):
		return "objet connecté"
	case has(s, "macbook", "imac", "thinkpad", "desktop", "laptop", "-pc", "workstation"):
		return "ordinateur"
	}
	return ""
}

func has(s string, subs ...string) bool {
	for _, sub := range subs {
		if strings.Contains(s, sub) {
			return true
		}
	}
	return false
}

// bindIP renvoie l'adresse d'écoute liée à l'interface active (pour que le
// multicast sorte par la bonne carte), ou 0.0.0.0 si srcIP est vide/invalide.
func bindIP(srcIP string) net.IP {
	if ip := net.ParseIP(srcIP); ip != nil {
		return ip
	}
	return net.IPv4zero
}

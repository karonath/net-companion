package models

// InterfaceInfo décrit l'interface réseau active de l'hôte.
type InterfaceInfo struct {
	Name string `json:"name"`
	MAC  string `json:"mac"`
	IPv4 string `json:"ipv4"`
	CIDR string `json:"cidr"` // ex: 192.168.1.10/24
}

// Host est un hôte découvert par le Radar, enrichi par les sondes d'identification.
type Host struct {
	IP           string   `json:"ip"`
	MAC          string   `json:"mac,omitempty"`
	Vendor       string   `json:"vendor,omitempty"`       // fabricant OUI (adresse MAC)
	Name         string   `json:"name,omitempty"`         // nom convivial (mDNS/UPnP/NetBIOS)
	Hostname     string   `json:"hostname,omitempty"`     // nom d'hôte technique (mDNS A/reverse-DNS)
	Model        string   `json:"model,omitempty"`        // modèle matériel (mDNS TXT/UPnP)
	Manufacturer string   `json:"manufacturer,omitempty"` // constructeur (UPnP/bannière)
	DeviceType   string   `json:"deviceType,omitempty"`   // catégorie (ordinateur, imprimante, TV…)
	Services     []string `json:"services,omitempty"`     // services/ports détectés
	Alive        bool     `json:"alive"`
	Source       string   `json:"source"`  // origine principale : "arp", "mdns", "ssdp"…
	Sources      []string `json:"sources,omitempty"` // toutes les sondes ayant vu l'hôte
}

// RadarResult agrège les hôtes vus + le contexte local.
type RadarResult struct {
	Interface InterfaceInfo `json:"interface"`
	Hosts     []Host        `json:"hosts"`
}

// PortLocation est le résultat du Port-Finder (localisation physique).
type PortLocation struct {
	Device     string `json:"device"`
	PortIfName string `json:"portIfName"`
	BridgePort int    `json:"bridgePort"`
	IfIndex    int    `json:"ifIndex"`
	VLAN       int    `json:"vlan"`
	Sentence   string `json:"sentence"`
}

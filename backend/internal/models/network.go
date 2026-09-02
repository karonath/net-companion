package models

// InterfaceInfo décrit l'interface réseau active de l'hôte.
type InterfaceInfo struct {
	Name string `json:"name"`
	MAC  string `json:"mac"`
	IPv4 string `json:"ipv4"`
	CIDR string `json:"cidr"` // ex: 192.168.1.10/24
}

// Host est un hôte découvert par le Radar.
type Host struct {
	IP     string `json:"ip"`
	MAC    string `json:"mac,omitempty"`
	Vendor string `json:"vendor,omitempty"`
	Name   string `json:"name,omitempty"`  // nom d'hôte (mDNS/reverse-DNS)
	Model  string `json:"model,omitempty"` // modèle (SSDP/UPnP)
	Alive  bool   `json:"alive"`
	Source string `json:"source"` // "arp", "sweep", "mdns", "ssdp"
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

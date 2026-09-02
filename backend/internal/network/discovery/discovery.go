// Package discovery cartographie un réseau non managé via des protocoles ouverts :
// SSDP/UPnP (port 1900) et mDNS/Bonjour (port 5353). Complète l'ARP/OUI.
package discovery

import (
	"net"
	"strings"
	"sync"
	"time"

	"golang.org/x/net/dns/dnsmessage"
)

// Device est un appareil découvert par un protocole ouvert.
type Device struct {
	IP     string `json:"ip"`
	Name   string `json:"name,omitempty"`   // nom d'hôte / instance (mDNS)
	Model  string `json:"model,omitempty"`  // SERVER/modèle (SSDP)
	Source string `json:"source"`           // "ssdp" | "mdns"
}

// parseSSDP extrait le modèle (SERVER) d'une réponse SSDP.
func parseSSDP(raw string) (server string) {
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimRight(line, "\r")
		if i := strings.IndexByte(line, ':'); i > 0 {
			key := strings.ToUpper(strings.TrimSpace(line[:i]))
			val := strings.TrimSpace(line[i+1:])
			if key == "SERVER" {
				return val
			}
		}
	}
	return ""
}

// bindIP renvoie l'adresse d'écoute liée à l'interface active (pour que le
// multicast sorte par la bonne carte), ou 0.0.0.0 si srcIP est vide/invalide.
func bindIP(srcIP string) net.IP {
	if ip := net.ParseIP(srcIP); ip != nil {
		return ip
	}
	return net.IPv4zero
}

// SSDP envoie un M-SEARCH et collecte les équipements qui répondent.
func SSDP(timeout time.Duration, srcIP string) []Device {
	group := &net.UDPAddr{IP: net.IPv4(239, 255, 255, 250), Port: 1900}
	conn, err := net.ListenUDP("udp4", &net.UDPAddr{IP: bindIP(srcIP), Port: 0})
	if err != nil {
		return nil
	}
	defer conn.Close()

	msg := "M-SEARCH * HTTP/1.1\r\n" +
		"HOST: 239.255.255.250:1900\r\n" +
		"MAN: \"ssdp:discover\"\r\n" +
		"MX: 1\r\n" +
		"ST: ssdp:all\r\n\r\n"
	if _, err := conn.WriteToUDP([]byte(msg), group); err != nil {
		return nil
	}

	_ = conn.SetReadDeadline(time.Now().Add(timeout))
	seen := map[string]Device{}
	buf := make([]byte, 2048)
	for {
		n, src, err := conn.ReadFromUDP(buf)
		if err != nil {
			break // deadline atteinte
		}
		ip := src.IP.String()
		if _, ok := seen[ip]; ok {
			continue
		}
		seen[ip] = Device{IP: ip, Model: parseSSDP(string(buf[:n])), Source: "ssdp"}
	}
	return toSlice(seen)
}

// mdnsQuery construit une requête mDNS (bit unicast-response) pour des types courants.
func mdnsQuery() ([]byte, error) {
	types := []string{
		"_services._dns-sd._udp.local.",
		"_http._tcp.local.", "_googlecast._tcp.local.", "_airplay._tcp.local.",
		"_ipp._tcp.local.", "_raop._tcp.local.", "_spotify-connect._tcp.local.",
	}
	b := dnsmessage.NewBuilder(nil, dnsmessage.Header{})
	if err := b.StartQuestions(); err != nil {
		return nil, err
	}
	for _, t := range types {
		name, err := dnsmessage.NewName(t)
		if err != nil {
			continue
		}
		// Class 0x8001 = IN + bit "unicast response" (réponse en unicast vers nous)
		_ = b.Question(dnsmessage.Question{Name: name, Type: dnsmessage.TypePTR, Class: dnsmessage.Class(0x8001)})
	}
	return b.Finish()
}

// parseMDNS extrait les enregistrements A (ip -> nom d'hôte) d'une réponse mDNS.
func parseMDNS(data []byte, out map[string]string) {
	var p dnsmessage.Parser
	if _, err := p.Start(data); err != nil {
		return
	}
	_ = p.SkipAllQuestions()
	collect := func(next func() (dnsmessage.ResourceHeader, error), skip func() error, aRes func() (dnsmessage.AResource, error)) {
		for {
			h, err := next()
			if err != nil {
				return
			}
			if h.Type == dnsmessage.TypeA {
				a, err := aRes()
				if err != nil {
					return
				}
				ip := net.IP(a.A[:]).String()
				name := strings.TrimSuffix(strings.TrimSuffix(h.Name.String(), "."), ".local")
				if name != "" {
					out[ip] = name
				}
			} else if err := skip(); err != nil {
				return
			}
		}
	}
	collect(p.AnswerHeader, p.SkipAnswer, p.AResource)
	collect(p.AdditionalHeader, p.SkipAdditional, p.AResource)
}

// MDNS interroge le réseau en mDNS et renvoie les appareils (nom d'hôte + IP).
func MDNS(timeout time.Duration, srcIP string) []Device {
	group := &net.UDPAddr{IP: net.IPv4(224, 0, 0, 251), Port: 5353}
	conn, err := net.ListenUDP("udp4", &net.UDPAddr{IP: bindIP(srcIP), Port: 0})
	if err != nil {
		return nil
	}
	defer conn.Close()

	q, err := mdnsQuery()
	if err != nil {
		return nil
	}
	if _, err := conn.WriteToUDP(q, group); err != nil {
		return nil
	}

	_ = conn.SetReadDeadline(time.Now().Add(timeout))
	names := map[string]string{}
	buf := make([]byte, 9000)
	for {
		n, _, err := conn.ReadFromUDP(buf)
		if err != nil {
			break
		}
		parseMDNS(buf[:n], names)
	}
	var out []Device
	for ip, name := range names {
		out = append(out, Device{IP: ip, Name: name, Source: "mdns"})
	}
	return out
}

// Discover lance SSDP et mDNS en parallèle (liés à l'interface active srcIP).
func Discover(timeout time.Duration, srcIP string) []Device {
	var wg sync.WaitGroup
	var ssdp, mdns []Device
	wg.Add(2)
	go func() { defer wg.Done(); ssdp = SSDP(timeout, srcIP) }()
	go func() { defer wg.Done(); mdns = MDNS(timeout, srcIP) }()
	wg.Wait()
	return append(ssdp, mdns...)
}

func toSlice(m map[string]Device) []Device {
	out := make([]Device, 0, len(m))
	for _, d := range m {
		out = append(out, d)
	}
	return out
}

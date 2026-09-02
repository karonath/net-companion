package discovery

import (
	"net"
	"strings"
	"time"

	"golang.org/x/net/dns/dnsmessage"
)

// mdnsServiceTypes : services DNS-SD interrogés pour couvrir un large éventail
// d'appareils domestiques et d'entreprise.
var mdnsServiceTypes = []string{
	"_services._dns-sd._udp.local.",
	"_http._tcp.local.", "_https._tcp.local.",
	"_googlecast._tcp.local.", "_airplay._tcp.local.", "_raop._tcp.local.",
	"_spotify-connect._tcp.local.", "_sonos._tcp.local.",
	"_ipp._tcp.local.", "_ipps._tcp.local.", "_printer._tcp.local.",
	"_pdl-datastream._tcp.local.", "_scanner._tcp.local.", "_uscan._tcp.local.",
	"_ssh._tcp.local.", "_sftp-ssh._tcp.local.",
	"_smb._tcp.local.", "_afpovertcp._tcp.local.", "_adisk._tcp.local.",
	"_homekit._tcp.local.", "_hap._tcp.local.", "_hue._tcp.local.",
	"_companion-link._tcp.local.", "_apple-mobdev2._tcp.local.",
	"_workstation._tcp.local.", "_device-info._tcp.local.",
}

// mdnsQuery construit une requête mDNS (bit unicast-response) pour tous les
// types de services suivis.
func mdnsQuery() ([]byte, error) {
	b := dnsmessage.NewBuilder(nil, dnsmessage.Header{})
	if err := b.StartQuestions(); err != nil {
		return nil, err
	}
	for _, t := range mdnsServiceTypes {
		name, err := dnsmessage.NewName(t)
		if err != nil {
			continue
		}
		// Class 0x8001 = IN + bit "unicast response".
		_ = b.Question(dnsmessage.Question{Name: name, Type: dnsmessage.TypePTR, Class: dnsmessage.Class(0x8001)})
	}
	return b.Finish()
}

// mdnsAcc accumule les enregistrements mDNS reçus sur plusieurs paquets, puis
// les corrèle : instance --SRV--> hôte --A--> IP, avec le modèle issu des TXT.
type mdnsAcc struct {
	a   map[string]string            // hôte (minuscule, sans point final) -> IP
	srv map[string]string            // instance -> hôte cible
	txt map[string]map[string]string // instance -> paires clé/valeur
	ptr map[string]string            // instance -> type de service (_googlecast._tcp)
}

func newMDNSAcc() *mdnsAcc {
	return &mdnsAcc{
		a:   map[string]string{},
		srv: map[string]string{},
		txt: map[string]map[string]string{},
		ptr: map[string]string{},
	}
}

func trimLocal(name string) string {
	n := strings.TrimSuffix(name, ".")
	return strings.TrimSuffix(n, ".local")
}

// serviceOf extrait le type de service d'un nom d'instance mDNS :
// "Chromecast-ab._googlecast._tcp.local." -> "_googlecast._tcp".
func serviceOf(instance string) string {
	n := strings.TrimSuffix(instance, ".")
	n = strings.TrimSuffix(n, ".local")
	i := strings.Index(n, "._")
	if i < 0 {
		return ""
	}
	return n[i+1:]
}

// instanceLabel renvoie le libellé lisible d'une instance ("Chromecast-ab").
func instanceLabel(instance string) string {
	n := strings.TrimSuffix(instance, ".")
	if i := strings.Index(n, "._"); i > 0 {
		return n[:i]
	}
	return trimLocal(n)
}

func (acc *mdnsAcc) parse(data []byte) {
	var p dnsmessage.Parser
	if _, err := p.Start(data); err != nil {
		return
	}
	_ = p.SkipAllQuestions()
	for _, section := range []func() (dnsmessage.ResourceHeader, error){p.AnswerHeader, p.AuthorityHeader, p.AdditionalHeader} {
		acc.parseSection(&p, section)
	}
}

func (acc *mdnsAcc) parseSection(p *dnsmessage.Parser, header func() (dnsmessage.ResourceHeader, error)) {
	for {
		h, err := header()
		if err != nil {
			return // fin de section
		}
		switch h.Type {
		case dnsmessage.TypeA:
			r, err := p.AResource()
			if err != nil {
				return
			}
			host := strings.ToLower(trimLocal(h.Name.String()))
			if host != "" {
				acc.a[host] = net.IP(r.A[:]).String()
			}
		case dnsmessage.TypeSRV:
			r, err := p.SRVResource()
			if err != nil {
				return
			}
			acc.srv[h.Name.String()] = strings.ToLower(trimLocal(r.Target.String()))
		case dnsmessage.TypeTXT:
			r, err := p.TXTResource()
			if err != nil {
				return
			}
			acc.txt[h.Name.String()] = parseTXT(r.TXT)
		case dnsmessage.TypePTR:
			r, err := p.PTRResource()
			if err != nil {
				return
			}
			inst := r.PTR.String()
			if svc := serviceOf(inst); svc != "" {
				acc.ptr[inst] = svc
			}
		default:
			if err := p.SkipAnswer(); err != nil {
				return
			}
		}
	}
}

// parseTXT transforme des enregistrements TXT ("md=Chromecast Ultra") en map.
func parseTXT(txt []string) map[string]string {
	kv := map[string]string{}
	for _, entry := range txt {
		if i := strings.IndexByte(entry, '='); i > 0 {
			kv[strings.ToLower(strings.TrimSpace(entry[:i]))] = strings.TrimSpace(entry[i+1:])
		}
	}
	return kv
}

// modelFromTXT lit le modèle depuis les clés TXT usuelles selon le fabricant.
func modelFromTXT(kv map[string]string) string {
	for _, k := range []string{"md", "model", "ty", "usb_mdl", "am", "product", "fv"} {
		if v := kv[k]; v != "" && k != "fv" {
			return v
		}
	}
	return ""
}

// nameFromTXT lit un nom convivial depuis les clés TXT usuelles.
func nameFromTXT(kv map[string]string) string {
	for _, k := range []string{"fn", "n", "nm", "name"} {
		if v := kv[k]; v != "" {
			return v
		}
	}
	return ""
}

// devices corrèle les enregistrements accumulés en appareils identifiés.
func (acc *mdnsAcc) devices() []Device {
	byIP := map[string]*Device{}
	get := func(ip string) *Device {
		if d, ok := byIP[ip]; ok {
			return d
		}
		d := &Device{IP: ip, Sources: []string{"mdns"}}
		byIP[ip] = d
		return d
	}

	// Instances de service : instance --SRV--> hôte --A--> IP.
	for inst, host := range acc.srv {
		ip := acc.a[host]
		if ip == "" {
			continue
		}
		d := get(ip)
		kv := acc.txt[inst]
		fillString(&d.Name, nameFromTXT(kv))
		fillString(&d.Name, instanceLabel(inst))
		fillString(&d.Model, modelFromTXT(kv))
		if svc := serviceOf(inst); svc != "" {
			d.Services = append(d.Services, svc)
			fillString(&d.DeviceType, classifyService(svc))
		}
		fillString(&d.Hostname, trimLocal(host))
	}

	// Enregistrements A restants (hôte sans instance de service corrélée).
	for host, ip := range acc.a {
		d := get(ip)
		fillString(&d.Hostname, trimLocal(host))
	}

	out := make([]Device, 0, len(byIP))
	for _, d := range byIP {
		d.Services = uniqueSorted(d.Services)
		out = append(out, *d)
	}
	return out
}

// MDNS rejoint le groupe multicast 224.0.0.251:5353 via l'interface active,
// émet les requêtes DNS-SD et écoute les réponses pendant la fenêtre timeout.
func MDNS(timeout time.Duration, srcIP string) []Device {
	laddr := &net.UDPAddr{IP: bindIP(srcIP), Port: 0}
	conn, err := net.ListenUDP("udp4", laddr)
	if err != nil {
		return nil
	}
	defer conn.Close()

	group := &net.UDPAddr{IP: net.IPv4(224, 0, 0, 251), Port: 5353}
	q, err := mdnsQuery()
	if err != nil {
		return nil
	}
	// Deux envois espacés : certains appareils ne répondent pas au premier.
	_, _ = conn.WriteToUDP(q, group)

	acc := newMDNSAcc()
	deadline := time.Now().Add(timeout)
	_ = conn.SetReadDeadline(deadline)
	resent := false
	buf := make([]byte, 9000)
	for {
		n, _, err := conn.ReadFromUDP(buf)
		if err != nil {
			break // deadline atteinte
		}
		acc.parse(buf[:n])
		if !resent && time.Now().Before(deadline.Add(-timeout/2)) {
			_, _ = conn.WriteToUDP(q, group)
			resent = true
		}
	}
	return acc.devices()
}

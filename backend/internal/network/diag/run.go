package diag

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/net/dns/dnsmessage"

	"netcompanion/internal/models"
)

// PortCheck teste la joignabilité d'un host:port en TCP.
func PortCheck(d Dialer, host string, port int) Check {
	c := Check{Name: fmt.Sprintf("Port %s:%d", host, port)}
	conn, err := d.Dial("tcp", net.JoinHostPort(host, strconv.Itoa(port)), 1500*time.Millisecond)
	if err != nil {
		c.Status, c.Detail = StatusFail, "fermé ou filtré"
		return c
	}
	_ = conn.Close()
	c.Status, c.Detail = StatusOK, "ouvert"
	return c
}

// hostProbePorts : ports courants sondés lors d'un diagnostic ciblé sur un hôte.
var hostProbePorts = []int{22, 23, 53, 80, 139, 443, 445, 631, 3389, 8009, 8080, 9100}

// portServiceNames : libellés lisibles pour les ports courants.
var portServiceNames = map[int]string{
	22: "SSH", 23: "Telnet", 53: "DNS", 80: "HTTP", 139: "NetBIOS",
	443: "HTTPS", 445: "SMB", 631: "IPP/impression", 3389: "RDP",
	8009: "Chromecast", 8080: "HTTP-alt", 9100: "impression",
}

// scanHostPorts teste en parallèle les ports fournis et renvoie ceux ouverts (triés).
func scanHostPorts(d Dialer, host string, ports []int) []int {
	type res struct {
		port int
		ok   bool
	}
	ch := make(chan res, len(ports))
	for _, p := range ports {
		go func(p int) {
			conn, err := d.Dial("tcp", net.JoinHostPort(host, strconv.Itoa(p)), dialTimeout)
			if err == nil {
				_ = conn.Close()
				ch <- res{p, true}
				return
			}
			ch <- res{p, false}
		}(p)
	}
	var open []int
	for range ports {
		if r := <-ch; r.ok {
			open = append(open, r.port)
		}
	}
	sort.Ints(open)
	return open
}

// portList formate une liste de ports avec leur service ("22 (SSH), 80 (HTTP)").
func portList(ports []int) string {
	parts := make([]string, 0, len(ports))
	for _, p := range ports {
		if name := portServiceNames[p]; name != "" {
			parts = append(parts, fmt.Sprintf("%d (%s)", p, name))
		} else {
			parts = append(parts, strconv.Itoa(p))
		}
	}
	return strings.Join(parts, ", ")
}

// RunHostSuite exécute un diagnostic ciblé sur UN hôte : joignabilité, latence,
// ports ouverts et nom d'hôte. Complète le bilan de connectivité générale.
func RunHostSuite(host string) []Check {
	d := NetDialer{}
	open := scanHostPorts(d, host, hostProbePorts)

	checks := make([]Check, 0, 4)
	reach := Check{Name: "Hôte joignable"}
	if len(open) > 0 {
		reach.Status = StatusOK
		reach.Detail = fmt.Sprintf("%s répond (port %d)", host, open[0])
	} else {
		reach.Status = StatusWarn
		reach.Detail = fmt.Sprintf("%s ne répond sur aucun port courant (filtré, pare-feu ou endormi)", host)
	}
	checks = append(checks, reach)

	if len(open) > 0 {
		lat := MeasureLatency(d, net.JoinHostPort(host, strconv.Itoa(open[0])), 4)
		lat.Name = "Latence vers l'hôte"
		checks = append(checks, lat)
	}

	ports := Check{Name: "Ports ouverts"}
	if len(open) > 0 {
		ports.Status, ports.Detail = StatusOK, portList(open)
	} else {
		ports.Status, ports.Detail = StatusWarn, "aucun port courant ouvert"
	}
	checks = append(checks, ports)

	if names, err := net.LookupAddr(host); err == nil && len(names) > 0 {
		checks = append(checks, Check{
			Name: "Nom d'hôte", Status: StatusOK,
			Detail: strings.TrimSuffix(names[0], "."),
		})
	}
	return checks
}

// RunSuite exécute un diagnostic de connectivité COMPLET, du poste jusqu'à
// Internet : interface/MTU, serveurs DNS, passerelle, DNS, accès + latence
// Internet, perte de paquets ICMP, ports sortants, IPv6, portail captif, IP
// publique et débit descendant. Les contrôles indépendants tournent en parallèle.
func RunSuite(ifi models.InterfaceInfo, gw string) []Check {
	d := NetDialer{}
	r := NetResolver{}
	jobs := []func() Check{
		func() Check { return interfaceCheck(ifi) },
		func() Check { return dnsServersCheck() },
		func() Check { return CheckGateway(d, gw) },
		func() Check { return gatewayLatency(d, gw) },
		func() Check { return CheckDNS(r, "example.com") },
		func() Check { return CheckInternet(d, []string{"1.1.1.1:443", "8.8.8.8:443"}) },
		func() Check { return internetLatency(d) },
		func() Check { return packetLossCheck("1.1.1.1") },
		func() Check { return egressPortsCheck() },
		func() Check { return ipv6Check(d) },
		func() Check { return captivePortalCheck() },
		func() Check { return publicIPCheck() },
		func() Check { return throughputCheck() },
	}
	out := make([]Check, len(jobs))
	var wg sync.WaitGroup
	for i, fn := range jobs {
		wg.Add(1)
		go func(i int, fn func() Check) {
			defer wg.Done()
			out[i] = fn()
		}(i, fn)
	}
	wg.Wait()
	return out
}

// internetLatency mesure le RTT/jitter/perte vers une cible publique.
func internetLatency(d Dialer) Check {
	c := MeasureLatency(d, "1.1.1.1:443", 4)
	c.Name = "Latence Internet"
	return c
}

// interfaceCheck vérifie qu'on dispose d'une adresse IP exploitable + le MTU.
func interfaceCheck(ifi models.InterfaceInfo) Check {
	c := Check{Name: "Interface locale"}
	if ifi.IPv4 == "" {
		c.Status, c.Detail = StatusFail, "aucune adresse IPv4 (pas de bail DHCP ?)"
		return c
	}
	if ip := net.ParseIP(ifi.IPv4); ip != nil && ip.IsLinkLocalUnicast() {
		c.Status, c.Detail = StatusWarn, fmt.Sprintf("APIPA %s (DHCP injoignable)", ifi.IPv4)
		return c
	}
	detail := fmt.Sprintf("%s sur %s (%s)", ifi.IPv4, ifi.Name, ifi.CIDR)
	if iface, err := net.InterfaceByName(ifi.Name); err == nil && iface.MTU > 0 {
		detail += fmt.Sprintf(", MTU %d", iface.MTU)
	}
	c.Status, c.Detail = StatusOK, detail
	return c
}

// dnsServersCheck détecte les serveurs DNS configurés et vérifie qu'ils répondent.
func dnsServersCheck() Check {
	c := Check{Name: "Serveurs DNS"}
	servers := configuredDNS()
	if len(servers) == 0 {
		c.Status, c.Detail = StatusWarn, "non détectés"
		return c
	}
	var parts []string
	anyOK := false
	for i, s := range servers {
		if i >= 3 {
			break
		}
		if rtt, ok := queryDNSServer(s, "example.com"); ok {
			parts = append(parts, fmt.Sprintf("%s (%d ms)", s, rtt.Milliseconds()))
			anyOK = true
		} else {
			parts = append(parts, s+" (pas de réponse)")
		}
	}
	c.Detail = strings.Join(parts, ", ")
	if anyOK {
		c.Status = StatusOK
	} else {
		c.Status = StatusFail
	}
	return c
}

// configuredDNS renvoie les serveurs DNS IPv4 configurés sur le système.
func configuredDNS() []string {
	if runtime.GOOS == "windows" {
		out, _ := exec.Command("powershell", "-NoProfile", "-Command",
			"(Get-DnsClientServerAddress -AddressFamily IPv4).ServerAddresses").Output()
		return uniqIPv4(string(out))
	}
	data, _ := os.ReadFile("/etc/resolv.conf")
	var out []string
	seen := map[string]bool{}
	for _, line := range strings.Split(string(data), "\n") {
		f := strings.Fields(line)
		if len(f) >= 2 && f[0] == "nameserver" && !seen[f[1]] {
			seen[f[1]] = true
			out = append(out, f[1])
		}
	}
	return out
}

func uniqIPv4(s string) []string {
	var out []string
	seen := map[string]bool{}
	for _, m := range reIPv4.FindAllString(s, -1) {
		if !seen[m] {
			seen[m] = true
			out = append(out, m)
		}
	}
	return out
}

// queryDNSServer envoie une requête A à un serveur DNS précis et mesure le RTT.
func queryDNSServer(server, name string) (time.Duration, bool) {
	conn, err := net.DialTimeout("udp", net.JoinHostPort(server, "53"), 1500*time.Millisecond)
	if err != nil {
		return 0, false
	}
	defer conn.Close()
	if !strings.HasSuffix(name, ".") {
		name += "."
	}
	b := dnsmessage.NewBuilder(nil, dnsmessage.Header{RecursionDesired: true, ID: 0x4e43})
	if err := b.StartQuestions(); err != nil {
		return 0, false
	}
	qn, err := dnsmessage.NewName(name)
	if err != nil {
		return 0, false
	}
	_ = b.Question(dnsmessage.Question{Name: qn, Type: dnsmessage.TypeA, Class: dnsmessage.ClassINET})
	q, err := b.Finish()
	if err != nil {
		return 0, false
	}
	_ = conn.SetDeadline(time.Now().Add(1500 * time.Millisecond))
	start := time.Now()
	if _, err := conn.Write(q); err != nil {
		return 0, false
	}
	resp := make([]byte, 512)
	n, err := conn.Read(resp)
	if err != nil || n == 0 {
		return 0, false
	}
	var p dnsmessage.Parser
	if _, err := p.Start(resp[:n]); err != nil {
		return 0, false
	}
	return time.Since(start), true
}

// packetLossCheck mesure la perte de paquets ICMP (via la commande ping de l'OS).
func packetLossCheck(target string) Check {
	c := Check{Name: "Perte de paquets"}
	var out []byte
	if runtime.GOOS == "windows" {
		out, _ = exec.Command("ping", "-n", "5", "-w", "1000", target).Output()
	} else {
		out, _ = exec.Command("ping", "-c", "5", "-W", "1", target).Output()
	}
	m := regexp.MustCompile(`(\d+)\s*%`).FindStringSubmatch(string(out))
	if m == nil {
		c.Status, c.Detail = StatusFail, target+" injoignable (ICMP bloqué ?)"
		return c
	}
	loss, _ := strconv.Atoi(m[1])
	c.Detail = fmt.Sprintf("%d%% de perte vers %s (5 paquets ICMP)", loss, target)
	switch {
	case loss == 0:
		c.Status = StatusOK
	case loss < 50:
		c.Status = StatusWarn
	default:
		c.Status = StatusFail
	}
	return c
}

// egressPortsCheck détecte un filtrage de sortie sur les ports courants.
func egressPortsCheck() Check {
	c := Check{Name: "Ports sortants"}
	ports := []struct{ p, label string }{{"53", "DNS/53"}, {"80", "HTTP/80"}, {"443", "HTTPS/443"}}
	var okp, blocked []string
	for _, pp := range ports {
		conn, err := net.DialTimeout("tcp", "1.1.1.1:"+pp.p, 1500*time.Millisecond)
		if err == nil {
			_ = conn.Close()
			okp = append(okp, pp.label)
		} else {
			blocked = append(blocked, pp.label)
		}
	}
	if len(blocked) == 0 {
		c.Status, c.Detail = StatusOK, "53, 80, 443 ouverts en sortie"
		return c
	}
	c.Status = StatusWarn
	c.Detail = "bloqués : " + strings.Join(blocked, ", ")
	if len(okp) > 0 {
		c.Detail += " (ouverts : " + strings.Join(okp, ", ") + ")"
	}
	return c
}

// ipv6Check vérifie la présence d'une IPv6 globale et la sortie IPv6.
func ipv6Check(d Dialer) Check {
	c := Check{Name: "IPv6"}
	hasGlobal := false
	if addrs, err := net.InterfaceAddrs(); err == nil {
		for _, a := range addrs {
			if ipn, ok := a.(*net.IPNet); ok {
				ip := ipn.IP
				if ip.To4() == nil && ip.IsGlobalUnicast() && !ip.IsPrivate() {
					hasGlobal = true
					break
				}
			}
		}
	}
	egress := false
	if conn, err := d.Dial("tcp6", "[2606:4700:4700::1111]:443", dialTimeout); err == nil {
		_ = conn.Close()
		egress = true
	}
	switch {
	case hasGlobal && egress:
		c.Status, c.Detail = StatusOK, "actif (adresse globale + sortie Internet)"
	case hasGlobal && !egress:
		c.Status, c.Detail = StatusWarn, "adresse globale mais pas de sortie IPv6"
	case !hasGlobal && egress:
		c.Status, c.Detail = StatusOK, "sortie IPv6 disponible"
	default:
		c.Status, c.Detail = StatusOK, "non configuré (réseau IPv4)"
	}
	return c
}

// throughputCheck estime le débit descendant (best-effort, ~2 Mo).
func throughputCheck() Check {
	c := Check{Name: "Débit descendant (estimation)"}
	client := &http.Client{Timeout: 6 * time.Second}
	const size = 2_000_000
	start := time.Now()
	resp, err := client.Get(fmt.Sprintf("https://speed.cloudflare.com/__down?bytes=%d", size))
	if err != nil {
		c.Status, c.Detail = StatusWarn, "indisponible"
		return c
	}
	defer resp.Body.Close()
	n, _ := io.Copy(io.Discard, resp.Body)
	dur := time.Since(start).Seconds()
	if n <= 0 || dur <= 0 {
		c.Status, c.Detail = StatusWarn, "indisponible"
		return c
	}
	mbps := float64(n) * 8 / dur / 1e6
	c.Status = StatusOK
	c.Detail = fmt.Sprintf("≈ %.0f Mbps (%.1f Mo en %.1fs)", mbps, float64(n)/1e6, dur)
	return c
}

// gatewayLatency mesure le RTT vers la passerelle (santé du LAN).
func gatewayLatency(d Dialer, gw string) Check {
	c := Check{Name: "Latence passerelle"}
	if gw == "" {
		c.Status, c.Detail = StatusWarn, "passerelle inconnue"
		return c
	}
	for _, p := range []string{"443", "80", "53"} {
		target := net.JoinHostPort(gw, p)
		conn, err := d.Dial("tcp", target, dialTimeout)
		if err != nil {
			continue
		}
		_ = conn.Close()
		m := MeasureLatency(d, target, 4)
		m.Name = "Latence passerelle"
		return m
	}
	c.Status, c.Detail = StatusWarn, gw+" ne répond pas (latence indisponible)"
	return c
}

// captivePortalCheck détecte un portail captif / proxy interceptant le trafic.
func captivePortalCheck() Check {
	c := Check{Name: "Portail captif"}
	client := &http.Client{
		Timeout:       2 * time.Second,
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}
	resp, err := client.Get("http://connectivitycheck.gstatic.com/generate_204")
	if err != nil {
		c.Status, c.Detail = StatusWarn, "vérification impossible (hors ligne ?)"
		return c
	}
	defer resp.Body.Close()
	if resp.StatusCode == 204 {
		c.Status, c.Detail = StatusOK, "aucun (accès Internet direct)"
		return c
	}
	c.Status = StatusWarn
	c.Detail = fmt.Sprintf("portail captif ou proxy probable (HTTP %d)", resp.StatusCode)
	return c
}

// publicIPCheck récupère l'adresse IP publique (WAN).
func publicIPCheck() Check {
	c := Check{Name: "IP publique (WAN)"}
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get("https://api.ipify.org")
	if err != nil {
		c.Status, c.Detail = StatusWarn, "indisponible (hors ligne ou bloqué)"
		return c
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(io.LimitReader(resp.Body, 64))
	ip := strings.TrimSpace(string(b))
	if ip == "" {
		c.Status, c.Detail = StatusWarn, "indisponible"
		return c
	}
	c.Status, c.Detail = StatusOK, ip
	return c
}

// Traceroute exécute le traceroute de l'OS et parse les sauts.
func Traceroute(target string) []Hop {
	if runtime.GOOS == "windows" {
		out, _ := exec.Command("tracert", "-d", "-h", "15", "-w", "800", target).Output()
		return parseTracerouteWindows(string(out))
	}
	out, _ := exec.Command("traceroute", "-n", "-m", "15", "-w", "1", target).Output()
	return parseTracerouteUnix(string(out))
}

package diag

import (
	"fmt"
	"net"
	"time"
)

// Dialer abstrait l'établissement d'une connexion TCP (injectable en test).
type Dialer interface {
	Dial(network, addr string, timeout time.Duration) (net.Conn, error)
}

// Resolver abstrait la résolution DNS (injectable en test).
type Resolver interface {
	LookupHost(host string) ([]string, error)
}

// NetDialer est l'implémentation stdlib par défaut.
type NetDialer struct{}

func (NetDialer) Dial(network, addr string, timeout time.Duration) (net.Conn, error) {
	return net.DialTimeout(network, addr, timeout)
}

// NetResolver est l'implémentation stdlib par défaut.
type NetResolver struct{}

func (NetResolver) LookupHost(host string) ([]string, error) { return net.LookupHost(host) }

const dialTimeout = 800 * time.Millisecond

// CheckGateway teste la joignabilité de la passerelle en TCP (ports usuels).
func CheckGateway(d Dialer, gw string) Check {
	c := Check{Name: "Passerelle joignable"}
	if gw == "" {
		c.Status, c.Detail = StatusWarn, "passerelle inconnue"
		return c
	}
	for _, port := range []string{"53", "80", "443"} {
		conn, err := d.Dial("tcp", net.JoinHostPort(gw, port), dialTimeout)
		if err == nil {
			_ = conn.Close()
			c.Status, c.Detail = StatusOK, fmt.Sprintf("%s joignable (port %s)", gw, port)
			return c
		}
	}
	c.Status, c.Detail = StatusFail, fmt.Sprintf("%s injoignable en TCP (53/80/443)", gw)
	return c
}

// CheckDNS vérifie la résolution d'un nom et mesure sa durée. Affiche l'IPv4 en
// priorité (plus familière) et signale la présence d'une IPv6 ainsi qu'un DNS lent.
func CheckDNS(r Resolver, name string) Check {
	c := Check{Name: "Résolution DNS"}
	start := time.Now()
	addrs, err := r.LookupHost(name)
	ms := time.Since(start).Milliseconds()
	if err != nil || len(addrs) == 0 {
		c.Status, c.Detail = StatusFail, fmt.Sprintf("échec de résolution de %s", name)
		return c
	}
	var ipv4, ipv6 string
	for _, a := range addrs {
		ip := net.ParseIP(a)
		if ip == nil {
			continue
		}
		if ip.To4() != nil {
			if ipv4 == "" {
				ipv4 = a
			}
		} else if ipv6 == "" {
			ipv6 = a
		}
	}
	shown := ipv4
	if shown == "" {
		shown = addrs[0]
	}
	detail := fmt.Sprintf("%s → %s", name, shown)
	if ipv4 != "" && ipv6 != "" {
		detail += " (IPv4 + IPv6)"
	} else if ipv6 != "" {
		detail += " (IPv6)"
	}
	detail += fmt.Sprintf(" en %d ms", ms)
	c.Status = StatusOK
	if ms > 400 {
		c.Status = StatusWarn
		detail += " — DNS lent"
	}
	c.Detail = detail
	return c
}

// CheckInternet teste l'accès Internet via une connexion TCP vers des cibles publiques.
func CheckInternet(d Dialer, targets []string) Check {
	c := Check{Name: "Accès Internet"}
	for _, t := range targets {
		conn, err := d.Dial("tcp", t, dialTimeout)
		if err == nil {
			_ = conn.Close()
			c.Status, c.Detail = StatusOK, fmt.Sprintf("connecté à %s", t)
			return c
		}
	}
	c.Status, c.Detail = StatusFail, "aucune cible publique joignable (hors ligne ?)"
	return c
}

// MeasureLatency mesure le RTT (via connexions TCP répétées) et le jitter.
func MeasureLatency(d Dialer, target string, n int) Check {
	c := Check{Name: "Latence"}
	if n < 1 {
		n = 3
	}
	samples := make([]time.Duration, 0, n)
	for i := 0; i < n; i++ {
		start := time.Now()
		conn, err := d.Dial("tcp", target, dialTimeout)
		if err != nil {
			samples = append(samples, 0) // 0 = perte
			continue
		}
		rtt := time.Since(start)
		if rtt <= 0 {
			rtt = time.Nanosecond // un dial réussi n'est jamais une perte
		}
		samples = append(samples, rtt)
		_ = conn.Close()
	}
	avg, jitter, loss := latencyStats(samples)
	if loss == 100 {
		c.Status, c.Detail = StatusFail, fmt.Sprintf("%s injoignable", target)
		return c
	}
	c.Status = StatusOK
	if loss > 0 {
		c.Status = StatusWarn
	}
	c.Detail = fmt.Sprintf("%s : %d ms (jitter %d ms, perte %d%%)",
		target, avg.Milliseconds(), jitter.Milliseconds(), loss)
	return c
}

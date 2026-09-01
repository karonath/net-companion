package radar

import (
	"errors"
	"net"
	"strconv"
	"syscall"
	"time"

	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
)

// Prober teste si un hôte est vivant.
type Prober interface {
	Probe(ip string) bool
}

// TCPProber teste une liste de ports TCP (aucun privilège requis).
type TCPProber struct {
	Ports   []int
	Timeout time.Duration
}

// Probe renvoie true si un port accepte la connexion, ou la refuse
// explicitement (« connection refused » = hôte présent, port fermé).
func (p TCPProber) Probe(ip string) bool {
	for _, port := range p.Ports {
		addr := net.JoinHostPort(ip, strconv.Itoa(port))
		conn, err := net.DialTimeout("tcp", addr, p.Timeout)
		if err == nil {
			_ = conn.Close()
			return true
		}
		if errors.Is(err, syscall.ECONNREFUSED) {
			return true
		}
	}
	return false
}

// ICMPProber envoie un echo ICMP (nécessite des privilèges sous Windows).
type ICMPProber struct {
	Timeout time.Duration
}

func (p ICMPProber) Probe(ip string) bool {
	c, err := icmp.ListenPacket("ip4:icmp", "0.0.0.0")
	if err != nil {
		return false
	}
	defer c.Close()

	msg := icmp.Message{
		Type: ipv4.ICMPTypeEcho, Code: 0,
		Body: &icmp.Echo{ID: 1, Seq: 1, Data: []byte("nc")},
	}
	b, err := msg.Marshal(nil)
	if err != nil {
		return false
	}
	if _, err := c.WriteTo(b, &net.IPAddr{IP: net.ParseIP(ip)}); err != nil {
		return false
	}
	_ = c.SetReadDeadline(time.Now().Add(p.Timeout))
	reply := make([]byte, 64)
	_, _, err = c.ReadFrom(reply)
	return err == nil
}

// icmpAvailable teste si l'on peut ouvrir un socket ICMP (droits suffisants).
func icmpAvailable() bool {
	c, err := icmp.ListenPacket("ip4:icmp", "0.0.0.0")
	if err != nil {
		return false
	}
	_ = c.Close()
	return true
}

// NewProber renvoie l'ICMP si les droits le permettent, sinon un prober TCP.
func NewProber() Prober {
	if icmpAvailable() {
		return ICMPProber{Timeout: 500 * time.Millisecond}
	}
	return TCPProber{
		Ports:   []int{80, 443, 22, 445, 135, 3389},
		Timeout: 300 * time.Millisecond,
	}
}

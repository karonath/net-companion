package diag

import (
	"fmt"
	"net"
	"os/exec"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"
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

// RunSuite exécute la batterie standard de diagnostics.
func RunSuite(gw string) []Check {
	d := NetDialer{}
	r := NetResolver{}
	return []Check{
		CheckGateway(d, gw),
		CheckDNS(r, "example.com"),
		CheckInternet(d, []string{"1.1.1.1:443", "8.8.8.8:443"}),
		MeasureLatency(d, "1.1.1.1:443", 4),
	}
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

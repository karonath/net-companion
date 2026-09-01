package diag

import (
	"fmt"
	"net"
	"os/exec"
	"runtime"
	"strconv"
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

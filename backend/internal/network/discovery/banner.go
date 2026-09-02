package discovery

import (
	"bufio"
	"fmt"
	"net"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

// bannerPorts : ports TCP sondés pour typer un appareil muet en multicast.
var bannerPorts = []int{22, 80, 443, 8080, 8000, 139, 445, 515, 631, 9100, 23, 53, 21, 3389, 5000}

var (
	reHTTPServer = regexp.MustCompile(`(?im)^Server:\s*(.+?)\s*$`)
	reHTTPTitle  = regexp.MustCompile(`(?is)<title>(.*?)</title>`)
	reSSHVendor  = regexp.MustCompile(`SSH-\d+\.\d+-(.+)`)
)

// parseHTTPHead extrait l'en-tête Server et le <title> d'une réponse HTTP.
func parseHTTPHead(resp string) (server, title string) {
	if m := reHTTPServer.FindStringSubmatch(resp); len(m) == 2 {
		server = strings.TrimSpace(m[1])
	}
	if m := reHTTPTitle.FindStringSubmatch(resp); len(m) == 2 {
		title = strings.TrimSpace(m[1])
	}
	return server, title
}

// parseSSHBanner extrait la chaîne logicielle d'une bannière SSH.
func parseSSHBanner(line string) string {
	if m := reSSHVendor.FindStringSubmatch(strings.TrimSpace(line)); len(m) == 2 {
		return strings.TrimSpace(m[1])
	}
	return ""
}

// grabBanner lit ce que le service émet spontanément (SSH, FTP, telnet…).
func grabBanner(conn net.Conn, timeout time.Duration) string {
	_ = conn.SetReadDeadline(time.Now().Add(timeout))
	line, _ := bufio.NewReader(conn).ReadString('\n')
	return strings.TrimSpace(line)
}

// grabHTTP envoie une requête HEAD et lit l'en-tête/titre de la réponse.
func grabHTTP(conn net.Conn, host string, timeout time.Duration) string {
	_ = conn.SetWriteDeadline(time.Now().Add(timeout))
	req := "GET / HTTP/1.1\r\nHost: " + host + "\r\nConnection: close\r\nUser-Agent: Net-Companion\r\n\r\n"
	if _, err := conn.Write([]byte(req)); err != nil {
		return ""
	}
	_ = conn.SetReadDeadline(time.Now().Add(timeout))
	buf := make([]byte, 8192)
	n, _ := conn.Read(buf)
	return string(buf[:n])
}

// probeHost sonde les ports TCP d'un hôte et renvoie un Device typé si au moins
// un port répond.
func probeHost(ip string, timeout time.Duration) *Device {
	d := &Device{IP: ip, Sources: []string{"banner"}}
	var mu sync.Mutex
	var wg sync.WaitGroup
	found := false

	for _, port := range bannerPorts {
		wg.Add(1)
		go func(port int) {
			defer wg.Done()
			conn, err := net.DialTimeout("tcp", net.JoinHostPort(ip, strconv.Itoa(port)), timeout)
			if err != nil {
				return
			}
			defer conn.Close()

			mu.Lock()
			found = true
			d.Services = append(d.Services, fmt.Sprintf("tcp/%d", port))
			if t := classifyService(fmt.Sprintf("tcp/%d", port)); t != "" {
				fillString(&d.DeviceType, t)
			}
			mu.Unlock()

			switch port {
			case 22:
				if v := parseSSHBanner(grabBanner(conn, timeout)); v != "" {
					mu.Lock()
					fillString(&d.Model, v)
					fillString(&d.DeviceType, "ordinateur")
					mu.Unlock()
				}
			case 80, 8080, 8000, 5000:
				server, title := parseHTTPHead(grabHTTP(conn, ip, timeout))
				mu.Lock()
				fillString(&d.Manufacturer, server)
				fillString(&d.Name, title)
				mu.Unlock()
			case 21, 23:
				if b := grabBanner(conn, timeout); b != "" {
					mu.Lock()
					fillString(&d.Model, b)
					mu.Unlock()
				}
			}
		}(port)
	}
	wg.Wait()
	if !found {
		return nil
	}
	d.Services = uniqueSorted(d.Services)
	return d
}

// Banners sonde en parallèle les bannières de services TCP de chaque hôte.
func Banners(timeout time.Duration, hosts []string) []Device {
	if len(hosts) == 0 {
		return nil
	}
	// Timeout par port court pour rester réactif malgré de nombreux hôtes.
	perPort := timeout
	if perPort > 1200*time.Millisecond {
		perPort = 1200 * time.Millisecond
	}

	sem := make(chan struct{}, 64) // borne le nombre d'hôtes sondés en parallèle
	var mu sync.Mutex
	var wg sync.WaitGroup
	var out []Device
	for _, ip := range hosts {
		if net.ParseIP(ip) == nil {
			continue
		}
		wg.Add(1)
		go func(ip string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			if d := probeHost(ip, perPort); d != nil {
				mu.Lock()
				out = append(out, *d)
				mu.Unlock()
			}
		}(ip)
	}
	wg.Wait()
	return out
}

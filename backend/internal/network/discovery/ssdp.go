package discovery

import (
	"io"
	"net"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"
)

// parseSSDPHeaders extrait les en-têtes utiles d'une réponse SSDP :
// SERVER (chaîne produit) et LOCATION (URL de la fiche descriptive UPnP).
func parseSSDPHeaders(raw string) (server, location string) {
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimRight(line, "\r")
		if i := strings.IndexByte(line, ':'); i > 0 {
			key := strings.ToUpper(strings.TrimSpace(line[:i]))
			val := strings.TrimSpace(line[i+1:])
			switch key {
			case "SERVER":
				server = val
			case "LOCATION":
				location = val
			}
		}
	}
	return server, location
}

// parseSSDP conserve l'ancienne signature (modèle = SERVER) pour compat.
func parseSSDP(raw string) string {
	s, _ := parseSSDPHeaders(raw)
	return s
}

var upnpTags = map[string]*regexp.Regexp{
	"friendlyName": regexp.MustCompile(`(?is)<friendlyName>(.*?)</friendlyName>`),
	"manufacturer": regexp.MustCompile(`(?is)<manufacturer>(.*?)</manufacturer>`),
	"modelName":    regexp.MustCompile(`(?is)<modelName>(.*?)</modelName>`),
	"modelNumber":  regexp.MustCompile(`(?is)<modelNumber>(.*?)</modelNumber>`),
	"deviceType":   regexp.MustCompile(`(?is)<deviceType>(.*?)</deviceType>`),
}

// parseUPnPXML extrait les champs d'identité d'une fiche descriptive UPnP.
func parseUPnPXML(xml string) map[string]string {
	out := map[string]string{}
	for key, re := range upnpTags {
		if m := re.FindStringSubmatch(xml); len(m) == 2 {
			if v := strings.TrimSpace(m[1]); v != "" {
				out[key] = v
			}
		}
	}
	return out
}

// deviceFromUPnP construit un Device à partir des champs XML extraits.
func deviceFromUPnP(ip, server string, fields map[string]string) Device {
	d := Device{IP: ip, Sources: []string{"ssdp"}}
	d.Name = fields["friendlyName"]
	d.Manufacturer = fields["manufacturer"]
	model := fields["modelName"]
	if n := fields["modelNumber"]; n != "" && !strings.Contains(model, n) {
		model = strings.TrimSpace(model + " " + n)
	}
	d.Model = model
	if d.Model == "" {
		d.Model = server // repli sur l'en-tête SERVER
	}
	d.DeviceType = classifyService(fields["deviceType"])
	return d
}

// fetchUPnP récupère et parse la fiche descriptive UPnP (timeout court).
func fetchUPnP(location string, timeout time.Duration) map[string]string {
	if location == "" {
		return nil
	}
	client := &http.Client{Timeout: timeout}
	resp, err := client.Get(location)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
	if err != nil {
		return nil
	}
	return parseUPnPXML(string(body))
}

// SSDP envoie un M-SEARCH, collecte les réponses et enrichit chaque appareil en
// récupérant sa fiche descriptive UPnP (friendlyName, manufacturer, modelName…).
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

	// Fenêtre d'écoute réservant du temps pour les fetch XML ensuite.
	listen := timeout * 2 / 3
	if listen < 700*time.Millisecond {
		listen = timeout
	}
	_ = conn.SetReadDeadline(time.Now().Add(listen))

	type resp struct{ server, location string }
	seen := map[string]resp{}
	buf := make([]byte, 4096)
	for {
		n, src, err := conn.ReadFromUDP(buf)
		if err != nil {
			break
		}
		ip := src.IP.String()
		server, location := parseSSDPHeaders(string(buf[:n]))
		if r, ok := seen[ip]; ok {
			// complète les champs manquants sur réponses multiples
			if r.server == "" {
				r.server = server
			}
			if r.location == "" {
				r.location = location
			}
			seen[ip] = r
			continue
		}
		seen[ip] = resp{server: server, location: location}
	}

	// Enrichissement XML en parallèle.
	var wg sync.WaitGroup
	var mu sync.Mutex
	var out []Device
	for ip, r := range seen {
		wg.Add(1)
		go func(ip string, r resp) {
			defer wg.Done()
			fields := fetchUPnP(r.location, 1500*time.Millisecond)
			d := deviceFromUPnP(ip, r.server, fields)
			mu.Lock()
			out = append(out, d)
			mu.Unlock()
		}(ip, r)
	}
	wg.Wait()
	return out
}

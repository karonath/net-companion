//go:build windows

package arp

import (
	"net"
	"sync"
	"syscall"
	"time"
	"unsafe"
)

// scanDeadline borne le temps total du balayage : les appareils présents
// répondent en quelques ms ; seules les IP inexistantes bloquent jusqu'au
// timeout ARP de Windows. On abandonne ces attentes mortes passé ce délai.
const scanDeadline = 2500 * time.Millisecond

// procSendARP : API Win32 SendARP (iphlpapi.dll). Résout une IP en MAC par une
// requête ARP, SANS privilège administrateur ni Npcap — idéal pour la clé USB.
var procSendARP = syscall.NewLazyDLL("iphlpapi.dll").NewProc("SendARP")

// ipToULong convertit une IPv4 en ULONG réseau (octet 0 = poids faible).
func ipToULong(ip net.IP) uint32 {
	ip = ip.To4()
	if ip == nil {
		return 0
	}
	return uint32(ip[0]) | uint32(ip[1])<<8 | uint32(ip[2])<<16 | uint32(ip[3])<<24
}

// sendARP interroge une IP et renvoie sa MAC si l'appareil répond à l'ARP.
func sendARP(ip net.IP) (string, bool) {
	var mac [2]uint32 // 8 octets, la MAC occupe les 6 premiers
	length := uint32(8)
	ret, _, _ := procSendARP.Call(
		uintptr(ipToULong(ip)),
		0, // srcIP 0 = laisser la pile choisir
		uintptr(unsafe.Pointer(&mac[0])),
		uintptr(unsafe.Pointer(&length)),
	)
	if ret != 0 || length < 6 {
		return "", false
	}
	b := (*[6]byte)(unsafe.Pointer(&mac[0]))
	return net.HardwareAddr(b[:6]).String(), true
}

// ScanIPs effectue un balayage ARP actif des IP fournies (en parallèle) et
// renvoie celles qui répondent, avec leur MAC. Capte les appareils réveillés
// sans port ouvert (objets, imprimantes, caméras) que le balayage TCP rate.
func ScanIPs(ips []string) []Neighbor {
	const workers = 256
	sem := make(chan struct{}, workers)
	results := make(chan Neighbor, len(ips))
	var wg sync.WaitGroup
	for _, s := range ips {
		ip := net.ParseIP(s)
		if ip == nil || ip.To4() == nil {
			continue
		}
		wg.Add(1)
		go func(ip net.IP) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			if mac, ok := sendARP(ip); ok {
				results <- Neighbor{IP: ip.String(), MAC: normalizeMAC(mac)}
			}
		}(ip)
	}

	done := make(chan struct{})
	go func() { wg.Wait(); close(done) }()

	// Collecte jusqu'à ce que tout réponde OU jusqu'au deadline. Les goroutines
	// restantes (IP mortes) finiront en tâche de fond ; le canal bufferisé évite
	// tout blocage et toute course sur le slice retourné.
	deadline := time.After(scanDeadline)
	var out []Neighbor
	drain := func() []Neighbor {
		for {
			select {
			case n := <-results:
				out = append(out, n)
			default:
				return out
			}
		}
	}
	for {
		select {
		case n := <-results:
			out = append(out, n)
		case <-done:
			return drain()
		case <-deadline:
			return drain()
		}
	}
}

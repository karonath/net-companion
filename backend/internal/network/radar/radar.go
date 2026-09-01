// Package radar énumère et sonde les hôtes d'un sous-réseau (sweep concurrent).
package radar

import (
	"net"
	"sync"
)

// HostsInCIDR renvoie les IPv4 hôtes utilisables du CIDR, plafonné à max.
func HostsInCIDR(cidr string, max int) ([]string, error) {
	_, ipnet, err := net.ParseCIDR(cidr)
	if err != nil {
		return nil, err
	}
	var out []string
	for ip := cloneIP(ipnet.IP.Mask(ipnet.Mask)); ipnet.Contains(ip); inc(ip) {
		v4 := ip.To4()
		if v4 == nil {
			break
		}
		if isNetworkOrBroadcast(v4, ipnet) {
			continue
		}
		out = append(out, v4.String())
		if len(out) >= max {
			break
		}
	}
	return out, nil
}

func isNetworkOrBroadcast(ip net.IP, ipnet *net.IPNet) bool {
	network := ip.Mask(ipnet.Mask)
	if ip.Equal(network) {
		return true
	}
	broadcast := make(net.IP, len(network))
	for i := range network {
		broadcast[i] = network[i] | ^ipnet.Mask[i]
	}
	return ip.Equal(broadcast)
}

// Sweep sonde tous les hosts avec p via un pool de workers et renvoie les vivants.
func Sweep(hosts []string, p Prober, workers int) []string {
	if workers < 1 {
		workers = 1
	}
	jobs := make(chan string)
	results := make(chan string)
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for ip := range jobs {
				if p.Probe(ip) {
					results <- ip
				}
			}
		}()
	}
	go func() {
		for _, h := range hosts {
			jobs <- h
		}
		close(jobs)
	}()
	go func() {
		wg.Wait()
		close(results)
	}()

	var alive []string
	for ip := range results {
		alive = append(alive, ip)
	}
	return alive
}

func cloneIP(ip net.IP) net.IP {
	c := make(net.IP, len(ip))
	copy(c, ip)
	return c
}

func inc(ip net.IP) {
	for i := len(ip) - 1; i >= 0; i-- {
		ip[i]++
		if ip[i] > 0 {
			break
		}
	}
}

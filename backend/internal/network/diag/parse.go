package diag

import (
	"regexp"
	"strconv"
	"strings"
	"time"
)

var (
	reHopNum = regexp.MustCompile(`^\s*(\d+)\s`)
	reIPv4   = regexp.MustCompile(`\b(\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3})\b`)
	reMS     = regexp.MustCompile(`(\d+(?:[.,]\d+)?)\s*ms`)
)

// latencyStats calcule la moyenne, le jitter et le taux de perte (%) d'une
// série de RTT (une valeur nulle = perte).
func latencyStats(samples []time.Duration) (avg, jitter time.Duration, lossPct int) {
	var ok []time.Duration
	lost := 0
	for _, s := range samples {
		if s <= 0 {
			lost++
			continue
		}
		ok = append(ok, s)
	}
	if len(samples) > 0 {
		lossPct = lost * 100 / len(samples)
	}
	if len(ok) == 0 {
		return 0, 0, lossPct
	}
	var sum time.Duration
	for _, d := range ok {
		sum += d
	}
	avg = sum / time.Duration(len(ok))

	if len(ok) > 1 {
		var jsum time.Duration
		for i := 1; i < len(ok); i++ {
			d := ok[i] - ok[i-1]
			if d < 0 {
				d = -d
			}
			jsum += d
		}
		jitter = jsum / time.Duration(len(ok)-1)
	}
	return avg, jitter, lossPct
}

// parseHops est la logique commune : chaque ligne débutant par un numéro de
// saut donne un Hop (host = 1re IPv4 trouvée, sinon "*").
func parseHops(raw string) []Hop {
	var hops []Hop
	for _, line := range strings.Split(raw, "\n") {
		m := reHopNum.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		num, _ := strconv.Atoi(m[1])
		host := "*"
		if ip := reIPv4.FindString(line); ip != "" {
			host = ip
		}
		ms := ""
		if mm := reMS.FindStringSubmatch(line); mm != nil {
			ms = mm[1]
		}
		hops = append(hops, Hop{Num: num, Host: host, MS: ms})
	}
	return hops
}

func parseTracerouteWindows(raw string) []Hop { return parseHops(raw) }
func parseTracerouteUnix(raw string) []Hop    { return parseHops(raw) }

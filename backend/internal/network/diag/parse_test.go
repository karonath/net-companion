package diag

import (
	"testing"
	"time"
)

func TestLatencyStats(t *testing.T) {
	samples := []time.Duration{
		10 * time.Millisecond,
		20 * time.Millisecond,
		0, // perte
		30 * time.Millisecond,
	}
	avg, jitter, loss := latencyStats(samples)
	// moyenne des non-nuls = (10+20+30)/3 = 20ms
	if avg != 20*time.Millisecond {
		t.Fatalf("avg = %v, want 20ms", avg)
	}
	if loss != 25 {
		t.Fatalf("loss = %d, want 25", loss)
	}
	if jitter <= 0 {
		t.Fatalf("jitter = %v, want > 0", jitter)
	}
}

func TestLatencyStatsAllLost(t *testing.T) {
	avg, _, loss := latencyStats([]time.Duration{0, 0})
	if avg != 0 || loss != 100 {
		t.Fatalf("avg=%v loss=%d, want 0/100", avg, loss)
	}
}

func TestParseTracerouteWindows(t *testing.T) {
	raw := `
Détermination de l'itinéraire vers example.com [93.184.216.34]

  1     2 ms     1 ms     1 ms  192.168.1.1
  2     *        *        *     Délai d'attente de la demande dépassé.
  3    12 ms    11 ms    13 ms  10.0.0.1

Itinéraire déterminé.
`
	hops := parseTracerouteWindows(raw)
	if len(hops) != 3 {
		t.Fatalf("got %d hops, want 3", len(hops))
	}
	if hops[0].Num != 1 || hops[0].Host != "192.168.1.1" {
		t.Fatalf("hop1 = %+v", hops[0])
	}
	if hops[1].Host != "*" {
		t.Fatalf("hop2 (timeout) = %+v", hops[1])
	}
	if hops[2].Host != "10.0.0.1" {
		t.Fatalf("hop3 = %+v", hops[2])
	}
}

func TestParseTracerouteUnix(t *testing.T) {
	raw := `traceroute to example.com (93.184.216.34), 30 hops max
 1  192.168.1.1  1.234 ms  1.111 ms  1.000 ms
 2  * * *
 3  10.0.0.1  12.5 ms  11.9 ms  13.1 ms
`
	hops := parseTracerouteUnix(raw)
	if len(hops) != 3 {
		t.Fatalf("got %d hops, want 3", len(hops))
	}
	if hops[0].Host != "192.168.1.1" || hops[2].Host != "10.0.0.1" {
		t.Fatalf("hops = %+v", hops)
	}
}

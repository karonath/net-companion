//go:build npcap

package lldp

import (
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/pcap"
)

// Capture écoute passivement les trames LLDP sur iface via Npcap.
func Capture(iface string, timeout time.Duration) Result {
	if timeout <= 0 {
		timeout = captureTimeout
	}
	handle, err := pcap.OpenLive(iface, 65536, true, pcap.BlockForever)
	if err != nil {
		return Result{Available: false, Reason: "ouverture pcap impossible : " + err.Error(), Neighbors: []Neighbor{}}
	}
	defer handle.Close()
	if err := handle.SetBPFFilter("ether proto 0x88cc"); err != nil {
		return Result{Available: false, Reason: "filtre BPF invalide : " + err.Error(), Neighbors: []Neighbor{}}
	}

	src := gopacket.NewPacketSource(handle, handle.LinkType())
	deadline := time.After(timeout)
	neighbors := []Neighbor{}
	for {
		select {
		case <-deadline:
			return Result{Available: true, Neighbors: neighbors}
		case pkt, ok := <-src.Packets():
			if !ok {
				return Result{Available: true, Neighbors: neighbors}
			}
			if n, err := ParseEthernet(pkt.Data()); err == nil {
				neighbors = append(neighbors, n)
			}
		}
	}
}

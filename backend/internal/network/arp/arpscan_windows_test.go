//go:build windows

package arp

import (
	"net"
	"testing"
)

func TestIPToULong(t *testing.T) {
	// 192.168.1.15 -> octet 0 (192) en poids faible
	got := ipToULong(net.ParseIP("192.168.1.15"))
	want := uint32(192) | uint32(168)<<8 | uint32(1)<<16 | uint32(15)<<24
	if got != want {
		t.Errorf("ipToULong = %#x, want %#x", got, want)
	}
}

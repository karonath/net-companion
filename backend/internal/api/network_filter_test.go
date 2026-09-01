package api

import "testing"

func TestIsMulticastOrBroadcastMAC(t *testing.T) {
	cases := map[string]bool{
		"01:00:5e:00:00:16": true,  // multicast IPv4
		"33:33:00:00:00:fb": true,  // multicast IPv6
		"ff:ff:ff:ff:ff:ff": true,  // broadcast
		"14:85:7f:27:dd:f5": false, // unicast réel
		"10:e9:92:21:22:30": false, // unicast réel
		"":                  false,
	}
	for mac, want := range cases {
		if got := isMulticastOrBroadcastMAC(mac); got != want {
			t.Errorf("isMulticastOrBroadcastMAC(%q) = %v, want %v", mac, got, want)
		}
	}
}

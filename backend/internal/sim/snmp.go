package sim

import "netcompanion/internal/network/portfinder"

// DemoMAC est l'adresse MAC de l'hôte connu du switch simulé.
const DemoMAC = "00:1a:2b:3c:4d:5e"

// demoSNMP rejoue les OID BRIDGE-MIB d'un switch de démonstration :
// MAC 00:1a:2b:3c:4d:5e sur bridge port 7 → ifName Gi1/0/5, VLAN 42,
// sysName SW-DEMO-01.
type demoSNMP struct{}

func (demoSNMP) Get(oid string) (string, bool) {
	m := map[string]string{
		"1.3.6.1.2.1.17.4.3.1.2.0.26.43.60.77.94": "7",                    // dot1dTpFdbPort.<mac>
		"1.3.6.1.2.1.17.1.4.1.2.7":                 "10001",               // dot1dBasePortIfIndex.7
		"1.3.6.1.2.1.31.1.1.1.1.10001":             "GigabitEthernet1/0/5", // ifName.10001
		"1.3.6.1.2.1.1.5.0":                        "SW-DEMO-01",          // sysName.0
	}
	v, ok := m[oid]
	return v, ok
}

func (demoSNMP) WalkStrings(root string) map[string]string {
	switch root {
	case "1.3.6.1.2.1.17.7.1.2.2.1.2": // dot1qTpFdbPort : index = <vlan>.<mac>
		return map[string]string{
			"1.3.6.1.2.1.17.7.1.2.2.1.2.42.0.26.43.60.77.94": "7",
		}
	// LLDP-MIB : SW-DEMO-01 voit SW-CORE-02 (port local 5, port distant Gi1/0/24)
	case "1.0.8802.1.1.2.1.4.1.1.9": // lldpRemSysName
		return map[string]string{"1.0.8802.1.1.2.1.4.1.1.9.0.5.1": "SW-CORE-02"}
	case "1.0.8802.1.1.2.1.4.1.1.7": // lldpRemPortId
		return map[string]string{"1.0.8802.1.1.2.1.4.1.1.7.0.5.1": "GigabitEthernet1/0/24"}
	case "1.0.8802.1.1.2.1.4.1.1.5": // lldpRemChassisId
		return map[string]string{"1.0.8802.1.1.2.1.4.1.1.5.0.5.1": "aa:bb:cc:dd:ee:01"}
	}
	return nil
}

// DemoSNMPClient renvoie un client SNMP simulant le switch de démonstration.
func DemoSNMPClient() portfinder.SNMPClient { return demoSNMP{} }

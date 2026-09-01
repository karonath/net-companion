package sim

import (
	"log"
	"os"
	"sync"
)

// Info décrit l'état du simulateur pour l'UI.
type Info struct {
	Enabled bool   `json:"enabled"`
	SSH     string `json:"ssh"`
	DemoMac string `json:"demoMac"`
	User    string `json:"user"`
}

var (
	mu      sync.RWMutex
	current Info
)

// Start démarre le simulateur si NC_SIMULATOR=1 (serveur SSH réel + client SNMP
// de démo). Sans la variable, ne fait rien (binaire de prod inchangé).
func Start() Info {
	if os.Getenv("NC_SIMULATOR") != "1" {
		return Info{}
	}
	_, addr, err := StartSSH("127.0.0.1:2222")
	if err != nil {
		log.Printf("simulateur SSH non démarré: %v", err)
		return Info{}
	}
	info := Info{Enabled: true, SSH: addr, DemoMac: DemoMAC, User: DemoSSHUser}
	mu.Lock()
	current = info
	mu.Unlock()
	log.Printf("SIMULATEUR actif — SSH %s (user %s/%s), MAC démo %s",
		addr, DemoSSHUser, DemoSSHPassword, DemoMAC)
	return info
}

// Current renvoie l'état courant du simulateur.
func Current() Info {
	mu.RLock()
	defer mu.RUnlock()
	return current
}

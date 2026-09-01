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
	stopFn  func() // arrête le serveur SSH simulé
)

// Start démarre le simulateur au boot si NC_SIMULATOR=1 (sinon no-op).
func Start() Info {
	if os.Getenv("NC_SIMULATOR") != "1" {
		return Info{}
	}
	info, err := Enable()
	if err != nil {
		log.Printf("simulateur non démarré: %v", err)
		return Info{}
	}
	return info
}

// Enable démarre le simulateur à la demande (serveur SSH réel + client SNMP de
// démo). Idempotent : si déjà actif, renvoie l'état courant sans redémarrer.
func Enable() (Info, error) {
	mu.Lock()
	defer mu.Unlock()
	if current.Enabled {
		return current, nil
	}
	listen := os.Getenv("NC_SIM_SSH_ADDR")
	if listen == "" {
		listen = "127.0.0.1:2222"
	}
	stop, addr, err := StartSSH(listen)
	if err != nil {
		return Info{}, err
	}
	stopFn = stop
	current = Info{Enabled: true, SSH: addr, DemoMac: DemoMAC, User: DemoSSHUser}
	log.Printf("SIMULATEUR actif — SSH %s (user %s/%s), MAC démo %s",
		addr, DemoSSHUser, DemoSSHPassword, DemoMAC)
	return current, nil
}

// Disable arrête le simulateur et repasse en mode réel. Idempotent.
func Disable() {
	mu.Lock()
	defer mu.Unlock()
	if stopFn != nil {
		stopFn()
		stopFn = nil
	}
	current = Info{}
	log.Print("SIMULATEUR désactivé")
}

// Current renvoie l'état courant du simulateur.
func Current() Info {
	mu.RLock()
	defer mu.RUnlock()
	return current
}

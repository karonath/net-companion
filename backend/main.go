// Command netcompanion démarre le serveur local et ouvre le navigateur.
package main

import (
	"crypto/rand"
	"encoding/hex"
	"log"
	"net"
	"net/http"
	"os"
	"time"

	"netcompanion/internal/browser"
	"netcompanion/internal/server"
	"netcompanion/internal/sim"
	"netcompanion/internal/vault"
	"netcompanion/web"
)

const defaultAddr = "127.0.0.1:8080"

// newToken génère un jeton de session aléatoire (32 octets hex).
func newToken() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		log.Fatalf("génération du jeton impossible: %v", err)
	}
	return hex.EncodeToString(b)
}

func main() {
	addr := defaultAddr
	if v := os.Getenv("NC_ADDR"); v != "" {
		addr = v
	}

	fsys, err := web.Dist()
	if err != nil {
		log.Fatalf("frontend embarqué illisible: %v", err)
	}

	vaultPath, err := vault.DefaultPath()
	if err != nil {
		log.Fatalf("chemin du coffre indéterminé: %v", err)
	}
	v := vault.New(vaultPath)
	log.Printf("coffre: %s", vaultPath)

	token := newToken() // jeton de session (non journalisé)

	sim.Start() // simulateur d'équipement si NC_SIMULATOR=1 (sinon no-op)

	ln, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("impossible d'écouter sur %s (déjà utilisé ?): %v", addr, err)
	}

	url := "http://" + addr
	log.Printf("Net-Companion prêt sur %s", url)

	// Ouvre le navigateur une fois le serveur en écoute.
	go func() {
		time.Sleep(300 * time.Millisecond)
		if err := browser.Open(url); err != nil {
			log.Printf("ouverture navigateur impossible: %v (ouvrez %s manuellement)", err, url)
		}
	}()

	if err := http.Serve(ln, server.Handler(fsys, v, token)); err != nil {
		log.Fatalf("serveur arrêté: %v", err)
	}
}

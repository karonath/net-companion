// Command netcompanion démarre le serveur local et ouvre le navigateur.
package main

import (
	"log"
	"net"
	"net/http"
	"time"

	"netcompanion/internal/browser"
	"netcompanion/internal/server"
	"netcompanion/web"
)

const addr = "127.0.0.1:8080"

func main() {
	fsys, err := web.Dist()
	if err != nil {
		log.Fatalf("frontend embarqué illisible: %v", err)
	}

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

	if err := http.Serve(ln, server.Handler(fsys)); err != nil {
		log.Fatalf("serveur arrêté: %v", err)
	}
}

// Package sim fournit un simulateur d'équipement réseau (SSH + SNMP) pour
// démontrer et tester Port-Finder et Config-Diff sans matériel réel.
package sim

import (
	"crypto/ed25519"
	"crypto/rand"
	"errors"
	"net"
	"strings"

	"golang.org/x/crypto/ssh"
)

var errAuth = errors.New("authentification refusée")

// Identifiants de démo acceptés par le serveur SSH simulé.
const (
	DemoSSHUser     = "admin"
	DemoSSHPassword = "demo"
)

const demoRunningConfig = `hostname SW-DEMO-01
!
vlan 42
 name USERS
!
interface GigabitEthernet1/0/5
 switchport access vlan 42
 no shutdown
!
ip route 0.0.0.0 0.0.0.0 192.168.1.254
`

const demoStartupConfig = `hostname SW-DEMO-01
!
vlan 42
 name USERS
!
interface GigabitEthernet1/0/5
 switchport access vlan 42
!
ip route 0.0.0.0 0.0.0.0 192.168.1.1
`

// StartSSH démarre un serveur SSH simulé sur addr. Renvoie une fonction d'arrêt
// et l'adresse réelle d'écoute (utile avec le port 0).
func StartSSH(addr string) (stop func(), actualAddr string, err error) {
	cfg := &ssh.ServerConfig{
		PasswordCallback: func(c ssh.ConnMetadata, pass []byte) (*ssh.Permissions, error) {
			if c.User() == DemoSSHUser && string(pass) == DemoSSHPassword {
				return &ssh.Permissions{}, nil
			}
			return nil, errAuth
		},
	}

	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, "", err
	}
	signer, err := ssh.NewSignerFromKey(priv)
	if err != nil {
		return nil, "", err
	}
	cfg.AddHostKey(signer)

	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, "", err
	}

	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return // listener fermé
			}
			go handleSSHConn(conn, cfg)
		}
	}()

	return func() { _ = ln.Close() }, ln.Addr().String(), nil
}

func handleSSHConn(conn net.Conn, cfg *ssh.ServerConfig) {
	sshConn, chans, reqs, err := ssh.NewServerConn(conn, cfg)
	if err != nil {
		_ = conn.Close()
		return
	}
	defer sshConn.Close()
	go ssh.DiscardRequests(reqs)

	for newCh := range chans {
		if newCh.ChannelType() != "session" {
			_ = newCh.Reject(ssh.UnknownChannelType, "seul 'session' est supporté")
			continue
		}
		ch, chReqs, err := newCh.Accept()
		if err != nil {
			continue
		}
		go handleSession(ch, chReqs)
	}
}

func handleSession(ch ssh.Channel, reqs <-chan *ssh.Request) {
	for req := range reqs {
		if req.Type != "exec" {
			if req.WantReply {
				_ = req.Reply(false, nil)
			}
			continue
		}
		var payload struct{ Command string }
		_ = ssh.Unmarshal(req.Payload, &payload)
		if req.WantReply {
			_ = req.Reply(true, nil)
		}

		_, _ = ch.Write([]byte(configFor(payload.Command)))
		// exit-status 0
		_, _ = ch.SendRequest("exit-status", false, ssh.Marshal(struct{ Status uint32 }{0}))
		_ = ch.Close()
		return
	}
}

func configFor(cmd string) string {
	switch {
	case strings.Contains(cmd, "running"):
		return demoRunningConfig
	case strings.Contains(cmd, "startup"):
		return demoStartupConfig
	default:
		return "% commande inconnue\n"
	}
}

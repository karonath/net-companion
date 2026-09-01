package configdiff

import (
	"net"
	"time"

	"golang.org/x/crypto/ssh"

	"netcompanion/internal/models"
)

// Runner exécute une commande distante et renvoie sa sortie.
type Runner interface {
	Run(cmd string) (string, error)
}

// sshRunner exécute les commandes via une session SSH.
type sshRunner struct {
	client *ssh.Client
}

// NewSSHRunner ouvre une connexion SSH ; le second retour ferme la connexion.
func NewSSHRunner(host string, cred models.SSHCredential) (Runner, func() error, error) {
	var auth []ssh.AuthMethod
	if cred.PrivateKey != "" {
		if signer, err := ssh.ParsePrivateKey([]byte(cred.PrivateKey)); err == nil {
			auth = append(auth, ssh.PublicKeys(signer))
		}
	}
	if cred.Password != "" {
		auth = append(auth, ssh.Password(cred.Password))
	}

	cfg := &ssh.ClientConfig{
		User:            cred.Username,
		Auth:            auth,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // terrain : équipements sans PKI gérée
		Timeout:         5 * time.Second,
	}
	addr := host
	if _, _, err := net.SplitHostPort(host); err != nil {
		addr = host + ":22"
	}
	client, err := ssh.Dial("tcp", addr, cfg)
	if err != nil {
		return nil, nil, err
	}
	return &sshRunner{client: client}, client.Close, nil
}

func (r *sshRunner) Run(cmd string) (string, error) {
	sess, err := r.client.NewSession()
	if err != nil {
		return "", err
	}
	defer sess.Close()
	out, err := sess.CombinedOutput(cmd)
	return string(out), err
}

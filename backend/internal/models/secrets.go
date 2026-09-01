// Package models contient les DTO JSON partagés.
package models

// SNMPCredential est une community SNMP nommée.
type SNMPCredential struct {
	ID        string `json:"id"`
	Label     string `json:"label"`
	Community string `json:"community"`
	Version   string `json:"version"` // ex: "v2c"
}

// SSHCredential est un identifiant SSH (mot de passe et/ou clé privée).
type SSHCredential struct {
	ID         string `json:"id"`
	Label      string `json:"label"`
	Username   string `json:"username"`
	Password   string `json:"password,omitempty"`
	PrivateKey string `json:"privateKey,omitempty"`
}

// Secrets est l'ensemble déchiffré détenu en RAM.
type Secrets struct {
	SNMP []SNMPCredential `json:"snmp"`
	SSH  []SSHCredential  `json:"ssh"`
}

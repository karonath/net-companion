// Package models contient les DTO JSON partagés.
package models

// SNMPCredential est un identifiant SNMP (v2c community, ou v3 USM).
type SNMPCredential struct {
	ID        string `json:"id"`
	Label     string `json:"label"`
	Version   string `json:"version"` // "v2c" | "v3"
	Community string `json:"community,omitempty"` // v2c

	// SNMPv3 (USM)
	SecurityName   string `json:"securityName,omitempty"`
	SecurityLevel  string `json:"securityLevel,omitempty"` // noAuthNoPriv | authNoPriv | authPriv
	AuthProtocol   string `json:"authProtocol,omitempty"`  // MD5 | SHA | SHA256 | SHA512
	AuthPassphrase string `json:"authPassphrase,omitempty"`
	PrivProtocol   string `json:"privProtocol,omitempty"` // DES | AES | AES192 | AES256
	PrivPassphrase string `json:"privPassphrase,omitempty"`
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

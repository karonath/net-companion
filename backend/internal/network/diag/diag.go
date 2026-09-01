// Package diag fournit des diagnostics de connectivité réseau (sans matériel).
package diag

// Statuts possibles d'un check.
const (
	StatusOK   = "ok"
	StatusWarn = "warn"
	StatusFail = "fail"
)

// Check est le résultat d'un diagnostic unitaire.
type Check struct {
	Name   string `json:"name"`
	Status string `json:"status"`
	Detail string `json:"detail"`
}

// Hop est un saut de traceroute.
type Hop struct {
	Num  int    `json:"num"`
	Host string `json:"host"`
	MS   string `json:"ms"`
}

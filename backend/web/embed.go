// Package web embarque le frontend Vue buildé dans le binaire.
package web

import (
	"embed"
	"io/fs"
)

//go:embed all:dist
var distFS embed.FS

// Dist renvoie le frontend buildé comme filesystem raciné sur dist/.
func Dist() (fs.FS, error) {
	return fs.Sub(distFS, "dist")
}

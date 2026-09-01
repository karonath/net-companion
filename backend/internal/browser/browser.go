// Package browser ouvre une URL dans le navigateur par défaut de l'OS.
package browser

import (
	"os/exec"
	"runtime"
)

// command renvoie le nom et les arguments de la commande OS ouvrant url.
func command(goos, url string) (string, []string) {
	switch goos {
	case "windows":
		// "start" premier arg vide = titre de fenêtre (évite d'avaler l'URL).
		return "cmd", []string{"/c", "start", "", url}
	case "darwin":
		return "open", []string{url}
	default:
		return "xdg-open", []string{url}
	}
}

// Open lance le navigateur par défaut sur url sans bloquer.
func Open(url string) error {
	name, args := command(runtime.GOOS, url)
	return exec.Command(name, args...).Start()
}

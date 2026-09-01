//go:build !windows

package macspoof

import (
	"os"
	"os/exec"
	"runtime"
)

func isElevated() bool { return os.Geteuid() == 0 }

func execStep(s Step) error {
	return exec.Command(s.Command, s.Args...).Run()
}

// Apply vérifie l'élévation puis exécute le plan pour l'OS courant.
func Apply(iface, mac string) (Plan, error) {
	p, err := BuildPlan(runtime.GOOS, iface, mac)
	if err != nil {
		return Plan{}, err
	}
	if !isElevated() {
		return p, ErrElevationRequired
	}
	if err := run(p); err != nil {
		return p, err
	}
	return p, nil
}

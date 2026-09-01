//go:build windows

package macspoof

import (
	"os/exec"
	"runtime"

	"golang.org/x/sys/windows"
)

func isElevated() bool {
	var sid *windows.SID
	// S-1-5-32-544 = groupe Administrateurs local.
	err := windows.AllocateAndInitializeSid(
		&windows.SECURITY_NT_AUTHORITY, 2,
		windows.SECURITY_BUILTIN_DOMAIN_RID, windows.DOMAIN_ALIAS_RID_ADMINS,
		0, 0, 0, 0, 0, 0, &sid)
	if err != nil {
		return false
	}
	defer windows.FreeSid(sid)
	token := windows.Token(0) // token du process courant
	member, err := token.IsMember(sid)
	return err == nil && member
}

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

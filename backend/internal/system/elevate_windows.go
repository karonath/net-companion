//go:build windows

package system

import (
	"os"
	"strings"
	"syscall"

	"golang.org/x/sys/windows"
)

// IsElevated indique si le process courant tourne réellement en administrateur (UAC).
func IsElevated() bool {
	return windows.GetCurrentProcessToken().IsElevated()
}

// Relaunch relance l'exécutable courant avec élévation (déclenche l'UAC via le
// verbe « runas »). Renvoie une erreur si l'utilisateur annule l'UAC.
func Relaunch() error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	verb, _ := syscall.UTF16PtrFromString("runas")
	file, _ := syscall.UTF16PtrFromString(exe)
	var args *uint16
	if len(os.Args) > 1 {
		args, _ = syscall.UTF16PtrFromString(strings.Join(os.Args[1:], " "))
	}
	var dir *uint16
	if cwd, err := os.Getwd(); err == nil {
		dir, _ = syscall.UTF16PtrFromString(cwd)
	}
	return windows.ShellExecute(0, verb, file, args, dir, windows.SW_SHOWNORMAL)
}

//go:build !windows

package system

import "os"

// IsElevated : sous Unix, l'élévation correspond à l'UID 0 (root).
func IsElevated() bool { return os.Geteuid() == 0 }

// Relaunch n'est pas supporté hors Windows (pas d'UAC).
func Relaunch() error { return ErrRelaunchUnsupported }

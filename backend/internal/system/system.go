// Package system gère l'état d'élévation (droits administrateur) et la relance
// élevée à la demande, sans forcer l'UAC au démarrage (philosophie Lite/USB).
package system

import "errors"

// ErrRelaunchUnsupported : la relance élevée n'existe que sous Windows.
var ErrRelaunchUnsupported = errors.New("relance en administrateur non supportée sur cette plateforme")

package api

import (
	"errors"
	"net/http"

	"netcompanion/internal/models"
	"netcompanion/internal/network/configdiff"
	"netcompanion/internal/network/portfinder"
	"netcompanion/internal/vault"
)

// registerVaultTest ajoute le test d'un credential (SNMP/SSH) contre un équipement.
func registerVaultTest(mux *http.ServeMux, v *vault.Vault) {
	mux.HandleFunc("POST /api/vault/test", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Type     string `json:"type"` // "snmp" | "ssh"
			ID       string `json:"id"`
			DeviceIP string `json:"deviceIp"`
		}
		if !decodeJSON(w, r, &body) {
			return
		}
		if body.DeviceIP == "" {
			writeError(w, http.StatusBadRequest, errors.New("deviceIp requis"))
			return
		}
		snap, err := v.Snapshot()
		if err != nil {
			writeLocked(w, err)
			return
		}

		switch body.Type {
		case "snmp":
			cred, ok := findSNMP(snap.SNMP, body.ID)
			if !ok {
				writeError(w, http.StatusNotFound, errors.New("credential SNMP introuvable"))
				return
			}
			writeJSON(w, http.StatusOK, testSNMP(body.DeviceIP, cred))
		case "ssh":
			cred, ok := findSSH(snap.SSH, body.ID)
			if !ok {
				writeError(w, http.StatusNotFound, errors.New("credential SSH introuvable"))
				return
			}
			writeJSON(w, http.StatusOK, testSSH(body.DeviceIP, cred))
		default:
			writeError(w, http.StatusBadRequest, errors.New("type doit être 'snmp' ou 'ssh'"))
		}
	})
}

func findSNMP(list []models.SNMPCredential, id string) (models.SNMPCredential, bool) {
	for _, c := range list {
		if c.ID == id {
			return c, true
		}
	}
	return models.SNMPCredential{}, false
}

func findSSH(list []models.SSHCredential, id string) (models.SSHCredential, bool) {
	for _, c := range list {
		if c.ID == id {
			return c, true
		}
	}
	return models.SSHCredential{}, false
}

func testSNMP(device string, cred models.SNMPCredential) map[string]any {
	client, closeFn, err := portfinder.NewGoSNMP(device, cred)
	if err != nil {
		return map[string]any{"ok": false, "detail": "connexion impossible : " + err.Error()}
	}
	defer closeFn()
	if name, ok := client.Get("1.3.6.1.2.1.1.5.0"); ok { // sysName.0
		return map[string]any{"ok": true, "detail": "OK — sysName : " + name}
	}
	return map[string]any{"ok": false, "detail": "pas de réponse SNMP (community/utilisateur ou accès invalide)"}
}

func testSSH(device string, cred models.SSHCredential) map[string]any {
	_, closeFn, err := configdiff.NewSSHRunner(device, cred)
	if err != nil {
		return map[string]any{"ok": false, "detail": "connexion SSH refusée : " + err.Error()}
	}
	_ = closeFn()
	return map[string]any{"ok": true, "detail": "OK — connexion SSH établie"}
}

// Package api expose les handlers HTTP REST branchés sur le domaine métier.
package api

import (
	"encoding/json"
	"errors"
	"net/http"

	"netcompanion/internal/models"
	"netcompanion/internal/vault"
)

// Register enregistre les routes /api/vault/* sur mux, servies par v.
func Register(mux *http.ServeMux, v *vault.Vault) {
	mux.HandleFunc("GET /api/vault/status", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, v.Status())
	})

	mux.HandleFunc("POST /api/vault/init", func(w http.ResponseWriter, r *http.Request) {
		pin, ok := decodePIN(w, r)
		if !ok {
			return
		}
		if err := v.Init(pin); err != nil {
			if errors.Is(err, vault.ErrAlreadyInitialized) {
				writeError(w, http.StatusConflict, err)
				return
			}
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		writeJSON(w, http.StatusOK, v.Status())
	})

	mux.HandleFunc("POST /api/vault/unlock", func(w http.ResponseWriter, r *http.Request) {
		pin, ok := decodePIN(w, r)
		if !ok {
			return
		}
		if err := v.Unlock(pin); err != nil {
			writeError(w, http.StatusUnauthorized, err)
			return
		}
		writeJSON(w, http.StatusOK, v.Status())
	})

	mux.HandleFunc("POST /api/vault/lock", func(w http.ResponseWriter, r *http.Request) {
		v.Lock()
		writeJSON(w, http.StatusOK, v.Status())
	})

	mux.HandleFunc("GET /api/vault/secrets/snmp", func(w http.ResponseWriter, r *http.Request) {
		snap, err := v.Snapshot()
		if err != nil {
			writeLocked(w, err)
			return
		}
		writeJSON(w, http.StatusOK, nonNilSNMP(snap.SNMP))
	})

	mux.HandleFunc("POST /api/vault/secrets/snmp", func(w http.ResponseWriter, r *http.Request) {
		var c models.SNMPCredential
		if !decodeJSON(w, r, &c) {
			return
		}
		created, err := v.AddSNMP(c)
		if err != nil {
			writeLocked(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, created)
	})

	mux.HandleFunc("DELETE /api/vault/secrets/snmp/{id}", func(w http.ResponseWriter, r *http.Request) {
		if err := v.DeleteSNMP(r.PathValue("id")); err != nil {
			writeLocked(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})

	mux.HandleFunc("GET /api/vault/secrets/ssh", func(w http.ResponseWriter, r *http.Request) {
		snap, err := v.Snapshot()
		if err != nil {
			writeLocked(w, err)
			return
		}
		writeJSON(w, http.StatusOK, nonNilSSH(snap.SSH))
	})

	mux.HandleFunc("POST /api/vault/secrets/ssh", func(w http.ResponseWriter, r *http.Request) {
		var c models.SSHCredential
		if !decodeJSON(w, r, &c) {
			return
		}
		created, err := v.AddSSH(c)
		if err != nil {
			writeLocked(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, created)
	})

	mux.HandleFunc("DELETE /api/vault/secrets/ssh/{id}", func(w http.ResponseWriter, r *http.Request) {
		if err := v.DeleteSSH(r.PathValue("id")); err != nil {
			writeLocked(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})

	registerNetwork(mux, v)
	registerNAC(mux)
	registerConfigDiff(mux, v)
	registerDiag(mux)
	registerSim(mux)
	registerCheckup(mux, v)
	registerVaultTest(mux, v)
	registerConfig(mux, v)
}

func decodePIN(w http.ResponseWriter, r *http.Request) (string, bool) {
	var body struct {
		PIN string `json:"pin"`
	}
	if !decodeJSON(w, r, &body) {
		return "", false
	}
	if body.PIN == "" {
		writeError(w, http.StatusBadRequest, errors.New("pin requis"))
		return "", false
	}
	return body.PIN, true
}

func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) bool {
	if err := json.NewDecoder(r.Body).Decode(dst); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return false
	}
	return true
}

func writeJSON(w http.ResponseWriter, code int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, code int, err error) {
	writeJSON(w, code, map[string]string{"error": err.Error()})
}

func writeLocked(w http.ResponseWriter, err error) {
	if errors.Is(err, vault.ErrLocked) {
		writeError(w, http.StatusLocked, err)
		return
	}
	writeError(w, http.StatusInternalServerError, err)
}

func nonNilSNMP(s []models.SNMPCredential) []models.SNMPCredential {
	if s == nil {
		return []models.SNMPCredential{}
	}
	return s
}

func nonNilSSH(s []models.SSHCredential) []models.SSHCredential {
	if s == nil {
		return []models.SSHCredential{}
	}
	return s
}

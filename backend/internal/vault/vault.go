package vault

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"

	"netcompanion/internal/models"
)

// Erreurs publiques du coffre-fort.
var (
	ErrUnlockFailed       = errors.New("déverrouillage impossible (PIN erroné ou fichier corrompu)")
	ErrLocked             = errors.New("coffre verrouillé")
	ErrAlreadyInitialized = errors.New("coffre déjà initialisé")
	ErrNotInitialized     = errors.New("coffre non initialisé")
)

const fileVersion = 1

const saltLen = 16

// fileFormat est la structure sérialisée dans vault.dat (JSON, []byte en base64).
type fileFormat struct {
	Version int    `json:"version"`
	Salt    []byte `json:"salt"`
	Nonce   []byte `json:"nonce"`
	Cipher  []byte `json:"cipher"`
}

// Status décrit l'état observable du coffre.
type Status struct {
	Initialized bool `json:"initialized"`
	Unlocked    bool `json:"unlocked"`
}

// Vault est le coffre-fort embarqué. Sûr pour un usage concurrent.
type Vault struct {
	mu      sync.Mutex
	path    string
	key     []byte          // nil si verrouillé
	secrets *models.Secrets // nil si verrouillé
}

// New crée un coffre pointant sur le fichier path (non lu tant qu'on n'agit pas).
func New(path string) *Vault { return &Vault{path: path} }

// DefaultPath renvoie l'emplacement standard du coffre dans le profil utilisateur.
func DefaultPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "netcompanion", "vault.dat"), nil
}

// Status renvoie l'état courant (initialisé = le fichier existe).
func (v *Vault) Status() Status {
	v.mu.Lock()
	defer v.mu.Unlock()
	_, err := os.Stat(v.path)
	return Status{Initialized: err == nil, Unlocked: v.key != nil}
}

// Init crée le coffre avec un PIN et un jeu de secrets vide, puis le déverrouille.
func (v *Vault) Init(pin string) error {
	v.mu.Lock()
	defer v.mu.Unlock()
	if _, err := os.Stat(v.path); err == nil {
		return ErrAlreadyInitialized
	}
	salt := make([]byte, saltLen)
	if _, err := rand.Read(salt); err != nil {
		return err
	}
	key := deriveKey(pin, salt)
	secrets := &models.Secrets{}
	if err := v.persist(salt, key, secrets); err != nil {
		return err
	}
	v.key = key
	v.secrets = secrets
	return nil
}

// Unlock lit le fichier, dérive la clé et déchiffre les secrets en RAM.
func (v *Vault) Unlock(pin string) error {
	v.mu.Lock()
	defer v.mu.Unlock()
	raw, err := os.ReadFile(v.path)
	if err != nil {
		return ErrNotInitialized
	}
	var f fileFormat
	if err := json.Unmarshal(raw, &f); err != nil {
		return ErrUnlockFailed
	}
	key := deriveKey(pin, f.Salt)
	plaintext, err := open(key, f.Nonce, f.Cipher)
	if err != nil {
		return ErrUnlockFailed
	}
	var secrets models.Secrets
	if err := json.Unmarshal(plaintext, &secrets); err != nil {
		return ErrUnlockFailed
	}
	v.key = key
	v.secrets = &secrets
	return nil
}

// Lock efface la clé et les secrets de la RAM.
func (v *Vault) Lock() {
	v.mu.Lock()
	defer v.mu.Unlock()
	for i := range v.key {
		v.key[i] = 0
	}
	v.key = nil
	v.secrets = nil
}

// Snapshot renvoie une copie des secrets courants (requiert déverrouillé).
func (v *Vault) Snapshot() (models.Secrets, error) {
	v.mu.Lock()
	defer v.mu.Unlock()
	if v.secrets == nil {
		return models.Secrets{}, ErrLocked
	}
	cp := models.Secrets{
		SNMP: append([]models.SNMPCredential(nil), v.secrets.SNMP...),
		SSH:  append([]models.SSHCredential(nil), v.secrets.SSH...),
	}
	return cp, nil
}

// AddSNMP ajoute une community, assigne un ID, persiste, renvoie l'élément créé.
func (v *Vault) AddSNMP(c models.SNMPCredential) (models.SNMPCredential, error) {
	v.mu.Lock()
	defer v.mu.Unlock()
	if v.secrets == nil {
		return models.SNMPCredential{}, ErrLocked
	}
	c.ID = newID()
	v.secrets.SNMP = append(v.secrets.SNMP, c)
	if err := v.save(); err != nil {
		return models.SNMPCredential{}, err
	}
	return c, nil
}

// DeleteSNMP supprime la community d'ID id (no-op si absente) puis persiste.
func (v *Vault) DeleteSNMP(id string) error {
	v.mu.Lock()
	defer v.mu.Unlock()
	if v.secrets == nil {
		return ErrLocked
	}
	v.secrets.SNMP = filterOutSNMP(v.secrets.SNMP, id)
	return v.save()
}

// AddSSH ajoute un identifiant SSH, assigne un ID, persiste.
func (v *Vault) AddSSH(c models.SSHCredential) (models.SSHCredential, error) {
	v.mu.Lock()
	defer v.mu.Unlock()
	if v.secrets == nil {
		return models.SSHCredential{}, ErrLocked
	}
	c.ID = newID()
	v.secrets.SSH = append(v.secrets.SSH, c)
	if err := v.save(); err != nil {
		return models.SSHCredential{}, err
	}
	return c, nil
}

// DeleteSSH supprime l'identifiant SSH d'ID id puis persiste.
func (v *Vault) DeleteSSH(id string) error {
	v.mu.Lock()
	defer v.mu.Unlock()
	if v.secrets == nil {
		return ErrLocked
	}
	v.secrets.SSH = filterOutSSH(v.secrets.SSH, id)
	return v.save()
}

// save re-scelle les secrets courants avec un nouveau nonce et réécrit le fichier.
// Le sel est relu depuis le fichier existant pour rester stable.
func (v *Vault) save() error {
	raw, err := os.ReadFile(v.path)
	if err != nil {
		return err
	}
	var f fileFormat
	if err := json.Unmarshal(raw, &f); err != nil {
		return err
	}
	return v.persist(f.Salt, v.key, v.secrets)
}

// persist chiffre secrets avec key et écrit atomiquement le fichier.
func (v *Vault) persist(salt, key []byte, secrets *models.Secrets) error {
	plaintext, err := json.Marshal(secrets)
	if err != nil {
		return err
	}
	nonce, ct, err := seal(key, plaintext)
	if err != nil {
		return err
	}
	out, err := json.Marshal(fileFormat{Version: fileVersion, Salt: salt, Nonce: nonce, Cipher: ct})
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(v.path), 0o700); err != nil {
		return err
	}
	tmp := v.path + ".tmp"
	if err := os.WriteFile(tmp, out, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, v.path)
}

func newID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func filterOutSNMP(in []models.SNMPCredential, id string) []models.SNMPCredential {
	out := in[:0]
	for _, c := range in {
		if c.ID != id {
			out = append(out, c)
		}
	}
	return out
}

func filterOutSSH(in []models.SSHCredential, id string) []models.SSHCredential {
	out := in[:0]
	for _, c := range in {
		if c.ID != id {
			out = append(out, c)
		}
	}
	return out
}

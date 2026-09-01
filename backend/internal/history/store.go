package history

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Store persiste les snapshots en fichiers JSON dans un répertoire.
type Store struct {
	dir string
}

// NewStore crée un store sur dir (créé à la demande).
func NewStore(dir string) *Store { return &Store{dir: dir} }

// DefaultDir renvoie le répertoire d'historique dans le profil utilisateur.
func DefaultDir() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "netcompanion", "history"), nil
}

// Save écrit le snapshot en <dir>/<id>.json.
func (s *Store) Save(snap Snapshot) error {
	if err := os.MkdirAll(s.dir, 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(snap, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(s.dir, snap.ID+".json"), data, 0o600)
}

// List renvoie les métadonnées de tous les snapshots, du plus récent au plus ancien.
func (s *Store) List() ([]Meta, error) {
	entries, err := os.ReadDir(s.dir)
	if os.IsNotExist(err) {
		return []Meta{}, nil
	}
	if err != nil {
		return nil, err
	}
	var metas []Meta
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		snap, err := s.Get(strings.TrimSuffix(e.Name(), ".json"))
		if err != nil {
			continue
		}
		metas = append(metas, Meta{ID: snap.ID, Timestamp: snap.Timestamp, HostCount: len(snap.Hosts)})
	}
	sort.Slice(metas, func(i, j int) bool { return metas[i].Timestamp.After(metas[j].Timestamp) })
	if metas == nil {
		metas = []Meta{}
	}
	return metas, nil
}

// Get lit un snapshot par son id.
func (s *Store) Get(id string) (Snapshot, error) {
	data, err := os.ReadFile(filepath.Join(s.dir, id+".json"))
	if err != nil {
		return Snapshot{}, err
	}
	var snap Snapshot
	err = json.Unmarshal(data, &snap)
	return snap, err
}

// Latest renvoie le snapshot le plus récent, ou false s'il n'y en a pas.
func (s *Store) Latest() (Snapshot, bool) {
	metas, err := s.List()
	if err != nil || len(metas) == 0 {
		return Snapshot{}, false
	}
	snap, err := s.Get(metas[0].ID)
	if err != nil {
		return Snapshot{}, false
	}
	return snap, true
}

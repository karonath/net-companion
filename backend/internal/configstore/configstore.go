// Package configstore persiste les configurations d'équipements et gère la baseline.
package configstore

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Snapshot est une sauvegarde de running-config d'un équipement.
type Snapshot struct {
	ID        string    `json:"id"`
	Device    string    `json:"device"`
	Timestamp time.Time `json:"timestamp"`
	Running   string    `json:"running"`
	Baseline  bool      `json:"baseline"`
}

// DeviceMeta résume l'état d'un équipement dans l'historique.
type DeviceMeta struct {
	Device      string    `json:"device"`
	Count       int       `json:"count"`
	Last        time.Time `json:"last"`
	HasBaseline bool      `json:"hasBaseline"`
}

// Store persiste les snapshots par équipement.
type Store struct{ dir string }

// NewStore crée un store sur dir.
func NewStore(dir string) *Store { return &Store{dir: dir} }

// DefaultDir renvoie le répertoire des configs dans le profil utilisateur.
func DefaultDir() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "netcompanion", "configs"), nil
}

func safeDevice(d string) string {
	return strings.NewReplacer(":", "_", "/", "_", "\\", "_").Replace(d)
}

func (s *Store) deviceDir(device string) string {
	return filepath.Join(s.dir, safeDevice(device))
}

// Save écrit un snapshot sous <dir>/<device>/<id>.json.
func (s *Store) Save(snap Snapshot) error {
	dir := s.deviceDir(snap.Device)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(snap, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, snap.ID+".json"), data, 0o600)
}

// List renvoie les snapshots d'un équipement, du plus récent au plus ancien.
func (s *Store) List(device string) ([]Snapshot, error) {
	entries, err := os.ReadDir(s.deviceDir(device))
	if os.IsNotExist(err) {
		return []Snapshot{}, nil
	}
	if err != nil {
		return nil, err
	}
	var out []Snapshot
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		snap, err := s.Get(device, strings.TrimSuffix(e.Name(), ".json"))
		if err == nil {
			out = append(out, snap)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Timestamp.After(out[j].Timestamp) })
	if out == nil {
		out = []Snapshot{}
	}
	return out, nil
}

// Get lit un snapshot.
func (s *Store) Get(device, id string) (Snapshot, error) {
	data, err := os.ReadFile(filepath.Join(s.deviceDir(device), id+".json"))
	if err != nil {
		return Snapshot{}, err
	}
	var snap Snapshot
	err = json.Unmarshal(data, &snap)
	return snap, err
}

// Latest renvoie le snapshot le plus récent d'un équipement.
func (s *Store) Latest(device string) (Snapshot, bool) {
	snaps, err := s.List(device)
	if err != nil || len(snaps) == 0 {
		return Snapshot{}, false
	}
	return snaps[0], true
}

// SetBaseline marque un snapshot comme baseline (et désactive les autres).
func (s *Store) SetBaseline(device, id string) error {
	snaps, err := s.List(device)
	if err != nil {
		return err
	}
	for _, snap := range snaps {
		want := snap.ID == id
		if snap.Baseline != want {
			snap.Baseline = want
			if err := s.Save(snap); err != nil {
				return err
			}
		}
	}
	return nil
}

// Baseline renvoie le snapshot baseline d'un équipement, s'il existe.
func (s *Store) Baseline(device string) (Snapshot, bool) {
	snaps, _ := s.List(device)
	for _, snap := range snaps {
		if snap.Baseline {
			return snap, true
		}
	}
	return Snapshot{}, false
}

// ListDevices agrège les équipements connus.
func (s *Store) ListDevices() ([]DeviceMeta, error) {
	entries, err := os.ReadDir(s.dir)
	if os.IsNotExist(err) {
		return []DeviceMeta{}, nil
	}
	if err != nil {
		return nil, err
	}
	var out []DeviceMeta
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		// retrouver le nom de device réel depuis un snapshot
		snaps, err := s.listByDir(e.Name())
		if err != nil || len(snaps) == 0 {
			continue
		}
		meta := DeviceMeta{Device: snaps[0].Device, Count: len(snaps), Last: snaps[0].Timestamp}
		for _, snap := range snaps {
			if snap.Baseline {
				meta.HasBaseline = true
			}
		}
		out = append(out, meta)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Device < out[j].Device })
	if out == nil {
		out = []DeviceMeta{}
	}
	return out, nil
}

func (s *Store) listByDir(dirName string) ([]Snapshot, error) {
	entries, err := os.ReadDir(filepath.Join(s.dir, dirName))
	if err != nil {
		return nil, err
	}
	var out []Snapshot
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(s.dir, dirName, e.Name()))
		if err != nil {
			continue
		}
		var snap Snapshot
		if json.Unmarshal(data, &snap) == nil {
			out = append(out, snap)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Timestamp.After(out[j].Timestamp) })
	return out, nil
}

package sim

import "testing"

func TestEnableIdempotent(t *testing.T) {
	// port éphémère pour ne pas entrer en conflit avec une instance existante
	t.Setenv("NC_SIM_SSH_ADDR", "127.0.0.1:0")
	info, err := Enable()
	if err != nil {
		t.Fatalf("Enable: %v", err)
	}
	if !info.Enabled || info.SSH == "" {
		t.Fatalf("info = %+v", info)
	}
	if !Current().Enabled {
		t.Fatal("Current() devrait être actif")
	}
	// 2e appel : même adresse, pas de nouveau serveur
	info2, err := Enable()
	if err != nil {
		t.Fatalf("Enable 2: %v", err)
	}
	if info2.SSH != info.SSH {
		t.Fatalf("SSH a changé au 2e Enable: %s vs %s", info2.SSH, info.SSH)
	}
}

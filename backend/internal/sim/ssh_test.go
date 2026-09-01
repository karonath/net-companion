package sim_test

import (
	"testing"

	"netcompanion/internal/models"
	"netcompanion/internal/network/configdiff"
	"netcompanion/internal/sim"
)

func TestSSHSimulatorServesConfigs(t *testing.T) {
	stop, addr, err := sim.StartSSH("127.0.0.1:0")
	if err != nil {
		t.Fatalf("StartSSH: %v", err)
	}
	defer stop()

	runner, closeFn, err := configdiff.NewSSHRunner(addr, models.SSHCredential{
		Username: sim.DemoSSHUser,
		Password: sim.DemoSSHPassword,
	})
	if err != nil {
		t.Fatalf("connexion SSH au simulateur: %v", err)
	}
	defer closeFn()

	running, startup, err := configdiff.Fetch(runner)
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	if running == "" || startup == "" {
		t.Fatalf("configs vides: running=%q startup=%q", running, startup)
	}
	lines := configdiff.Diff(startup, running)
	changed := false
	for _, l := range lines {
		if l.Op != "same" {
			changed = true
			break
		}
	}
	if !changed {
		t.Fatal("le simulateur doit produire un diff non vide entre running et startup")
	}
}

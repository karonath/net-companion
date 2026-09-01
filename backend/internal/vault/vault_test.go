package vault_test

import (
	"path/filepath"
	"testing"

	"netcompanion/internal/models"
	"netcompanion/internal/vault"
)

func tmpVault(t *testing.T) *vault.Vault {
	t.Helper()
	return vault.New(filepath.Join(t.TempDir(), "vault.dat"))
}

func TestInitThenUnlockLifecycle(t *testing.T) {
	v := tmpVault(t)
	if s := v.Status(); s.Initialized || s.Unlocked {
		t.Fatalf("état initial = %+v, want tout false", s)
	}
	if err := v.Init("1234"); err != nil {
		t.Fatalf("Init: %v", err)
	}
	if s := v.Status(); !s.Initialized || !s.Unlocked {
		t.Fatalf("après Init = %+v, want initialized+unlocked", s)
	}
	v.Lock()
	if s := v.Status(); !s.Initialized || s.Unlocked {
		t.Fatalf("après Lock = %+v, want initialized, verrouillé", s)
	}
	if err := v.Unlock("1234"); err != nil {
		t.Fatalf("Unlock bon PIN: %v", err)
	}
	if !v.Status().Unlocked {
		t.Fatal("devrait être déverrouillé")
	}
}

func TestUnlockWrongPIN(t *testing.T) {
	v := tmpVault(t)
	if err := v.Init("1234"); err != nil {
		t.Fatalf("Init: %v", err)
	}
	v.Lock()
	if err := v.Unlock("0000"); err != vault.ErrUnlockFailed {
		t.Fatalf("mauvais PIN err = %v, want ErrUnlockFailed", err)
	}
	if v.Status().Unlocked {
		t.Fatal("ne doit pas être déverrouillé après mauvais PIN")
	}
}

func TestPersistenceAcrossInstances(t *testing.T) {
	path := filepath.Join(t.TempDir(), "vault.dat")
	v1 := vault.New(path)
	if err := v1.Init("1234"); err != nil {
		t.Fatalf("Init: %v", err)
	}
	if _, err := v1.AddSNMP(models.SNMPCredential{Label: "prod", Community: "public", Version: "v2c"}); err != nil {
		t.Fatalf("AddSNMP: %v", err)
	}

	// Nouvelle instance lisant le même fichier.
	v2 := vault.New(path)
	if !v2.Status().Initialized {
		t.Fatal("v2 devrait voir le fichier initialisé")
	}
	if err := v2.Unlock("1234"); err != nil {
		t.Fatalf("v2 Unlock: %v", err)
	}
	snap, err := v2.Snapshot()
	if err != nil {
		t.Fatalf("Snapshot: %v", err)
	}
	if len(snap.SNMP) != 1 || snap.SNMP[0].Community != "public" {
		t.Fatalf("secrets non persistés: %+v", snap.SNMP)
	}
}

func TestMutationRequiresUnlock(t *testing.T) {
	v := tmpVault(t)
	if err := v.Init("1234"); err != nil {
		t.Fatalf("Init: %v", err)
	}
	v.Lock()
	if _, err := v.AddSNMP(models.SNMPCredential{Label: "x", Community: "c"}); err != vault.ErrLocked {
		t.Fatalf("AddSNMP verrouillé err = %v, want ErrLocked", err)
	}
}

func TestAddAssignsIDAndDelete(t *testing.T) {
	v := tmpVault(t)
	if err := v.Init("1234"); err != nil {
		t.Fatalf("Init: %v", err)
	}
	c, err := v.AddSSH(models.SSHCredential{Label: "core", Username: "admin", Password: "s3cr3t"})
	if err != nil {
		t.Fatalf("AddSSH: %v", err)
	}
	if c.ID == "" {
		t.Fatal("AddSSH doit assigner un ID non vide")
	}
	if err := v.DeleteSSH(c.ID); err != nil {
		t.Fatalf("DeleteSSH: %v", err)
	}
	snap, _ := v.Snapshot()
	if len(snap.SSH) != 0 {
		t.Fatalf("SSH non supprimé: %+v", snap.SSH)
	}
}

func TestInitTwiceFails(t *testing.T) {
	v := tmpVault(t)
	if err := v.Init("1234"); err != nil {
		t.Fatalf("Init: %v", err)
	}
	if err := v.Init("1234"); err != vault.ErrAlreadyInitialized {
		t.Fatalf("2e Init err = %v, want ErrAlreadyInitialized", err)
	}
}

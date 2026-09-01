package vault

import (
	"bytes"
	"testing"
)

func TestSealOpenRoundtrip(t *testing.T) {
	key := deriveKey("1234", []byte("0123456789abcdef"))
	if len(key) != 32 {
		t.Fatalf("clé = %d o, want 32", len(key))
	}
	plaintext := []byte(`{"snmp":[],"ssh":[]}`)

	nonce, ct, err := seal(key, plaintext)
	if err != nil {
		t.Fatalf("seal: %v", err)
	}
	got, err := open(key, nonce, ct)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if !bytes.Equal(got, plaintext) {
		t.Fatalf("roundtrip = %q, want %q", got, plaintext)
	}
}

func TestOpenWrongKeyFails(t *testing.T) {
	salt := []byte("0123456789abcdef")
	nonce, ct, err := seal(deriveKey("1234", salt), []byte("secret"))
	if err != nil {
		t.Fatalf("seal: %v", err)
	}
	if _, err := open(deriveKey("9999", salt), nonce, ct); err == nil {
		t.Fatal("open avec mauvaise clé devrait échouer")
	}
}

func TestDeriveKeyDeterministic(t *testing.T) {
	salt := []byte("0123456789abcdef")
	a := deriveKey("1234", salt)
	b := deriveKey("1234", salt)
	if !bytes.Equal(a, b) {
		t.Fatal("deriveKey doit être déterministe pour (pin, salt) identiques")
	}
}

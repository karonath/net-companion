package portfinder

import (
	"testing"

	"github.com/gosnmp/gosnmp"

	"netcompanion/internal/models"
)

func TestBuildGoSNMPv2c(t *testing.T) {
	g, err := buildGoSNMP("192.168.1.1:161", models.SNMPCredential{
		Version: "v2c", Community: "public",
	})
	if err != nil {
		t.Fatalf("buildGoSNMP v2c: %v", err)
	}
	if g.Version != gosnmp.Version2c || g.Community != "public" {
		t.Fatalf("v2c mal configuré: version=%v community=%q", g.Version, g.Community)
	}
	if g.Port != 161 || g.Target != "192.168.1.1" {
		t.Fatalf("target/port = %s:%d", g.Target, g.Port)
	}
}

func TestBuildGoSNMPv3AuthPriv(t *testing.T) {
	g, err := buildGoSNMP("10.0.0.1", models.SNMPCredential{
		Version:        "v3",
		SecurityName:   "netops",
		SecurityLevel:  "authPriv",
		AuthProtocol:   "SHA",
		AuthPassphrase: "authpass123",
		PrivProtocol:   "AES",
		PrivPassphrase: "privpass123",
	})
	if err != nil {
		t.Fatalf("buildGoSNMP v3: %v", err)
	}
	if g.Version != gosnmp.Version3 {
		t.Fatalf("version = %v, want v3", g.Version)
	}
	if g.MsgFlags != gosnmp.AuthPriv {
		t.Fatalf("msgFlags = %v, want AuthPriv", g.MsgFlags)
	}
	usm, ok := g.SecurityParameters.(*gosnmp.UsmSecurityParameters)
	if !ok {
		t.Fatalf("SecurityParameters type = %T", g.SecurityParameters)
	}
	if usm.UserName != "netops" {
		t.Fatalf("userName = %q", usm.UserName)
	}
	if usm.AuthenticationProtocol != gosnmp.SHA {
		t.Fatalf("auth proto = %v, want SHA", usm.AuthenticationProtocol)
	}
	if usm.PrivacyProtocol != gosnmp.AES {
		t.Fatalf("priv proto = %v, want AES", usm.PrivacyProtocol)
	}
	if usm.AuthenticationPassphrase != "authpass123" || usm.PrivacyPassphrase != "privpass123" {
		t.Fatalf("passphrases mal transmises")
	}
}

func TestBuildGoSNMPv3AuthNoPriv(t *testing.T) {
	g, err := buildGoSNMP("10.0.0.1", models.SNMPCredential{
		Version: "v3", SecurityName: "u", SecurityLevel: "authNoPriv",
		AuthProtocol: "MD5", AuthPassphrase: "x",
	})
	if err != nil {
		t.Fatalf("v3 authNoPriv: %v", err)
	}
	if g.MsgFlags != gosnmp.AuthNoPriv {
		t.Fatalf("msgFlags = %v, want AuthNoPriv", g.MsgFlags)
	}
	usm := g.SecurityParameters.(*gosnmp.UsmSecurityParameters)
	if usm.AuthenticationProtocol != gosnmp.MD5 {
		t.Fatalf("auth = %v, want MD5", usm.AuthenticationProtocol)
	}
	if usm.PrivacyProtocol != gosnmp.NoPriv {
		t.Fatalf("priv = %v, want NoPriv", usm.PrivacyProtocol)
	}
}

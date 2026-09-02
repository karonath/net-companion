package portfinder

import (
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/gosnmp/gosnmp"

	"netcompanion/internal/models"
)

// goSNMP adapte gosnmp à l'interface SNMPClient.
type goSNMP struct {
	conn *gosnmp.GoSNMP
}

// splitTargetPort sépare "host[:port]" ; port par défaut = 161.
func splitTargetPort(target string) (host string, port uint16) {
	if h, p, err := net.SplitHostPort(target); err == nil {
		if n, err := strconv.Atoi(p); err == nil && n > 0 && n <= 65535 {
			return h, uint16(n)
		}
		return h, 161
	}
	return target, 161
}

func authProto(name string) gosnmp.SnmpV3AuthProtocol {
	switch strings.ToUpper(name) {
	case "MD5":
		return gosnmp.MD5
	case "SHA":
		return gosnmp.SHA
	case "SHA224":
		return gosnmp.SHA224
	case "SHA256":
		return gosnmp.SHA256
	case "SHA384":
		return gosnmp.SHA384
	case "SHA512":
		return gosnmp.SHA512
	default:
		return gosnmp.NoAuth
	}
}

func privProto(name string) gosnmp.SnmpV3PrivProtocol {
	switch strings.ToUpper(name) {
	case "DES":
		return gosnmp.DES
	case "AES":
		return gosnmp.AES
	case "AES192":
		return gosnmp.AES192
	case "AES256":
		return gosnmp.AES256
	default:
		return gosnmp.NoPriv
	}
}

func msgFlags(level string) gosnmp.SnmpV3MsgFlags {
	switch level {
	case "authPriv":
		return gosnmp.AuthPriv
	case "authNoPriv":
		return gosnmp.AuthNoPriv
	default:
		return gosnmp.NoAuthNoPriv
	}
}

// buildGoSNMP construit une session gosnmp (non connectée) v2c ou v3 selon cred.
func buildGoSNMP(target string, cred models.SNMPCredential) (*gosnmp.GoSNMP, error) {
	host, port := splitTargetPort(target)
	g := &gosnmp.GoSNMP{
		Target:  host,
		Port:    port,
		Timeout: 2 * time.Second,
		Retries: 1,
	}
	if cred.Version == "v3" {
		g.Version = gosnmp.Version3
		g.SecurityModel = gosnmp.UserSecurityModel
		g.MsgFlags = msgFlags(cred.SecurityLevel)
		g.SecurityParameters = &gosnmp.UsmSecurityParameters{
			UserName:                 cred.SecurityName,
			AuthenticationProtocol:   authProto(cred.AuthProtocol),
			AuthenticationPassphrase: cred.AuthPassphrase,
			PrivacyProtocol:          privProto(cred.PrivProtocol),
			PrivacyPassphrase:        cred.PrivPassphrase,
		}
		return g, nil
	}
	g.Version = gosnmp.Version2c
	g.Community = cred.Community
	return g, nil
}

// NewGoSNMP ouvre une session SNMP (v2c ou v3) ; le second retour la ferme.
// target accepte "ip" ou "ip:port".
func NewGoSNMP(target string, cred models.SNMPCredential) (SNMPClient, func() error, error) {
	conn, err := buildGoSNMP(target, cred)
	if err != nil {
		return nil, nil, err
	}
	if err := conn.Connect(); err != nil {
		return nil, nil, err
	}
	return &goSNMP{conn: conn}, conn.Conn.Close, nil
}

// NewGoSNMPFast ouvre une session SNMP à délai court et sans réessai, adaptée au
// sondage en masse (classification de nombreux hôtes) : les non-SNMP échouent vite.
func NewGoSNMPFast(target string, cred models.SNMPCredential, timeout time.Duration) (SNMPClient, func() error, error) {
	conn, err := buildGoSNMP(target, cred)
	if err != nil {
		return nil, nil, err
	}
	conn.Timeout = timeout
	conn.Retries = 0
	if err := conn.Connect(); err != nil {
		return nil, nil, err
	}
	return &goSNMP{conn: conn}, conn.Conn.Close, nil
}

func (g *goSNMP) Get(oid string) (string, bool) {
	res, err := g.conn.Get([]string{"." + oid})
	if err != nil || len(res.Variables) == 0 {
		return "", false
	}
	v := res.Variables[0]
	if v.Type == gosnmp.NoSuchObject || v.Type == gosnmp.NoSuchInstance || v.Value == nil {
		return "", false
	}
	return gosnmpValueToString(v), true
}

func (g *goSNMP) WalkStrings(root string) map[string]string {
	out := map[string]string{}
	_ = g.conn.Walk("."+root, func(v gosnmp.SnmpPDU) error {
		name := v.Name
		if len(name) > 0 && name[0] == '.' {
			name = name[1:]
		}
		out[name] = gosnmpValueToString(v)
		return nil
	})
	return out
}

func gosnmpValueToString(v gosnmp.SnmpPDU) string {
	switch val := v.Value.(type) {
	case []byte:
		return string(val)
	case string:
		return val
	case int:
		return strconv.Itoa(val)
	case uint:
		return strconv.FormatUint(uint64(val), 10)
	case int64:
		return strconv.FormatInt(val, 10)
	case uint64:
		return strconv.FormatUint(val, 10)
	default:
		return ""
	}
}

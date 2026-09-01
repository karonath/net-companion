package portfinder

import (
	"net"
	"strconv"
	"time"

	"github.com/gosnmp/gosnmp"
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

// NewGoSNMP ouvre une session SNMP v2c ; le second retour ferme la session.
// target accepte "ip" ou "ip:port".
func NewGoSNMP(target, community string) (SNMPClient, func() error, error) {
	host, port := splitTargetPort(target)
	conn := &gosnmp.GoSNMP{
		Target:    host,
		Port:      port,
		Community: community,
		Version:   gosnmp.Version2c,
		Timeout:   2 * time.Second,
		Retries:   1,
	}
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

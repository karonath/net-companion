// Package oui résout le fabricant d'une MAC via la base OUI IEEE embarquée.
package oui

import (
	"bufio"
	"compress/gzip"
	"embed"
	"strings"
	"sync"
)

//go:embed data/oui.tsv.gz
var data embed.FS

var (
	once  sync.Once
	table map[string]string
)

// prefix extrait les 6 premiers hex (OUI 24 bits) d'une MAC, en majuscules.
func prefix(mac string) string {
	var sb strings.Builder
	for _, r := range mac {
		switch {
		case r >= '0' && r <= '9', r >= 'A' && r <= 'F':
			sb.WriteRune(r)
		case r >= 'a' && r <= 'f':
			sb.WriteRune(r - 32)
		}
		if sb.Len() == 6 {
			break
		}
	}
	if sb.Len() < 6 {
		return ""
	}
	return sb.String()
}

// vendorFromTable résout la MAC contre une table fournie (testable).
func vendorFromTable(mac string, table map[string]string) string {
	p := prefix(mac)
	if p == "" {
		return ""
	}
	return table[p]
}

// load décompresse et parse la base embarquée une seule fois.
func load() map[string]string {
	m := map[string]string{}
	f, err := data.Open("data/oui.tsv.gz")
	if err != nil {
		return m
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		return m
	}
	defer gz.Close()
	sc := bufio.NewScanner(gz)
	sc.Buffer(make([]byte, 1024*1024), 1024*1024)
	for sc.Scan() {
		line := sc.Text()
		tab := strings.IndexByte(line, '\t')
		if tab <= 0 {
			continue
		}
		m[line[:tab]] = line[tab+1:]
	}
	return m
}

// Vendor renvoie le fabricant de la MAC ("" si inconnu).
func Vendor(mac string) string {
	once.Do(func() { table = load() })
	return vendorFromTable(mac, table)
}

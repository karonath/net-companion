package history

import (
	"bytes"
	"html/template"
	"sort"
	"strings"

	"netcompanion/internal/models"
)

var reportTmpl = template.Must(template.New("report").Funcs(template.FuncMap{
	"join": strings.Join,
}).Parse(`<!doctype html>
<html lang="fr">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>Rapport réseau — {{.Snap.ID}}</title>
<style>
  :root { --ink:#1a2230; --muted:#5b6472; --line:#e3e7ee; --bg:#fff; --accent:#0b63d6; }
  * { box-sizing: border-box; }
  body { font-family: -apple-system, "Segoe UI", Roboto, sans-serif; margin: 0; color: var(--ink); background: #f5f7fa; }
  .page { max-width: 960px; margin: 0 auto; background: var(--bg); padding: 2.2rem 2.4rem; }
  header.rep { display: flex; justify-content: space-between; align-items: flex-start; border-bottom: 3px solid var(--accent); padding-bottom: 1rem; }
  .brand { font-size: 1.5rem; font-weight: 700; }
  .brand span { color: var(--accent); }
  .doctype { color: var(--muted); font-size: 0.95rem; margin-top: 0.2rem; }
  .gen { text-align: right; color: var(--muted); font-size: 0.82rem; }
  .meta { display: grid; grid-template-columns: repeat(auto-fit, minmax(180px, 1fr)); gap: 0.6rem 1.4rem; margin: 1.2rem 0 0.4rem; }
  .meta div { font-size: 0.9rem; }
  .meta dt { color: var(--muted); font-size: 0.72rem; text-transform: uppercase; letter-spacing: 0.03em; }
  .meta dd { margin: 0.1rem 0 0; font-weight: 600; }
  .cards { display: flex; flex-wrap: wrap; gap: 0.8rem; margin: 1.4rem 0; }
  .card { flex: 1 1 130px; border: 1px solid var(--line); border-radius: 10px; padding: 0.8rem 1rem; }
  .card .n { font-size: 1.6rem; font-weight: 700; }
  .card .l { color: var(--muted); font-size: 0.8rem; }
  h2 { font-size: 1.05rem; margin: 1.8rem 0 0.6rem; padding-bottom: 0.3rem; border-bottom: 1px solid var(--line); }
  table { border-collapse: collapse; width: 100%; font-size: 0.85rem; }
  th, td { text-align: left; padding: 0.4rem 0.6rem; border-bottom: 1px solid var(--line); vertical-align: top; }
  th { color: var(--muted); font-size: 0.72rem; text-transform: uppercase; letter-spacing: 0.03em; }
  tr:nth-child(even) td { background: #fafbfd; }
  .pill { display: inline-block; padding: 0.05rem 0.5rem; border-radius: 999px; font-size: 0.75rem; font-weight: 600; }
  .ok { color: #0f7b3b; } .warn { color: #9a6700; } .fail { color: #cf222e; }
  .pill.ok { background: #e6f4ea; color: #0f7b3b; } .pill.warn { background: #fdf3d8; color: #9a6700; } .pill.fail { background: #fbe3e4; color: #cf222e; }
  .pill.sev-critical { background: #fbe3e4; color: #cf222e; } .pill.sev-warning { background: #fdf3d8; color: #9a6700; } .pill.sev-info { background: #e7edf5; color: #4b6584; }
  .health { display: flex; align-items: center; gap: 1.2rem; margin: 1.4rem 0; padding: 1rem 1.2rem; border: 1px solid var(--line); border-radius: 12px; border-left: 6px solid #8b949e; }
  .health .score { font-size: 2.4rem; font-weight: 800; line-height: 1; }
  .health .score span { font-size: 1rem; font-weight: 600; color: var(--muted); }
  .health .grade { font-weight: 700; }
  .health.g-A { border-left-color: #1a7f37; } .health.g-B { border-left-color: #4c8dff; }
  .health.g-C { border-left-color: #9a6700; } .health.g-D { border-left-color: #e16f24; } .health.g-E { border-left-color: #cf222e; }
  .added { color: #0f7b3b; } .removed { color: #cf222e; }
  .types span { display: inline-block; margin: 0 0.9rem 0.3rem 0; font-size: 0.85rem; }
  .types b { color: var(--accent); }
  footer.rep { margin-top: 2.2rem; padding-top: 0.8rem; border-top: 1px solid var(--line); color: var(--muted); font-size: 0.78rem; display: flex; justify-content: space-between; }
  .muted { color: var(--muted); }
  @media print { body { background: #fff; } .page { padding: 0; } }
</style>
</head>
<body>
<div class="page">
  <header class="rep">
    <div>
      <div class="brand">Net<span>-</span>Companion</div>
      <div class="doctype">Rapport d'intervention réseau</div>
    </div>
    <div class="gen">Généré le {{.TimeStr}}<br>Réf. {{.Snap.ID}}</div>
  </header>

  <dl class="meta">
    {{if .Snap.Label}}<div><dt>Site</dt><dd>{{.Snap.Label}}</dd></div>{{end}}
    <div><dt>Interface</dt><dd>{{.Snap.Interface.Name}} · {{.Snap.Interface.IPv4}}</dd></div>
    <div><dt>Sous-réseau</dt><dd>{{.Snap.Interface.CIDR}}</dd></div>
    <div><dt>Passerelle</dt><dd>{{.Snap.Gateway}}</dd></div>
    {{if .Snap.Notes}}<div style="grid-column:1/-1"><dt>Notes</dt><dd>{{.Snap.Notes}}</dd></div>{{end}}
  </dl>

  <div class="cards">
    <div class="card"><div class="n">{{len .Snap.Hosts}}</div><div class="l">Appareils découverts</div></div>
    <div class="card"><div class="n {{if eq .ChecksOK .ChecksTotal}}ok{{else}}warn{{end}}">{{.ChecksOK}}/{{.ChecksTotal}}</div><div class="l">Contrôles OK</div></div>
    <div class="card"><div class="n">{{len .Types}}</div><div class="l">Catégories d'appareils</div></div>
  </div>

  {{if .Snap.Health}}
  <div class="health g-{{.Snap.Health.Grade}}">
    <div class="score">{{.Snap.Health.Score}}<span>/100</span></div>
    <div class="hmeta">
      <div class="grade">Santé réseau — Note {{.Snap.Health.Grade}}</div>
      <div class="muted">{{.Snap.Health.Summary}}</div>
    </div>
  </div>
  {{if .Snap.Health.Issues}}
  <h2>Anomalies détectées</h2>
  <table>
    <tr><th>Gravité</th><th>Type</th><th>Détail</th></tr>
    {{range .Snap.Health.Issues}}<tr><td><span class="pill sev-{{.Severity}}">{{.Severity}}</span></td><td>{{.Title}}</td><td>{{.Detail}}</td></tr>{{end}}
  </table>
  {{end}}
  {{end}}

  {{if .Types}}
  <h2>Synthèse du parc</h2>
  <div class="types">
    {{range .Types}}<span>{{.Type}} <b>{{.Count}}</b></span>{{end}}
  </div>
  {{end}}

  {{if .HasChanges}}
  <h2>Changements depuis le passage précédent</h2>
  <ul>
    {{range .Changes.HostsAdded}}<li class="added">+ Apparu : {{.IP}} {{if .Name}}({{.Name}}){{end}} {{.Vendor}}</li>{{end}}
    {{range .Changes.HostsRemoved}}<li class="removed">− Disparu : {{.IP}} {{if .Name}}({{.Name}}){{end}} {{.Vendor}}</li>{{end}}
    {{if .Changes.GatewayTo}}<li>Passerelle : {{.Changes.GatewayFrom}} → {{.Changes.GatewayTo}}</li>{{end}}
    {{range .Changes.ChecksChanged}}<li>{{.Name}} : {{.From}} → {{.To}}</li>{{end}}
  </ul>
  {{end}}

  <h2>Diagnostics de connectivité</h2>
  <table>
    <tr><th>Contrôle</th><th>Statut</th><th>Détail</th></tr>
    {{range .Snap.Diag}}<tr><td>{{.Name}}</td><td><span class="pill {{.Status}}">{{.Status}}</span></td><td>{{.Detail}}</td></tr>{{end}}
  </table>

  <h2>Inventaire des appareils ({{len .Snap.Hosts}})</h2>
  <table>
    <tr><th>IP</th><th>Nom</th><th>Type</th><th>Modèle</th><th>Constructeur</th><th>MAC</th><th>Services</th><th>Vu par</th></tr>
    {{range .Snap.Hosts}}<tr>
      <td>{{.IP}}</td>
      <td>{{if .Name}}{{.Name}}{{else}}{{.Hostname}}{{end}}</td>
      <td>{{if .DeviceType}}{{.DeviceType}}{{else}}<span class="muted">non classé</span>{{end}}</td>
      <td>{{.Model}}</td>
      <td>{{if .Manufacturer}}{{.Manufacturer}}{{else}}{{.Vendor}}{{end}}</td>
      <td>{{.MAC}}</td>
      <td>{{join .Services ", "}}</td>
      <td>{{if .Sources}}{{join .Sources ", "}}{{else}}{{.Source}}{{end}}</td>
    </tr>{{end}}
  </table>

  <footer class="rep">
    <span>Net-Companion — diagnostic réseau de terrain</span>
    <span>{{.TimeStr}}</span>
  </footer>
</div>
</body>
</html>`))

type typeCount struct {
	Type  string
	Count int
}

// summarizeTypes compte les appareils par catégorie (triés par effectif décroissant).
func summarizeTypes(hosts []models.Host) []typeCount {
	counts := map[string]int{}
	for _, h := range hosts {
		t := h.DeviceType
		if t == "" {
			t = "non classé"
		}
		counts[t]++
	}
	out := make([]typeCount, 0, len(counts))
	for t, c := range counts {
		out = append(out, typeCount{Type: t, Count: c})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Count != out[j].Count {
			return out[i].Count > out[j].Count
		}
		return out[i].Type < out[j].Type
	})
	return out
}

// RenderHTML produit un rapport HTML autonome, professionnel et imprimable.
func RenderHTML(snap Snapshot, ch *Changes) string {
	hasChanges := ch != nil && (len(ch.HostsAdded) > 0 || len(ch.HostsRemoved) > 0 ||
		ch.GatewayTo != "" || len(ch.ChecksChanged) > 0)
	okChecks := 0
	for _, c := range snap.Diag {
		if c.Status == "ok" {
			okChecks++
		}
	}
	data := struct {
		Snap        Snapshot
		Changes     *Changes
		HasChanges  bool
		TimeStr     string
		Types       []typeCount
		ChecksOK    int
		ChecksTotal int
	}{
		Snap:        snap,
		Changes:     ch,
		HasChanges:  hasChanges,
		TimeStr:     snap.Timestamp.Format("02/01/2006 à 15:04"),
		Types:       summarizeTypes(snap.Hosts),
		ChecksOK:    okChecks,
		ChecksTotal: len(snap.Diag),
	}
	var buf bytes.Buffer
	_ = reportTmpl.Execute(&buf, data)
	return buf.String()
}

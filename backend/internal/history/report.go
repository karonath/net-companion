package history

import (
	"bytes"
	"html/template"
)

var reportTmpl = template.Must(template.New("report").Parse(`<!doctype html>
<html lang="fr">
<head>
<meta charset="utf-8">
<title>Rapport d'intervention — {{.Snap.ID}}</title>
<style>
  body { font-family: system-ui, "Segoe UI", sans-serif; margin: 2rem; color: #111; background: #fff; }
  h1 { font-size: 1.4rem; margin: 0 0 0.2rem; }
  .sub { color: #666; margin: 0 0 1.5rem; }
  h2 { font-size: 1.05rem; border-bottom: 1px solid #ddd; padding-bottom: 0.3rem; margin-top: 1.6rem; }
  table { border-collapse: collapse; width: 100%; font-size: 0.9rem; }
  th, td { text-align: left; padding: 0.35rem 0.6rem; border-bottom: 1px solid #eee; }
  th { color: #555; }
  .ok { color: #1a7f37; } .warn { color: #9a6700; } .fail { color: #cf222e; }
  .added { color: #1a7f37; } .removed { color: #cf222e; }
  ul { margin: 0.3rem 0; }
  @media print { body { margin: 0.5cm; } }
</style>
</head>
<body>
  <h1>Net-Companion — Rapport d'intervention</h1>
  <p class="sub">{{.TimeStr}} · Interface {{.Snap.Interface.Name}} ({{.Snap.Interface.IPv4}}) · Passerelle {{.Snap.Gateway}}</p>

  {{if .HasChanges}}
  <h2>Changements depuis le passage précédent</h2>
  <ul>
    {{range .Changes.HostsAdded}}<li class="added">+ hôte apparu : {{.IP}} {{.Vendor}}</li>{{end}}
    {{range .Changes.HostsRemoved}}<li class="removed">− hôte disparu : {{.IP}} {{.Vendor}}</li>{{end}}
    {{if .Changes.GatewayTo}}<li>Passerelle : {{.Changes.GatewayFrom}} → {{.Changes.GatewayTo}}</li>{{end}}
    {{range .Changes.ChecksChanged}}<li>{{.Name}} : {{.From}} → {{.To}}</li>{{end}}
  </ul>
  {{end}}

  <h2>Diagnostics de connectivité</h2>
  <table>
    <tr><th>Contrôle</th><th>Statut</th><th>Détail</th></tr>
    {{range .Snap.Diag}}<tr><td>{{.Name}}</td><td class="{{.Status}}">{{.Status}}</td><td>{{.Detail}}</td></tr>{{end}}
  </table>

  <h2>Hôtes découverts ({{len .Snap.Hosts}})</h2>
  <table>
    <tr><th>IP</th><th>MAC</th><th>Fabricant</th></tr>
    {{range .Snap.Hosts}}<tr><td>{{.IP}}</td><td>{{.MAC}}</td><td>{{.Vendor}}</td></tr>{{end}}
  </table>
</body>
</html>`))

// RenderHTML produit un rapport HTML autonome et imprimable pour un snapshot.
func RenderHTML(snap Snapshot, ch *Changes) string {
	hasChanges := ch != nil && (len(ch.HostsAdded) > 0 || len(ch.HostsRemoved) > 0 ||
		ch.GatewayTo != "" || len(ch.ChecksChanged) > 0)
	data := struct {
		Snap       Snapshot
		Changes    *Changes
		HasChanges bool
		TimeStr    string
	}{
		Snap:       snap,
		Changes:    ch,
		HasChanges: hasChanges,
		TimeStr:    snap.Timestamp.Format("02/01/2006 15:04:05"),
	}
	var buf bytes.Buffer
	_ = reportTmpl.Execute(&buf, data)
	return buf.String()
}

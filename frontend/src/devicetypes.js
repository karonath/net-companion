// Taxonomie partagée des types d'appareils : icône, forme et couleur.
// Utilisée par le graphe (TopologyGraph) et l'arborescence (TopologyTree).

export const TYPE_ICONS = {
  'routeur / box': '🌐',
  switch: '🔀',
  'pare-feu': '🛡️',
  "point d'accès": '📡',
  serveur: '🖥️',
  hyperviseur: '⚙️',
  'NAS / stockage': '🗄️',
  ordinateur: '💻',
  'smartphone / tablette': '📱',
  imprimante: '🖨️',
  scanner: '🖨️',
  'TV / média': '📺',
  'téléphone VoIP': '☎️',
  'console de jeu': '🎮',
  'électroménager': '🧺',
  'caméra': '📷',
  onduleur: '🔋',
  'automate / OT': '🏭',
  'objet connecté': '💡',
  Autre: '🔌',
}

// Forme + couleur : infra en carré/losange/triangle, terminaux en point.
export const TYPE_STYLE = {
  'routeur / box': { shape: 'star', color: '#f0883e' },
  'pare-feu': { shape: 'diamond', color: '#f85149' },
  switch: { shape: 'square', color: '#4c8dff' },
  "point d'accès": { shape: 'triangle', color: '#2bb6a3' },
  serveur: { shape: 'square', color: '#a970ff' },
  hyperviseur: { shape: 'square', color: '#7c5cff' },
  'NAS / stockage': { shape: 'square', color: '#c084fc' },
  imprimante: { shape: 'square', color: '#d29922' },
  scanner: { shape: 'square', color: '#d29922' },
  'téléphone VoIP': { shape: 'triangle', color: '#58a6ff' },
  'caméra': { shape: 'triangle', color: '#f0883e' },
  onduleur: { shape: 'square', color: '#bf8700' },
  'automate / OT': { shape: 'diamond', color: '#f85149' },
  ordinateur: { shape: 'dot', color: '#3fb950' },
  'smartphone / tablette': { shape: 'dot', color: '#56d4dd' },
  'TV / média': { shape: 'dot', color: '#db61a2' },
  'console de jeu': { shape: 'dot', color: '#db61a2' },
  'électroménager': { shape: 'dot', color: '#8b949e' },
  'objet connecté': { shape: 'dot', color: '#3fb950' },
  Autre: { shape: 'dot', color: '#8b949e' },
}

export function typeIcon(t) {
  return TYPE_ICONS[t] || '🔌'
}

export function styleFor(cat) {
  return TYPE_STYLE[cat] || TYPE_STYLE.Autre
}

// categoryOf : catégorie affichée d'un hôte (la passerelle est toujours « box »).
export function categoryOf(host, gatewayIp) {
  return host.ip === gatewayIp ? 'routeur / box' : host.deviceType || 'Autre'
}

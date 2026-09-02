<script setup>
import { ref, computed, watch, onMounted, onBeforeUnmount } from 'vue'
import { Network, DataSet } from 'vis-network/standalone'
import { api } from '../api'
import { friendlyError } from '../errors'
import { selectHost, state } from '../state'
import { typeIcon, styleFor, categoryOf } from '../devicetypes'
import TopologyTree from './TopologyTree.vue'

const el = ref(null)
const busy = ref(false)
const nbBusy = ref(false)
const err = ref('')
const phase = ref('')
const count = ref(0)
const ifaces = ref([])
const autoName = ref('')
const filter = ref('')
const activeType = ref('') // filtre de catégorie via la légende ('' = toutes)
const layoutMode = ref('tree') // 'star' | 'tree' (graphe) — hiérarchie lisible par défaut
const autoRefresh = ref(false)
const hostsRef = ref([]) // hôtes courants (pour l'arborescence)
const gwRef = ref('') // IP passerelle (pour l'arborescence)
const groupMode = ref('uplink') // 'uplink' | 'type' (arborescence)
const menuEl = ref(null)
const splitEl = ref(null)
const treeW = ref(300)
try {
  const v = Number(localStorage.getItem('nc.treeW'))
  if (v >= 200 && v <= 560) treeW.value = v
} catch {
  /* localStorage indisponible */
}

let network = null
let hostMap = {}
let gwHint = ''
let curNodes = null
let curEdges = null
let lastRes = null
let lastIPs = new Set()
let refreshTimer = null

async function loadInterfaces() {
  try {
    const res = await api.networkInterfaces()
    ifaces.value = res.interfaces || []
    autoName.value = res.auto || ''
  } catch {
    ifaces.value = []
  }
}

// Icône matériel rendue en image (emoji dessiné sur un canvas) pour les nœuds.
const emojiCache = {}
function emojiImg(emoji) {
  if (emojiCache[emoji]) return emojiCache[emoji]
  const size = 64
  const canvas = document.createElement('canvas')
  canvas.width = canvas.height = size
  const ctx = canvas.getContext('2d')
  ctx.font = '46px "Segoe UI Emoji","Apple Color Emoji","Noto Color Emoji",sans-serif'
  ctx.textAlign = 'center'
  ctx.textBaseline = 'middle'
  ctx.fillText(emoji, size / 2, size / 2 + 2)
  const url = canvas.toDataURL()
  emojiCache[emoji] = url
  return url
}
function catOf(h) {
  return categoryOf(h, gwHint)
}

// Légende dynamique (catégories présentes + compteur), cliquable pour filtrer.
const typeCounts = ref({})
const legendItems = computed(() =>
  Object.entries(typeCounts.value)
    .map(([label, n]) => ({ label, count: n, icon: typeIcon(label) }))
    .sort((a, b) => b.count - a.count || a.label.localeCompare(b.label))
)

function cssVar(n) {
  return getComputedStyle(document.documentElement).getPropertyValue(n).trim()
}

function styleOptions() {
  const base = {
    nodes: {
      size: 16, font: { color: cssVar('--text'), face: 'system-ui', size: 13 },
      borderWidth: 2, shapeProperties: { useBorderWithImage: true },
    },
    edges: { color: { color: cssVar('--border'), highlight: cssVar('--accent') }, width: 1 },
    interaction: { hover: true, tooltipDelay: 150 },
  }
  if (layoutMode.value === 'tree') {
    // Disposition hiérarchique déterministe : calme (aucune physique).
    base.layout = { hierarchical: { enabled: true, direction: 'UD', sortMethod: 'directed', levelSeparation: 110, nodeSpacing: 140, treeSpacing: 160 } }
    base.physics = false
    base.edges.smooth = { type: 'cubicBezier', forceDirection: 'vertical', roundness: 0.5 }
  } else {
    base.layout = { improvedLayout: false }
    base.physics = { stabilization: { enabled: true, iterations: 200, fit: true }, barnesHut: { springLength: 120 } }
    base.edges.smooth = { type: 'continuous' }
  }
  return base
}

// ---- Construction du graphe -------------------------------------------------
function buildGraphData(res) {
  const local = res.interface
  gwHint = local && local.ipv4 ? local.ipv4.replace(/\.\d+$/, '.1') : ''
  const hosts = res.hosts || []
  const ipset = new Set(hosts.map((h) => h.ip))
  const hubId = ipset.has(gwHint) ? gwHint : 'local'

  const nodes = [{
    id: 'local', label: 'Cette machine', title: `Cette machine<br>${local?.ipv4 || ''}`,
    shape: 'image', image: emojiImg('📍'),
    color: { background: cssVar('--accent'), border: cssVar('--accent') }, borderWidth: 3,
    size: hubId === 'local' ? 22 : 16, font: { color: cssVar('--text') },
  }]
  const edges = []
  const counts = {}
  for (const h of hosts) {
    const isGw = h.ip === gwHint
    const cat = catOf(h)
    counts[cat] = (counts[cat] || 0) + 1
    // Nœud épuré : icône matériel + nom court. Détails au survol et dans la fiche.
    const color = isGw ? cssVar('--orange') : styleFor(cat).color
    const title = h.name || h.hostname || ''
    const detail = h.model || h.manufacturer || h.vendor || h.mac || ''
    const tip = [title, h.ip, cat, detail].filter(Boolean).join('<br>')
    nodes.push({
      id: h.ip, label: title || h.ip, title: tip,
      shape: 'image', image: emojiImg(typeIcon(cat)),
      color: { background: color, border: color },
      borderWidth: isGw ? 4 : 3, size: isGw ? 26 : 18, font: { color: cssVar('--text') },
    })
  }
  // Arêtes : étoile (tout sur le hub) ou hiérarchie (via uplink).
  for (const h of hosts) {
    if (h.ip === hubId) continue
    let parent = hubId
    if (layoutMode.value === 'tree' && h.uplink && ipset.has(h.uplink) && h.uplink !== h.ip) {
      parent = h.uplink
    }
    edges.push({ from: parent, to: h.ip })
  }
  if (hubId !== 'local') edges.push({ from: hubId, to: 'local' })
  return { nodes, edges, counts }
}

function onNodeClick(params) {
  if (!params.nodes.length) return
  const id = params.nodes[0]
  if (id === 'local') return
  const h = hostMap[id]
  if (h) selectHost({ ...h, isGateway: id === gwHint })
}

function applyTopology(res, { highlightNew = false, fit = true } = {}) {
  lastRes = res
  hostMap = {}
  for (const h of res.hosts || []) hostMap[h.ip] = h
  const { nodes, edges, counts } = buildGraphData(res)
  typeCounts.value = counts
  count.value = (res.hosts || []).length
  hostsRef.value = res.hosts || [] // pour l'arborescence
  gwRef.value = gwHint

  if (!network) {
    curNodes = new DataSet(nodes)
    curEdges = new DataSet(edges)
    network = new Network(el.value, { nodes: curNodes, edges: curEdges }, styleOptions())
    network.on('stabilizationIterationsDone', () => network && network.fit())
    el.value.__ncNetwork = network // affordance E2E
    network.on('click', onNodeClick)
  } else {
    const desired = new Set(nodes.map((n) => n.id))
    curNodes.getIds().forEach((id) => {
      if (!desired.has(id)) curNodes.remove(id)
    })
    const added = []
    nodes.forEach((n) => {
      if (curNodes.get(n.id)) curNodes.update(n)
      else {
        curNodes.add(n)
        if (highlightNew && lastIPs.size && n.id !== 'local' && !lastIPs.has(n.id)) added.push(n.id)
      }
    })
    curEdges.clear()
    curEdges.add(edges)
    if (added.length) flashNew(added)
  }
  applyFilter()
  if (fit && network) network.fit()
}

// Met en évidence les hôtes nouvellement apparus (auto-rafraîchissement).
function flashNew(ids) {
  curNodes.update(ids.map((id) => ({
    id, borderWidth: 5, color: { border: '#39d353' },
    shadow: { enabled: true, color: '#39d353', size: 22 },
  })))
  setTimeout(() => {
    if (!curNodes) return
    curNodes.update(ids.map((id) => {
      const h = hostMap[id]
      if (!h) return { id }
      const color = id === gwHint ? cssVar('--orange') : styleFor(catOf(h)).color
      return { id, borderWidth: id === gwHint ? 4 : 3, color: { background: color, border: color }, shadow: { enabled: false } }
    }))
  }, 4000)
}

// ---- Filtres (texte + catégorie de légende) --------------------------------
function applyFilter() {
  if (!curNodes) return
  const term = filter.value.trim().toLowerCase()
  const updates = Object.values(hostMap).map((h) => {
    const hay = [h.ip, h.name, h.hostname, h.model, h.vendor, h.manufacturer, h.mac, h.deviceType]
      .filter(Boolean).join(' ').toLowerCase()
    const textOk = term === '' || hay.includes(term)
    const typeOk = activeType.value === '' || catOf(h) === activeType.value
    return { id: h.ip, hidden: !(textOk && typeOk) }
  })
  if (updates.length) curNodes.update(updates)
}
watch([filter, activeType], applyFilter)

function toggleType(label) {
  activeType.value = activeType.value === label ? '' : label
}

// ---- Disposition étoile / hiérarchie ---------------------------------------
function setLayout(mode) {
  if (layoutMode.value === mode) return
  layoutMode.value = mode
  if (!network) return
  if (mode === 'tree') {
    network.setOptions({
      layout: { hierarchical: { enabled: true, direction: 'UD', sortMethod: 'directed', levelSeparation: 110, nodeSpacing: 130 } },
      physics: false,
    })
  } else {
    network.setOptions({
      layout: { hierarchical: { enabled: false } },
      physics: { stabilization: { enabled: true, iterations: 200, fit: true }, barnesHut: { springLength: 120 } },
    })
  }
  if (lastRes) applyTopology(lastRes, {})
}

// ---- Scan progressif : vue ARP rapide puis identification complète ---------
async function scan() {
  busy.value = true
  err.value = ''
  try {
    phase.value = 'Balayage ARP…'
    const quick = await api.radar(state.selectedIface, { quick: true })
    applyTopology(quick, {})
    phase.value = 'Identification…'
    const full = await api.radar(state.selectedIface)
    applyTopology(full, { highlightNew: true })
    lastIPs = new Set((full.hosts || []).map((h) => h.ip))
  } catch (e) {
    err.value = friendlyError(e, 'Scan du réseau impossible.')
  } finally {
    busy.value = false
    phase.value = ''
  }
}

// Rafraîchissement silencieux (auto) : garde la caméra, met en évidence les nouveaux.
async function refreshSilent() {
  try {
    const full = await api.radar(state.selectedIface)
    applyTopology(full, { highlightNew: true, fit: false })
    lastIPs = new Set((full.hosts || []).map((h) => h.ip))
  } catch {
    /* silencieux */
  }
}

watch(autoRefresh, (on) => {
  clearInterval(refreshTimer)
  if (on) refreshTimer = setInterval(refreshSilent, 15000)
})

// Sélection depuis l'arborescence : ouvre la même fiche que le graphe.
function onTreeSelect(h) {
  selectHost({ ...h, isGateway: h.ip === gwHint })
}
// Synchronise la sélection graphe ↔ arbre ↔ fiche : quand l'hôte sélectionné
// change (clic dans l'arbre, ou fermeture), on met le graphe en cohérence.
watch(
  () => state.selectedHost && state.selectedHost.ip,
  (ip) => {
    if (!network) return
    if (ip && curNodes && curNodes.get(ip)) network.selectNodes([ip])
    else network.unselectAll()
  }
)

// Bascule du mode démo : reconstruire la topologie depuis le backend.
watch(() => state.sim.enabled, () => scan())

async function addNeighbors() {
  if (!curNodes || !curEdges) {
    err.value = "Lancez d'abord un scan (« Rescanner ») avant d'ajouter les voisins."
    return
  }
  nbBusy.value = true
  err.value = ''
  try {
    const res = await api.neighbors('', state.sim.enabled)
    const anchor = curNodes.get(gwHint) ? gwHint : 'local'
    for (const n of res.neighbors || []) {
      const name = n.remoteSysName || n.remoteChassisId || 'switch'
      const id = 'sw:' + name
      if (!curNodes.get(id)) {
        curNodes.add({
          id, label: name, shape: 'square',
          color: { background: '#4c8dff', border: '#4c8dff' }, font: { color: cssVar('--text') },
        })
      }
      const eid = 'nbedge:' + anchor + ':' + id
      if (!curEdges.get(eid)) {
        curEdges.add({
          id: eid, from: anchor, to: id,
          label: (n.localPort ? 'p' + n.localPort : '') + ' ↔ ' + (n.remotePortId || ''),
          font: { color: cssVar('--muted'), size: 10 },
        })
      }
    }
    if (network) network.fit()
  } catch (e) {
    err.value = e.status === 423 ? 'Coffre verrouillé (voisins SNMP).' : friendlyError(e, 'Voisins indisponibles.')
  } finally {
    nbBusy.value = false
  }
}

// ---- Export ----------------------------------------------------------------
function download(blob, name) {
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = name
  document.body.appendChild(a)
  a.click()
  a.remove()
  setTimeout(() => URL.revokeObjectURL(url), 60000)
}
function exportPNG() {
  const canvas = el.value && el.value.querySelector('canvas')
  if (!canvas) return
  canvas.toBlob((b) => b && download(b, 'radar-topologie.png'))
}
function exportCSV() {
  const hosts = (lastRes && lastRes.hosts) || []
  const cols = ['ip', 'name', 'hostname', 'deviceType', 'model', 'manufacturer', 'vendor', 'mac', 'services', 'uplink']
  const head = ['IP', 'Nom', 'Hote', 'Type', 'Modele', 'Constructeur', 'Fabricant OUI', 'MAC', 'Services', 'Uplink']
  const esc = (v) => {
    v = v == null ? '' : String(v)
    return /[",\n;]/.test(v) ? '"' + v.replace(/"/g, '""') + '"' : v
  }
  const rows = hosts.map((h) =>
    cols.map((k) => (k === 'services' ? esc((h.services || []).join(' ')) : esc(h[k]))).join(';')
  )
  const csv = [head.join(';'), ...rows].join('\r\n')
  download(new Blob(['﻿' + csv], { type: 'text/csv;charset=utf-8' }), 'inventaire.csv')
}

function onIfaceChange() {
  scan()
}

// Exécute une action du menu « ⋯ » puis referme le menu.
function runMenu(fn) {
  fn()
  if (menuEl.value) menuEl.value.open = false
}

// Séparateur redimensionnable entre l'arbre et le graphe.
function startDrag(e) {
  e.preventDefault()
  const move = (ev) => {
    if (!splitEl.value) return
    const rect = splitEl.value.getBoundingClientRect()
    treeW.value = Math.max(200, Math.min(560, ev.clientX - rect.left))
  }
  const up = () => {
    document.removeEventListener('mousemove', move)
    document.removeEventListener('mouseup', up)
    try {
      localStorage.setItem('nc.treeW', String(Math.round(treeW.value)))
    } catch {
      /* ignoré */
    }
    if (network) network.redraw()
  }
  document.addEventListener('mousemove', move)
  document.addEventListener('mouseup', up)
}

onMounted(() => {
  loadInterfaces()
  scan()
})
onBeforeUnmount(() => {
  clearInterval(refreshTimer)
  if (network) network.destroy()
})
</script>

<template>
  <div class="graph panel">
    <div class="bar">
      <div class="title">
        <strong title="Découvre les hôtes du sous-réseau (ARP + sondes), les identifie et les affiche. Clique un nœud pour ses détails.">Radar — topologie</strong>
        <span class="muted"> · {{ count }} hôte(s)</span>
        <span v-if="phase" class="phase">{{ phase }}</span>
      </div>
      <div class="controls">
        <input v-model="filter" class="filter" placeholder="🔎 Rechercher (nom, IP, type…)"
          aria-label="Rechercher un appareil" />
        <select v-model="state.selectedIface" class="iface" @change="onIfaceChange"
          aria-label="Interface réseau à scanner" title="Interface à scanner (défaut : auto)">
          <option value="">Auto{{ autoName ? ' (' + autoName + ')' : '' }}</option>
          <option v-for="i in ifaces" :key="i.name" :value="i.name">{{ i.name }} · {{ i.ipv4 }}</option>
        </select>
        <button class="primary" @click="scan" :disabled="busy">{{ busy ? 'Analyse…' : 'Analyser' }}</button>
        <details ref="menuEl" class="menu">
          <summary class="menubtn" title="Plus d'options" aria-label="Plus d'options">⋯</summary>
          <div class="menupop">
            <div class="menugrp">Disposition du graphe</div>
            <div class="seg" role="group">
              <button :class="{ on: layoutMode === 'tree' }" @click="setLayout('tree')">Hiérarchie</button>
              <button :class="{ on: layoutMode === 'star' }" @click="setLayout('star')">Étoile</button>
            </div>
            <label class="mitem"><input type="checkbox" v-model="autoRefresh" /> Rafraîchir auto (15 s)</label>
            <button class="mitem" @click="runMenu(addNeighbors)" :disabled="nbBusy">Découvrir les voisins (LLDP/CDP)</button>
            <div class="menugrp">Export</div>
            <button class="mitem" @click="runMenu(exportPNG)">Exporter la carte (PNG)</button>
            <button class="mitem" @click="runMenu(exportCSV)">Exporter l'inventaire (CSV)</button>
          </div>
        </details>
      </div>
    </div>
    <p v-if="err" class="err">{{ err }}</p>
    <div v-if="legendItems.length" class="legend">
      <button v-for="l in legendItems" :key="l.label" class="leg-item"
        :class="{ active: activeType === l.label }" @click="toggleType(l.label)"
        :title="activeType === l.label ? 'Afficher tout' : 'Filtrer : ' + l.label">
        {{ l.icon }} {{ l.label }} <b>{{ l.count }}</b>
      </button>
      <button v-if="activeType" class="leg-clear" @click="activeType = ''">✕ tout afficher</button>
    </div>
    <div ref="splitEl" class="split">
      <div class="treepane" :style="{ flexBasis: treeW + 'px', width: treeW + 'px' }">
        <div class="treehdr">
          <span>Inventaire <b>{{ count }}</b></span>
          <div class="seg sm" role="group" aria-label="Regroupement de l'inventaire">
            <button :class="{ on: groupMode === 'uplink' }" @click="groupMode = 'uplink'" title="Hiérarchie L2 (uplinks)">Hiérarchie</button>
            <button :class="{ on: groupMode === 'type' }" @click="groupMode = 'type'" title="Regrouper par type d'appareil">Par type</button>
          </div>
        </div>
        <TopologyTree :hosts="hostsRef" :gateway="gwRef" :filter="filter"
          :active-type="activeType" :group="groupMode" @select="onTreeSelect" />
      </div>
      <div class="divider" @mousedown="startDrag" title="Glisser pour redimensionner"></div>
      <div class="viewarea">
        <div class="canvas-wrap">
          <div ref="el" class="canvas"></div>
          <p v-if="!busy && !err && count === 0" class="empty">
            Aucun hôte détecté sur ce sous-réseau.<br />
            <span class="muted">Vérifiez l'interface, ou réveillez les appareils puis rescannez.</span>
          </p>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.graph { height: 100%; min-height: 0; display: flex; flex-direction: column; overflow: hidden; }
.bar {
  display: flex; align-items: center; justify-content: space-between; gap: 0.6rem;
  padding: 0.6rem 1rem; border-bottom: 1px solid var(--border); flex: 0 0 auto; flex-wrap: wrap;
}
.title { display: flex; align-items: baseline; gap: 0.4rem; }
.phase { color: var(--accent); font-size: 0.8rem; }
.controls { display: flex; align-items: center; gap: 0.4rem; flex-wrap: wrap; }
.filter { max-width: 150px; }
.iface { max-width: 170px; }
.seg { display: inline-flex; border: 1px solid var(--border); border-radius: 8px; overflow: hidden; }
.seg button { border: none; border-radius: 0; padding: 0.35rem 0.6rem; font-size: 0.82rem; background: transparent; color: var(--muted); }
.seg button.on { background: var(--accent); color: #04101f; }
.seg.sm button { padding: 0.15rem 0.4rem; font-size: 0.68rem; }
/* Menu « ⋯ » */
.menu { position: relative; }
.menu > summary { list-style: none; cursor: pointer; padding: 0.3rem 0.6rem; border: 1px solid var(--border); border-radius: 8px; color: var(--text); }
.menu > summary::-webkit-details-marker { display: none; }
.menupop {
  position: absolute; right: 0; top: calc(100% + 4px); z-index: 20; min-width: 230px;
  background: var(--panel); border: 1px solid var(--border); border-radius: 10px;
  box-shadow: 0 12px 30px rgba(0, 0, 0, 0.4); padding: 0.4rem; display: flex; flex-direction: column; gap: 0.2rem;
}
.menugrp { font-size: 0.68rem; text-transform: uppercase; letter-spacing: 0.03em; color: var(--muted); padding: 0.35rem 0.4rem 0.1rem; }
.mitem {
  display: flex; align-items: center; gap: 0.4rem; text-align: left; width: 100%;
  border: none; background: transparent; color: var(--text); padding: 0.4rem 0.4rem; border-radius: 6px; font-size: 0.85rem; cursor: pointer;
}
.mitem:hover { background: var(--panel-2); }
.menupop .seg { margin: 0.1rem 0.4rem 0.3rem; }
.split { flex: 1; min-height: 0; display: flex; }
.treepane { min-height: 0; display: flex; flex-direction: column; flex-shrink: 0; }
.treehdr {
  flex: 0 0 auto; padding: 0.4rem 0.8rem; font-size: 0.72rem; text-transform: uppercase;
  letter-spacing: 0.03em; color: var(--muted); border-bottom: 1px solid var(--border);
  display: flex; align-items: center; justify-content: space-between; gap: 0.5rem;
}
.treehdr b { color: var(--text); }
.divider { flex: 0 0 6px; cursor: col-resize; background: var(--border); opacity: 0.5; }
.divider:hover { opacity: 1; background: var(--accent); }
.viewarea { position: relative; flex: 1; min-width: 0; min-height: 0; }
.canvas-wrap { position: absolute; inset: 0; }
.canvas { height: 100%; min-height: 0; }
@media (max-width: 900px) {
  .split { flex-direction: column; }
  .treepane { flex: 0 0 40% !important; width: auto !important; border-bottom: 1px solid var(--border); }
  .divider { display: none; }
}
.empty {
  position: absolute; top: 50%; left: 50%; transform: translate(-50%, -50%);
  text-align: center; color: var(--text); margin: 0; pointer-events: none;
}
.legend {
  display: flex; flex-wrap: wrap; gap: 0.3rem 0.4rem; padding: 0.4rem 1rem;
  border-bottom: 1px solid var(--border); flex: 0 0 auto;
}
.leg-item {
  border: 1px solid var(--border); background: transparent; color: var(--muted);
  border-radius: 999px; padding: 0.1rem 0.55rem; font-size: 0.72rem; cursor: pointer; white-space: nowrap;
}
.leg-item.active { border-color: var(--accent); color: var(--text); background: color-mix(in srgb, var(--accent) 15%, transparent); }
.leg-item b { color: var(--text); font-variant-numeric: tabular-nums; }
.leg-clear { border: none; background: transparent; color: var(--accent); font-size: 0.72rem; cursor: pointer; }
.err { color: var(--red); padding: 0.5rem 1rem; margin: 0; }
</style>

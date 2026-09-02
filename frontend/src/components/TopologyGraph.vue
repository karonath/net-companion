<script setup>
import { ref, watch, onMounted, onBeforeUnmount } from 'vue'
import { Network, DataSet } from 'vis-network/standalone'
import { api } from '../api'
import { selectHost, state } from '../state'

const el = ref(null)
const busy = ref(false)
const nbBusy = ref(false)
const err = ref('')
const count = ref(0)
const ifaces = ref([])
const autoName = ref('')

async function loadInterfaces() {
  try {
    const res = await api.networkInterfaces()
    ifaces.value = res.interfaces || []
    autoName.value = res.auto || ''
  } catch {
    ifaces.value = []
  }
}
let network = null
let hostMap = {}
let gwHint = ''
let curNodes = null
let curEdges = null
const filter = ref('')

// Désactivation du mode démo : retire les voisins simulés du graphe.
watch(
  () => state.sim.enabled,
  (on) => {
    if (on || !curNodes || !curEdges) return
    const swIds = curNodes.getIds().filter((id) => String(id).startsWith('sw:'))
    const edgeIds = curEdges.getIds().filter((id) => String(id).startsWith('nbedge:'))
    if (edgeIds.length) curEdges.remove(edgeIds)
    if (swIds.length) curNodes.remove(swIds)
  }
)

watch(filter, (q) => {
  if (!curNodes) return
  const term = q.trim().toLowerCase()
  const updates = Object.values(hostMap).map((h) => {
    const hay = (h.ip + ' ' + (h.vendor || '') + ' ' + (h.mac || '')).toLowerCase()
    return { id: h.ip, hidden: term !== '' && !hay.includes(term) }
  })
  if (updates.length) curNodes.update(updates)
})

function styleOptions() {
  const css = getComputedStyle(document.documentElement)
  const c = (n) => css.getPropertyValue(n).trim()
  return {
    nodes: {
      shape: 'dot',
      size: 14,
      font: { color: c('--text'), face: 'system-ui', size: 13 },
      borderWidth: 2,
    },
    edges: {
      color: { color: c('--border'), highlight: c('--accent') },
      width: 1,
      smooth: { type: 'continuous' },
    },
    layout: { improvedLayout: false },
    physics: {
      stabilization: { enabled: true, iterations: 200, fit: true },
      barnesHut: { springLength: 120 },
    },
    interaction: { hover: true, tooltipDelay: 120 },
  }
}

async function scan() {
  busy.value = true
  err.value = ''
  try {
    const res = await api.radar(state.selectedIface)
    const css = getComputedStyle(document.documentElement)
    const c = (n) => css.getPropertyValue(n).trim()

    const local = res.interface
    gwHint = local && local.ipv4 ? local.ipv4.replace(/\.\d+$/, '.1') : ''
    hostMap = {}

    const nodes = new DataSet()
    const edges = new DataSet()

    const localId = 'local'
    nodes.add({
      id: localId,
      label: `Cette machine\n${local?.ipv4 || ''}`,
      color: { background: c('--accent'), border: c('--accent') },
      size: 20,
    })

    for (const h of res.hosts) {
      const isGw = h.ip === gwHint
      hostMap[h.ip] = h
      const label = [h.ip, h.vendor || h.mac || ''].filter(Boolean).join('\n')
      nodes.add({
        id: h.ip,
        label,
        color: {
          background: isGw ? c('--orange') : c('--green'),
          border: isGw ? c('--orange') : c('--green'),
        },
        size: isGw ? 18 : 12,
      })
      edges.add({ from: localId, to: h.ip })
    }

    count.value = res.hosts.length
    curNodes = nodes
    curEdges = edges
    const data = { nodes, edges }
    if (network) {
      network.setData(data)
    } else {
      network = new Network(el.value, data, styleOptions())
      // Recadre la caméra sur les nœuds une fois la physique stabilisée
      // (improvedLayout est désactivé car il abandonne au-delà de ~100 nœuds).
      network.on('stabilizationIterationsDone', () => network && network.fit())
      // Affordance de test E2E : accès à l'instance pour cliquer aux coords exactes.
      el.value.__ncNetwork = network
      network.on('click', (params) => {
        if (!params.nodes.length) return
        const id = params.nodes[0]
        if (id === 'local') return
        const h = hostMap[id]
        if (h) {
          selectHost({ ip: h.ip, mac: h.mac || '', vendor: h.vendor || '', isGateway: id === gwHint })
        }
      })
    }
    network.fit()
  } catch (e) {
    err.value = e.message
  } finally {
    busy.value = false
  }
}

async function addNeighbors() {
  if (!curNodes || !curEdges) return
  nbBusy.value = true
  err.value = ''
  try {
    const res = await api.neighbors('', state.sim.enabled)
    const css = getComputedStyle(document.documentElement)
    const c = (n) => css.getPropertyValue(n).trim()
    const anchor = curNodes.get(gwHint) ? gwHint : 'local'
    for (const n of res.neighbors || []) {
      const name = n.remoteSysName || n.remoteChassisId || 'switch'
      const id = 'sw:' + name
      if (!curNodes.get(id)) {
        curNodes.add({
          id, label: name, shape: 'box',
          color: { background: c('--accent'), border: c('--accent') },
          font: { color: '#04101f' },
        })
      }
      const eid = 'nbedge:' + anchor + ':' + id
      if (!curEdges.get(eid)) {
        curEdges.add({
          id: eid, from: anchor, to: id,
          label: (n.localPort ? 'p' + n.localPort : '') + ' ↔ ' + (n.remotePortId || ''),
          font: { color: c('--muted'), size: 10 },
        })
      }
    }
    if (network) network.fit()
  } catch (e) {
    err.value = e.status === 423 ? 'Coffre verrouillé (voisins SNMP).' : 'Voisins indisponibles : ' + e.message
  } finally {
    nbBusy.value = false
  }
}

function onIfaceChange() {
  scan()
}

onMounted(() => {
  loadInterfaces()
  scan()
})
onBeforeUnmount(() => {
  if (network) network.destroy()
})
</script>

<template>
  <div class="graph panel">
    <div class="bar">
      <div>
        <strong title="Découvre les hôtes du sous-réseau (table ARP + sondes) et les affiche en graphe. Clique un nœud pour ses détails et des actions. « Voisins LLDP » ajoute les switches adjacents (SNMP).">Radar — topologie L2 ⓘ</strong>
        <span class="muted"> · {{ count }} hôte(s)</span>
      </div>
      <select v-model="state.selectedIface" class="iface" @change="onIfaceChange"
        title="Mode Universel : forcer l'interface à scanner (défaut : auto)">
        <option value="">Auto{{ autoName ? ' (' + autoName + ')' : '' }}</option>
        <option v-for="i in ifaces" :key="i.name" :value="i.name">{{ i.name }} · {{ i.ipv4 }}</option>
      </select>
      <input v-model="filter" class="filter" placeholder="Filtrer (IP/vendor)" />
      <div class="bar-btns">
        <button @click="addNeighbors" :disabled="nbBusy" title="Découvrir les voisins LLDP/CDP par SNMP">
          {{ nbBusy ? '…' : 'Voisins LLDP' }}
        </button>
        <button @click="scan" :disabled="busy">
          {{ busy ? 'Scan…' : 'Rescanner' }}
        </button>
      </div>
    </div>
    <p v-if="err" class="err">{{ err }}</p>
    <div ref="el" class="canvas"></div>
  </div>
</template>

<style scoped>
.graph {
  height: 100%;
  min-height: 0;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}
.bar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0.75rem 1rem;
  border-bottom: 1px solid var(--border);
  flex: 0 0 auto;
}
.canvas {
  flex: 1;
  min-height: 0;
}
.bar-btns { display: flex; gap: 0.5rem; }
.filter { max-width: 150px; }
.iface { max-width: 190px; }
.bar { gap: 0.5rem; flex-wrap: wrap; }
.err {
  color: var(--red);
  padding: 0.5rem 1rem;
  margin: 0;
}
</style>

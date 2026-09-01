<script setup>
import { ref, onMounted, onBeforeUnmount } from 'vue'
import { Network, DataSet } from 'vis-network/standalone'
import { api } from '../api'

const el = ref(null)
const busy = ref(false)
const err = ref('')
const count = ref(0)
let network = null

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
    const res = await api.radar()
    const css = getComputedStyle(document.documentElement)
    const c = (n) => css.getPropertyValue(n).trim()

    const local = res.interface
    const gwHint = local && local.ipv4 ? local.ipv4.replace(/\.\d+$/, '.1') : ''

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
    const data = { nodes, edges }
    if (network) {
      network.setData(data)
    } else {
      network = new Network(el.value, data, styleOptions())
      // Recadre la caméra sur les nœuds une fois la physique stabilisée
      // (improvedLayout est désactivé car il abandonne au-delà de ~100 nœuds).
      network.on('stabilizationIterationsDone', () => network && network.fit())
    }
    network.fit()
  } catch (e) {
    err.value = e.message
  } finally {
    busy.value = false
  }
}

onMounted(scan)
onBeforeUnmount(() => {
  if (network) network.destroy()
})
</script>

<template>
  <div class="graph panel">
    <div class="bar">
      <div>
        <strong>Radar — topologie L2</strong>
        <span class="muted"> · {{ count }} hôte(s)</span>
      </div>
      <button @click="scan" :disabled="busy">
        {{ busy ? 'Scan…' : 'Rescanner' }}
      </button>
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
.err {
  color: var(--red);
  padding: 0.5rem 1rem;
  margin: 0;
}
</style>

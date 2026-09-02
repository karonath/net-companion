<script setup>
import { ref, computed } from 'vue'
import { typeIcon, categoryOf } from '../devicetypes'

const props = defineProps({
  hosts: { type: Array, default: () => [] },
  gateway: { type: String, default: '' },
  filter: { type: String, default: '' },
  activeType: { type: String, default: '' },
})
const emit = defineEmits(['select'])

const collapsed = ref(new Set())
function toggle(ip) {
  const s = new Set(collapsed.value)
  s.has(ip) ? s.delete(ip) : s.add(ip)
  collapsed.value = s
}

function matches(h) {
  const term = props.filter.trim().toLowerCase()
  const cat = categoryOf(h, props.gateway)
  const hay = [h.ip, h.name, h.hostname, h.model, h.vendor, h.manufacturer, h.mac, h.deviceType]
    .filter(Boolean).join(' ').toLowerCase()
  const textOk = term === '' || hay.includes(term)
  const typeOk = !props.activeType || cat === props.activeType
  return textOk && typeOk
}

// Construit la hiérarchie via uplink ; les hôtes sans uplink connu sont rattachés
// à la passerelle. Calcule aussi le nombre de descendants et la visibilité.
const model = computed(() => {
  const hosts = props.hosts || []
  const byIp = {}
  hosts.forEach((h) => (byIp[h.ip] = h))
  const children = {}
  const roots = []
  hosts.forEach((h) => {
    let up = h.uplink && byIp[h.uplink] ? h.uplink : null
    if (!up && h.ip !== props.gateway && byIp[props.gateway]) up = props.gateway
    if (up) (children[up] = children[up] || []).push(h)
    else roots.push(h)
  })
  const sortFn = (a, b) => a.ip.localeCompare(b.ip, undefined, { numeric: true })
  roots.sort(sortFn)
  Object.values(children).forEach((l) => l.sort(sortFn))

  const count = {}
  const vis = new Set()
  const dfs = (h) => {
    let c = 0
    let anyVis = false
    for (const k of children[h.ip] || []) {
      const r = dfs(k)
      c += 1 + r.c
      if (r.v) anyVis = true
    }
    const v = matches(h) || anyVis
    if (v) vis.add(h.ip)
    count[h.ip] = c
    return { c, v }
  }
  roots.forEach(dfs)
  return { children, roots, count, vis }
})

// Aplatit l'arbre en respectant repli/visibilité (rendu en liste indentée).
const flat = computed(() => {
  const { children, roots, count, vis } = model.value
  const out = []
  const walk = (list, depth) => {
    for (const h of list) {
      if (!vis.has(h.ip)) continue
      const kids = (children[h.ip] || []).filter((k) => vis.has(k.ip))
      out.push({ host: h, depth, hasChildren: kids.length > 0, count: count[h.ip] })
      if (kids.length && !collapsed.value.has(h.ip)) walk(kids, depth + 1)
    }
  }
  walk(roots, 0)
  return out
})

function cat(h) {
  return categoryOf(h, props.gateway)
}
</script>

<template>
  <div class="tree">
    <div v-for="n in flat" :key="n.host.ip" class="row"
      :style="{ paddingLeft: n.depth * 18 + 8 + 'px' }"
      @click="emit('select', n.host)">
      <span v-if="n.hasChildren" class="caret" @click.stop="toggle(n.host.ip)">
        {{ collapsed.has(n.host.ip) ? '▸' : '▾' }}
      </span>
      <span v-else class="caret ph"></span>
      <span class="ico">{{ typeIcon(cat(n.host)) }}</span>
      <span class="nm">{{ n.host.name || n.host.hostname || n.host.ip }}</span>
      <span class="ip">{{ n.host.ip }}</span>
      <span class="badge">{{ cat(n.host) }}</span>
      <span v-if="n.hasChildren" class="cnt" :title="n.count + ' appareil(s) en aval'">{{ n.count }}</span>
    </div>
    <p v-if="!flat.length" class="empty">Aucun appareil à afficher.</p>
  </div>
</template>

<style scoped>
.tree { height: 100%; overflow: auto; padding: 0.3rem 0; font-size: 0.85rem; }
.row {
  display: flex; align-items: center; gap: 0.4rem; padding: 0.28rem 0.8rem;
  cursor: pointer; border-bottom: 1px solid transparent;
}
.row:hover { background: var(--panel-2); }
.caret { width: 1rem; text-align: center; color: var(--muted); flex: 0 0 auto; }
.caret.ph { visibility: hidden; }
.ico { flex: 0 0 auto; font-size: 1.05rem; }
.nm { font-weight: 600; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.ip { color: var(--muted); font-family: ui-monospace, monospace; font-size: 0.78rem; flex: 0 0 auto; }
.badge {
  margin-left: auto; flex: 0 0 auto; color: var(--muted); font-size: 0.72rem;
  border: 1px solid var(--border); border-radius: 999px; padding: 0 0.5rem;
}
.cnt {
  flex: 0 0 auto; background: var(--accent); color: #04101f; font-size: 0.7rem;
  border-radius: 999px; padding: 0 0.45rem; font-variant-numeric: tabular-nums;
}
.empty { color: var(--muted); text-align: center; padding: 2rem 1rem; }
</style>


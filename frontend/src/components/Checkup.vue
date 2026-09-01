<script setup>
import { ref, onMounted, computed } from 'vue'
import { api } from '../api'

const busy = ref(false)
const snap = ref(null)
const changes = ref(null)
const hist = ref([])
const err = ref('')
const label = ref('')
const notes = ref('')

const okChecks = computed(() =>
  snap.value ? snap.value.diag.filter((c) => c.status === 'ok').length : 0
)

async function loadHistory() {
  try {
    hist.value = await api.history()
  } catch {
    hist.value = []
  }
}

async function run() {
  busy.value = true
  err.value = ''
  try {
    const res = await api.checkup(label.value, notes.value)
    snap.value = res.snapshot
    changes.value = res.changes
    await loadHistory()
  } catch (e) {
    err.value = e.message
  } finally {
    busy.value = false
  }
}

function openReport(id) {
  window.open(api.reportUrl(id), '_blank')
}

function fmt(ts) {
  return new Date(ts).toLocaleString()
}

const hasChanges = computed(() => {
  const c = changes.value
  return c && (c.hostsAdded?.length || c.hostsRemoved?.length || c.gatewayTo || c.checksChanged?.length)
})

onMounted(loadHistory)
</script>

<template>
  <div class="checkup">
    <div class="meta">
      <input v-model="label" placeholder="Site (ex: Client X — Salle 2)" />
      <input v-model="notes" placeholder="Notes (optionnel)" />
    </div>
    <button class="primary big" @click="run" :disabled="busy">
      {{ busy ? 'Analyse du site…' : 'Lancer le check de site' }}
    </button>
    <p v-if="err" class="err">{{ err }}</p>

    <div v-if="snap" class="result">
      <div class="summary">
        <span class="tag muted">{{ new Date(snap.timestamp).toLocaleString() }}</span>
        <span class="tag"><span class="dot green"></span> {{ snap.hosts.length }} hôte(s)</span>
        <span class="tag">
          <span class="dot" :class="okChecks === snap.diag.length ? 'green' : 'orange'"></span>
          {{ okChecks }}/{{ snap.diag.length }} contrôles OK
        </span>
      </div>

      <div class="report-actions">
        <button class="primary" @click="openReport(snap.id)">Ouvrir le rapport</button>
        <a class="btn" :href="api.reportJsonUrl(snap.id)" :download="'net-companion-' + snap.id + '.json'">
          Télécharger (JSON)
        </a>
      </div>

      <div v-if="hasChanges" class="changes">
        <h4>Changements depuis le dernier passage</h4>
        <ul>
          <li v-for="h in changes.hostsAdded" :key="'a' + h.ip" class="added">+ {{ h.ip }} {{ h.vendor }}</li>
          <li v-for="h in changes.hostsRemoved" :key="'r' + h.ip" class="removed">− {{ h.ip }} {{ h.vendor }}</li>
          <li v-if="changes.gatewayTo">Passerelle : {{ changes.gatewayFrom }} → {{ changes.gatewayTo }}</li>
          <li v-for="c in changes.checksChanged" :key="c.name" class="warn">{{ c.name }} : {{ c.from }} → {{ c.to }}</li>
        </ul>
      </div>
      <p v-else-if="changes" class="muted">Aucun changement depuis le dernier passage.</p>
    </div>

    <div v-if="hist.length" class="history">
      <h4>Historique</h4>
      <ul>
        <li v-for="m in hist" :key="m.id">
          <button class="link" @click="openReport(m.id)">{{ fmt(m.timestamp) }}</button>
          <span class="muted">· {{ m.hostCount }} hôte(s)</span>
        </li>
      </ul>
    </div>
  </div>
</template>

<style scoped>
.meta { display: flex; flex-direction: column; gap: 0.5rem; margin-bottom: 0.6rem; }
.big { width: 100%; padding: 0.8rem; font-size: 1rem; }
.err { color: var(--red); }
.result { margin-top: 1rem; }
.summary { display: flex; gap: 0.6rem; flex-wrap: wrap; margin-bottom: 0.8rem; }
.report-actions { display: flex; gap: 0.5rem; margin-bottom: 1rem; }
.btn {
  display: inline-flex; align-items: center; padding: 0.5rem 0.9rem;
  border: 1px solid var(--border); border-radius: 8px; color: var(--text); text-decoration: none;
}
.btn:hover { border-color: var(--accent); }
h4 { margin: 1rem 0 0.5rem; font-size: 0.9rem; }
.changes ul, .history ul { list-style: none; padding: 0; margin: 0; display: flex; flex-direction: column; gap: 0.3rem; font-size: 0.9rem; }
.added { color: var(--green); }
.removed { color: var(--red); }
.warn { color: var(--orange); }
.link { border: none; background: transparent; color: var(--accent); padding: 0; cursor: pointer; }
</style>

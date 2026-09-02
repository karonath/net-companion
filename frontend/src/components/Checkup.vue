<script setup>
import { ref, onMounted, computed } from 'vue'
import { api } from '../api'
import { friendlyError } from '../errors'
import HelpNote from './HelpNote.vue'

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
  } catch (e) {
    hist.value = []
    err.value = friendlyError(e, "Chargement de l'historique impossible.")
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
    err.value = friendlyError(e, 'Check de site impossible.')
  } finally {
    busy.value = false
  }
}

async function openReport(id) {
  err.value = ''
  try {
    await api.openReport(id)
  } catch (e) {
    err.value = friendlyError(e, 'Ouverture du rapport impossible.')
  }
}

async function downloadJson(id) {
  err.value = ''
  try {
    await api.downloadReportJson(id)
  } catch (e) {
    err.value = friendlyError(e, 'Téléchargement du rapport impossible.')
  }
}

async function clearHistory() {
  if (!window.confirm("Vider tout l'historique des checks ? Cette action est irréversible.")) return
  err.value = ''
  try {
    await api.clearHistory()
    hist.value = []
    snap.value = null
    changes.value = null
  } catch (e) {
    err.value = friendlyError(e, "Vidage de l'historique impossible.")
  }
}

function fmt(ts) {
  return new Date(ts).toLocaleString()
}

function sevLabel(s) {
  return { critical: 'critique', warning: 'attention', info: 'info' }[s] || s
}

const hasChanges = computed(() => {
  const c = changes.value
  return c && (c.hostsAdded?.length || c.hostsRemoved?.length || c.gatewayTo || c.checksChanged?.length)
})

onMounted(loadHistory)
</script>

<template>
  <div class="checkup">
    <HelpNote>
      Lance en un clic un état des lieux du site : découverte des hôtes (radar)
      + diagnostics de connectivité, assemblés en quelques secondes. Chaque
      passage est horodaté et conservé ; tu peux comparer d'un passage à l'autre
      et exporter un rapport à joindre à un ticket. Idéal en arrivant sur site.
    </HelpNote>
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

      <div v-if="snap.health" class="health" :class="'g-' + snap.health.grade">
        <div class="score">{{ snap.health.score }}<span>/100</span></div>
        <div>
          <div class="grade">Santé réseau — Note {{ snap.health.grade }}</div>
          <div class="muted">{{ snap.health.summary }}</div>
        </div>
      </div>
      <ul v-if="snap.health && snap.health.issues && snap.health.issues.length" class="issues">
        <li v-for="(iss, i) in snap.health.issues" :key="i" :class="iss.severity">
          <span class="sev">{{ sevLabel(iss.severity) }}</span>
          <span><strong>{{ iss.title }}</strong> — {{ iss.detail }}</span>
        </li>
      </ul>

      <div class="report-actions">
        <button class="primary" @click="openReport(snap.id)" title="Ouvre le rapport dans un nouvel onglet">
          Ouvrir le rapport ↗
        </button>
        <button class="btn" @click="downloadJson(snap.id)">Télécharger (JSON)</button>
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
      <div class="hist-head">
        <h4>Historique</h4>
        <button class="link danger-link" @click="clearHistory">Vider l'historique</button>
      </div>
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
.health {
  display: flex; align-items: center; gap: 1rem; margin: 0 0 0.8rem;
  padding: 0.7rem 0.9rem; border: 1px solid var(--border); border-radius: 10px; border-left: 5px solid var(--muted);
}
.health .score { font-size: 1.9rem; font-weight: 800; line-height: 1; }
.health .score span { font-size: 0.8rem; font-weight: 600; color: var(--muted); }
.health .grade { font-weight: 700; }
.health.g-A { border-left-color: var(--green); }
.health.g-B { border-left-color: var(--accent); }
.health.g-C { border-left-color: var(--orange); }
.health.g-D { border-left-color: var(--orange); }
.health.g-E { border-left-color: var(--red); }
.issues { list-style: none; padding: 0; margin: 0 0 1rem; display: flex; flex-direction: column; gap: 0.35rem; font-size: 0.88rem; }
.issues li { display: flex; gap: 0.5rem; align-items: baseline; }
.issues .sev {
  flex: 0 0 auto; font-size: 0.68rem; text-transform: uppercase; letter-spacing: 0.03em;
  border-radius: 999px; padding: 0.05rem 0.5rem; border: 1px solid var(--border);
}
.issues li.critical .sev { color: var(--red); border-color: var(--red); }
.issues li.warning .sev { color: var(--orange); border-color: var(--orange); }
.issues li.info .sev { color: var(--muted); }
.report-actions { display: flex; gap: 0.5rem; margin-bottom: 1rem; }
.btn {
  display: inline-flex; align-items: center; padding: 0.5rem 0.9rem;
  border: 1px solid var(--border); border-radius: 8px; color: var(--text);
  text-decoration: none; background: transparent; font: inherit; cursor: pointer;
}
.btn:hover { border-color: var(--accent); }
h4 { margin: 1rem 0 0.5rem; font-size: 0.9rem; }
.changes ul, .history ul { list-style: none; padding: 0; margin: 0; display: flex; flex-direction: column; gap: 0.3rem; font-size: 0.9rem; }
.added { color: var(--green); }
.removed { color: var(--red); }
.warn { color: var(--orange); }
.link { border: none; background: transparent; color: var(--accent); padding: 0; cursor: pointer; }
.hist-head { display: flex; align-items: center; justify-content: space-between; }
.danger-link { color: var(--red); font-size: 0.82rem; }
</style>

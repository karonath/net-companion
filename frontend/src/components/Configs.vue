<script setup>
import { ref, onMounted } from 'vue'
import { api } from '../api'
import { state } from '../state'
import HelpNote from './HelpNote.vue'

const devicesText = ref('')
const busy = ref(false)
const err = ref('')
const results = ref([])
const devices = ref([])
const drift = ref(null) // { device, lines }

async function loadDevices() {
  try {
    devices.value = await api.configDevices()
  } catch {
    devices.value = []
  }
}

function useDemo() {
  if (state.sim.enabled) devicesText.value = state.sim.ssh
}

async function backup() {
  const list = devicesText.value.split(/[\n,;]+/).map((s) => s.trim()).filter(Boolean)
  if (!list.length) return
  busy.value = true
  err.value = ''
  results.value = []
  try {
    const res = await api.configBackup(list)
    results.value = res.results || []
    await loadDevices()
  } catch (e) {
    err.value = e.status === 423 ? 'Coffre verrouillé.' : e.message
  } finally {
    busy.value = false
  }
}

async function setBaseline(device) {
  const hist = await api.configHistory(device)
  if (!hist.length) return
  await api.configBaseline(device, hist[0].id)
  await loadDevices()
}

async function showDrift(device) {
  const d = await api.configDrift(device)
  drift.value = { device, lines: d.lines || [], hasBaseline: d.hasBaseline }
}

function fmt(ts) {
  return new Date(ts).toLocaleString()
}

onMounted(loadDevices)
</script>

<template>
  <div class="configs">
    <h3>Sauvegarde de configuration</h3>
    <HelpNote>
      Sauvegarde la running-config de plusieurs équipements par SSH et détecte
      la <strong>dérive</strong> : définis une « baseline » (config approuvée)
      puis, à chaque backup, l'outil montre ce qui a changé vs cette référence.
      Nécessite un credential SSH dans le coffre (ou le Mode démo).
    </HelpNote>

    <button v-if="state.sim.enabled" class="demo" @click="useDemo">Utiliser l'équipement de démo</button>
    <textarea v-model="devicesText" rows="3" placeholder="IP par ligne (ex: 192.168.1.1)"></textarea>
    <button class="primary" @click="backup" :disabled="busy">
      {{ busy ? 'Sauvegarde…' : 'Sauvegarder' }}
    </button>
    <p v-if="err" class="err">{{ err }}</p>

    <ul v-if="results.length" class="results">
      <li v-for="(r, i) in results" :key="i">
        <span class="dot" :class="r.ok ? 'green' : 'red'"></span>
        <span>{{ r.device }} — {{ r.ok ? ('OK, dérive ' + r.driftCount + ' ligne(s)') : r.error }}</span>
      </li>
    </ul>

    <div v-if="devices.length" class="devices">
      <h3>Équipements</h3>
      <ul>
        <li v-for="d in devices" :key="d.device">
          <div class="drow">
            <span>
              <strong>{{ d.device }}</strong>
              <span class="muted"> · {{ d.count }} backup(s) · {{ fmt(d.last) }}</span>
              <span v-if="d.hasBaseline" class="tag">baseline</span>
            </span>
            <span class="acts">
              <button class="sm" @click="setBaseline(d.device)">Baseline</button>
              <button class="sm" @click="showDrift(d.device)">Drift</button>
            </span>
          </div>
        </li>
      </ul>
    </div>

    <div v-if="drift" class="driftview">
      <h3>Dérive — {{ drift.device }}</h3>
      <p v-if="!drift.hasBaseline" class="muted">Aucune baseline définie pour cet équipement.</p>
      <p v-else-if="!drift.lines.some(l => l.op !== 'same')" class="muted">Conforme à la baseline (aucune dérive).</p>
      <pre v-else class="diff"><code
        v-for="(l, i) in drift.lines" :key="i" :class="l.op"
      >{{ l.op === 'add' ? '+ ' : l.op === 'del' ? '- ' : '  ' }}{{ l.text }}
</code></pre>
    </div>
  </div>
</template>

<style scoped>
h3 { margin: 1rem 0 0.4rem; font-size: 0.95rem; }
h3:first-child { margin-top: 0; }
p { margin: 0 0 0.7rem; }
.demo { width: 100%; margin-bottom: 0.5rem; font-size: 0.85rem; }
textarea { width: 100%; background: var(--bg); color: var(--text); border: 1px solid var(--border); border-radius: 8px; padding: 0.5rem; font: inherit; resize: vertical; }
.primary { width: 100%; margin-top: 0.5rem; }
.err { color: var(--red); }
.results, .devices ul { list-style: none; padding: 0; margin: 0.8rem 0 0; display: flex; flex-direction: column; gap: 0.5rem; }
.results li { display: flex; gap: 0.5rem; align-items: center; font-size: 0.9rem; }
.drow { display: flex; align-items: center; justify-content: space-between; gap: 0.5rem; }
.acts { display: flex; gap: 0.4rem; flex: 0 0 auto; }
button.sm { padding: 0.2rem 0.5rem; }
.diff { margin: 0; padding: 0.6rem; background: var(--bg); border: 1px solid var(--border); border-radius: 8px; overflow-x: auto; font-size: 0.82rem; }
.diff code { display: block; white-space: pre; }
.diff code.add { color: var(--green); }
.diff code.del { color: var(--red); }
.diff code.same { color: var(--muted); }
</style>

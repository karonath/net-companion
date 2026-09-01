<script setup>
import { ref, watch } from 'vue'
import { api } from '../api'
import { state, copyText } from '../state'

const checks = ref([])
const busy = ref(false)

// Pré-remplissage depuis le détail d'hôte (test de port + traceroute).
watch(
  () => state.prefill.diagHost,
  (h) => {
    if (h) {
      ph.value = h
      tt.value = h
    }
  }
)

async function run() {
  busy.value = true
  try {
    const res = await api.diag()
    checks.value = res.checks || []
  } catch (e) {
    checks.value = [{ name: 'Erreur', status: 'fail', detail: e.message }]
  } finally {
    busy.value = false
  }
}

// Test de port
const ph = ref('')
const pp = ref('')
const portResult = ref(null)
async function checkPort() {
  if (!ph.value || !pp.value) return
  portResult.value = null
  try {
    portResult.value = await api.diagPort(ph.value, Number(pp.value))
  } catch (e) {
    portResult.value = { status: 'fail', detail: e.message }
  }
}

// Traceroute
const tt = ref('')
const hops = ref([])
const trBusy = ref(false)
async function traceroute() {
  if (!tt.value) return
  trBusy.value = true
  hops.value = []
  try {
    const res = await api.diagTraceroute(tt.value)
    hops.value = res.hops || []
  } catch (e) {
    hops.value = [{ num: 0, host: 'erreur: ' + e.message, ms: '' }]
  } finally {
    trBusy.value = false
  }
}

function dotClass(status) {
  return status === 'ok' ? 'green' : status === 'warn' ? 'orange' : 'red'
}
</script>

<template>
  <div class="diag">
    <section>
      <h3>Diagnostics de connectivité</h3>
      <button class="primary" @click="run" :disabled="busy">
        {{ busy ? 'Analyse…' : 'Lancer les diagnostics' }}
      </button>
      <button v-if="checks.length" class="copy"
        @click="copyText(checks.map(c => c.name + ': ' + c.status + ' — ' + c.detail).join('\n'))">
        Copier
      </button>
      <ul v-if="checks.length" class="checks">
        <li v-for="(c, i) in checks" :key="i">
          <span class="dot" :class="dotClass(c.status)"></span>
          <div>
            <strong>{{ c.name }}</strong>
            <div class="muted">{{ c.detail }}</div>
          </div>
        </li>
      </ul>
    </section>

    <section>
      <h3>Test de port</h3>
      <div class="form">
        <input v-model="ph" placeholder="Hôte (ex: 192.168.1.1)" />
        <input v-model="pp" placeholder="Port" style="max-width: 90px" />
        <button @click="checkPort">Tester</button>
      </div>
      <div v-if="portResult" class="line">
        <span class="dot" :class="dotClass(portResult.status)"></span>
        {{ portResult.detail }}
      </div>
    </section>

    <section>
      <h3>Traceroute</h3>
      <div class="form">
        <input v-model="tt" placeholder="Cible (ex: 1.1.1.1)" @keyup.enter="traceroute" />
        <button @click="traceroute" :disabled="trBusy">{{ trBusy ? '…' : 'Tracer' }}</button>
      </div>
      <ol v-if="hops.length" class="hops">
        <li v-for="(h, i) in hops" :key="i">
          <span class="num">{{ h.num }}</span>
          <span>{{ h.host }}</span>
          <span class="muted" v-if="h.ms">{{ h.ms }} ms</span>
        </li>
      </ol>
    </section>
  </div>
</template>

<style scoped>
section { margin-bottom: 1.5rem; }
h3 { margin: 0 0 0.6rem; font-size: 0.95rem; }
.copy { margin: 0.8rem 0 0; font-size: 0.82rem; padding: 0.3rem 0.6rem; }
.checks { list-style: none; padding: 0; margin: 0.8rem 0 0; display: flex; flex-direction: column; gap: 0.6rem; }
.checks li { display: flex; gap: 0.6rem; align-items: flex-start; }
.checks .dot { margin-top: 5px; flex: 0 0 auto; }
.form { display: flex; gap: 0.5rem; }
.form input { flex: 1; }
.line { display: flex; align-items: center; gap: 0.5rem; margin-top: 0.6rem; }
.hops { margin: 0.7rem 0 0; padding-left: 1.2rem; font-size: 0.85rem; }
.hops li { display: flex; gap: 0.6rem; margin-bottom: 0.25rem; }
.hops .num { color: var(--muted); min-width: 1.5rem; }
</style>

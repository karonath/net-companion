<script setup>
import { ref, watch } from 'vue'
import { api } from '../api'
import { state, copyText } from '../state'
import { friendlyError } from '../errors'
import HelpNote from './HelpNote.vue'

const checks = ref([])
const busy = ref(false)
const err = ref('')

// Diagnostic ciblé sur un hôte (joignabilité, latence, ports ouverts, nom).
const hostTarget = ref('')
const hostChecks = ref([])
const hostBusy = ref(false)
const hostErr = ref('')
async function runHost(host) {
  if (!host) return
  hostTarget.value = host
  hostBusy.value = true
  hostErr.value = ''
  hostChecks.value = []
  try {
    const res = await api.diagHost(host)
    hostChecks.value = res.checks || []
  } catch (e) {
    hostErr.value = friendlyError(e, "Diagnostic de l'hôte impossible.")
  } finally {
    hostBusy.value = false
  }
}

// Pré-remplissage depuis le détail d'hôte (via seq : re-déclenche même si l'hôte
// ciblé est le même qu'au clic précédent). Lance directement le diagnostic ciblé.
watch(
  () => state.prefill.seq,
  () => {
    if (state.prefill.tab === 'diag' && state.prefill.diagHost) {
      const h = state.prefill.diagHost
      ph.value = h
      tt.value = h
      runHost(h)
    }
  }
)

async function run() {
  busy.value = true
  err.value = ''
  try {
    const res = await api.diag()
    checks.value = res.checks || []
  } catch (e) {
    checks.value = []
    err.value = friendlyError(e, 'Diagnostics impossibles.')
  } finally {
    busy.value = false
  }
}

// Test de port
const ph = ref('')
const pp = ref('')
const portResult = ref(null)
const portBusy = ref(false)
async function checkPort() {
  const port = Number(pp.value)
  if (!ph.value) {
    portResult.value = { status: 'fail', detail: 'Renseignez un hôte.' }
    return
  }
  if (!Number.isInteger(port) || port < 1 || port > 65535) {
    portResult.value = { status: 'fail', detail: 'Port invalide (1–65535).' }
    return
  }
  portBusy.value = true
  portResult.value = null
  try {
    portResult.value = await api.diagPort(ph.value, port)
  } catch (e) {
    portResult.value = { status: 'fail', detail: friendlyError(e, 'Test du port impossible.') }
  } finally {
    portBusy.value = false
  }
}

// Traceroute
const tt = ref('')
const hops = ref([])
const trBusy = ref(false)
const trErr = ref('')
async function traceroute() {
  if (!tt.value) {
    trErr.value = 'Renseignez une cible.'
    return
  }
  trBusy.value = true
  trErr.value = ''
  hops.value = []
  try {
    const res = await api.diagTraceroute(tt.value)
    hops.value = res.hops || []
  } catch (e) {
    trErr.value = friendlyError(e, 'Traceroute impossible.')
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
    <HelpNote>
      Vérifie la santé réseau sans matériel : passerelle joignable, résolution
      DNS, accès Internet, latence/jitter. En bas, deux outils à la demande :
      test d'un port TCP (host:port) et traceroute vers une cible. Depuis un hôte
      du radar, un diagnostic <strong>ciblé</strong> (joignabilité, latence, ports
      ouverts) s'affiche ci-dessous.
    </HelpNote>

    <section v-if="hostTarget" class="hostdiag">
      <h3>Diagnostic de l'hôte — {{ hostTarget }}</h3>
      <button class="primary" @click="runHost(hostTarget)" :disabled="hostBusy">
        {{ hostBusy ? 'Analyse…' : 'Relancer' }}
      </button>
      <p v-if="hostErr" class="err">{{ hostErr }}</p>
      <ul v-if="hostChecks.length" class="checks">
        <li v-for="(c, i) in hostChecks" :key="i">
          <span class="dot" :class="dotClass(c.status)"></span>
          <div>
            <strong>{{ c.name }}</strong>
            <div class="muted">{{ c.detail }}</div>
          </div>
        </li>
      </ul>
    </section>

    <section>
      <h3>Diagnostics de connectivité <span class="muted">(Internet / DNS / passerelle)</span></h3>
      <button class="primary" @click="run" :disabled="busy">
        {{ busy ? 'Analyse…' : 'Lancer les diagnostics' }}
      </button>
      <p v-if="err" class="err">{{ err }}</p>
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
        <input v-model="ph" placeholder="Hôte (ex: 192.168.1.1)" aria-label="Hôte à tester" />
        <input v-model="pp" placeholder="Port" inputmode="numeric" style="max-width: 90px" aria-label="Port TCP à tester" />
        <button @click="checkPort" :disabled="portBusy">{{ portBusy ? '…' : 'Tester' }}</button>
      </div>
      <div v-if="portResult" class="line">
        <span class="dot" :class="dotClass(portResult.status)"></span>
        {{ portResult.detail }}
      </div>
    </section>

    <section>
      <h3>Traceroute</h3>
      <div class="form">
        <input v-model="tt" placeholder="Cible (ex: 1.1.1.1)" @keyup.enter="traceroute" aria-label="Cible du traceroute" />
        <button @click="traceroute" :disabled="trBusy">{{ trBusy ? '…' : 'Tracer' }}</button>
      </div>
      <p v-if="trErr" class="err">{{ trErr }}</p>
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
.err { color: var(--red); margin: 0.6rem 0 0; }
.hostdiag {
  border: 1px solid var(--accent);
  border-radius: 10px;
  padding: 0.9rem;
  background: var(--panel-2);
}
.hostdiag h3 { margin-top: 0; }
</style>

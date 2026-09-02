<script setup>
import { ref, watch } from 'vue'
import { api } from '../api'
import { state, clearHost, gotoConfigDiff, gotoDiag } from '../state'

const info = ref(null)
const portMsg = ref('')
const portState = ref('')
const busy = ref(false)

watch(
  () => state.selectedHost,
  async (h) => {
    info.value = null
    portMsg.value = ''
    portState.value = ''
    if (!h) return
    try {
      info.value = await api.networkHost(h.ip)
    } catch {
      info.value = null
    }
  },
  { immediate: true }
)

async function locatePort() {
  const h = state.selectedHost
  if (!h) return
  busy.value = true
  portMsg.value = ''
  portState.value = ''
  try {
    const loc = await api.portfinder({ targetMac: h.mac })
    portMsg.value = loc.sentence
    portState.value = 'ok'
  } catch (e) {
    portState.value = 'warn'
    if (e.status === 423) portMsg.value = 'Coffre verrouillé — déverrouillez pour interroger le switch.'
    else if (e.status === 400) portMsg.value = 'Ajoutez une community SNMP dans le coffre.'
    else portMsg.value = 'Localisation impossible : ' + e.message
  } finally {
    busy.value = false
  }
}
</script>

<template>
  <div v-if="state.selectedHost" class="card panel">
    <button class="close" @click="clearHost" aria-label="Fermer">✕</button>
    <div class="head">
      <span class="dot" :class="state.selectedHost.isGateway ? 'orange' : 'green'"></span>
      <strong>{{ state.selectedHost.ip }}</strong>
      <span v-if="state.selectedHost.isGateway" class="tag muted">passerelle</span>
    </div>
    <dl>
      <div v-if="state.selectedHost.name"><dt>Nom (mDNS)</dt><dd>{{ state.selectedHost.name }}</dd></div>
      <div v-if="state.selectedHost.model"><dt>Modèle</dt><dd>{{ state.selectedHost.model }}</dd></div>
      <div v-if="state.selectedHost.vendor"><dt>Fabricant</dt><dd>{{ state.selectedHost.vendor }}</dd></div>
      <div v-if="state.selectedHost.mac"><dt>MAC</dt><dd>{{ state.selectedHost.mac }}</dd></div>
      <div v-if="info && info.hostname"><dt>Nom d'hôte</dt><dd>{{ info.hostname }}</dd></div>
      <div v-if="info"><dt>Latence</dt><dd>{{ info.latencyMs >= 0 ? info.latencyMs + ' ms' : 'injoignable' }}</dd></div>
    </dl>

    <div class="actions">
      <button class="primary" @click="locatePort" :disabled="busy">
        {{ busy ? '…' : 'Localiser son port' }}
      </button>
      <button @click="gotoConfigDiff(state.selectedHost.ip)">Config-Diff</button>
      <button @click="gotoDiag(state.selectedHost.ip)">Diagnostics</button>
    </div>

    <p v-if="portMsg" :class="portState === 'ok' ? 'ok' : 'warn'">{{ portMsg }}</p>
  </div>
</template>

<style scoped>
.card {
  position: absolute;
  top: 4rem;
  left: 1rem;
  width: 300px;
  padding: 1rem 1.1rem;
  z-index: 5;
  box-shadow: 0 10px 30px rgba(0, 0, 0, 0.4);
}
.close {
  position: absolute; top: 0.5rem; right: 0.5rem;
  border: none; background: transparent; color: var(--muted); padding: 0.2rem 0.4rem;
}
.head { display: flex; align-items: center; gap: 0.5rem; font-size: 1.05rem; margin-bottom: 0.6rem; }
dl { margin: 0 0 0.8rem; }
dl > div { display: flex; justify-content: space-between; gap: 1rem; padding: 0.2rem 0; font-size: 0.9rem; }
dt { color: var(--muted); }
dd { margin: 0; text-align: right; word-break: break-all; }
.actions { display: flex; flex-wrap: wrap; gap: 0.5rem; }
.ok { color: var(--green); margin: 0.8rem 0 0; }
.warn { color: var(--orange); margin: 0.8rem 0 0; }
</style>

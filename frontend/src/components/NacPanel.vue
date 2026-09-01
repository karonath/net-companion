<script setup>
import { ref } from 'vue'
import { api } from '../api'

// LLDP
const lldp = ref(null)
const lldpBusy = ref(false)
async function scanLLDP() {
  lldpBusy.value = true
  try {
    lldp.value = await api.lldp()
  } catch (e) {
    lldp.value = { available: false, reason: e.message, neighbors: [] }
  } finally {
    lldpBusy.value = false
  }
}

// MAC spoof
const mac = ref('')
const plan = ref(null)
const spoofMsg = ref('')
const spoofState = ref('') // '' | ok | warn
const spoofBusy = ref(false)

async function dryRun() {
  spoofMsg.value = ''
  spoofState.value = ''
  plan.value = null
  spoofBusy.value = true
  try {
    const res = await api.spoof({ mac: mac.value, apply: false })
    plan.value = res.plan
  } catch (e) {
    spoofMsg.value = e.message
    spoofState.value = 'warn'
  } finally {
    spoofBusy.value = false
  }
}

async function apply() {
  spoofMsg.value = ''
  spoofState.value = ''
  spoofBusy.value = true
  try {
    await api.spoof({ mac: mac.value, apply: true })
    spoofMsg.value = 'MAC appliquée et bail DHCP renouvelé.'
    spoofState.value = 'ok'
  } catch (e) {
    if (e.status === 403) {
      spoofMsg.value = 'Élévation requise : relancez Net-Companion en administrateur.'
    } else {
      spoofMsg.value = 'Échec : ' + e.message
    }
    spoofState.value = 'warn'
  } finally {
    spoofBusy.value = false
  }
}
</script>

<template>
  <div class="nac">
    <p class="intro muted">
      Que faire quand le réseau te <strong>bloque</strong> (prise verrouillée, pas d'IP).
      Différent de l'onglet <strong>Voisins</strong> qui interroge par SNMP (nécessite une IP).
    </p>
    <section>
      <h3>Écoute passive LLDP/CDP</h3>
      <p class="muted">
        Détecte le switch/port même sans adresse IP (réseau bloqué).
      </p>
      <button @click="scanLLDP" :disabled="lldpBusy">
        {{ lldpBusy ? 'Écoute…' : 'Écouter le LLDP' }}
      </button>

      <div v-if="lldp" class="result">
        <div v-if="!lldp.available" class="tag warn">
          <span class="dot orange"></span> {{ lldp.reason }}
        </div>
        <ul v-else-if="lldp.neighbors.length">
          <li v-for="(n, i) in lldp.neighbors" :key="i">
            <span class="dot green"></span>
            Branché sur <strong>{{ n.systemName || n.chassisId }}</strong>,
            port <strong>{{ n.portId }}</strong>
          </li>
        </ul>
        <div v-else class="muted">Aucune trame LLDP captée.</div>
      </div>
    </section>

    <section>
      <h3>Bypass NAC — MAC Spoofing</h3>
      <p class="muted">
        Usurpez la MAC d'un appareil légitime débranché (ex : imprimante).
      </p>
      <div class="form">
        <input v-model="mac" placeholder="MAC cible (ex: 00:11:22:33:44:55)" />
        <button @click="dryRun" :disabled="spoofBusy">Aperçu</button>
        <button class="danger" @click="apply" :disabled="spoofBusy || !plan">Appliquer</button>
      </div>

      <div v-if="plan" class="plan">
        <div class="muted">Plan ({{ plan.os }}, MAC {{ plan.mac }}) :</div>
        <ol>
          <li v-for="(s, i) in plan.steps" :key="i">{{ s.description }}</li>
        </ol>
      </div>

      <p v-if="spoofMsg" :class="spoofState === 'ok' ? 'ok' : 'err'">{{ spoofMsg }}</p>
    </section>
  </div>
</template>

<style scoped>
.intro { margin: 0 0 1.2rem; font-size: 0.85rem; }
section { margin-bottom: 1.5rem; }
h3 { margin: 0 0 0.35rem; font-size: 0.95rem; }
p { margin: 0 0 0.7rem; }
.result { margin-top: 0.8rem; }
.form { display: flex; flex-wrap: wrap; gap: 0.5rem; }
.form input { flex: 1 1 100%; }
.tag.warn { border-color: var(--orange); }
ul, ol { margin: 0.5rem 0 0; padding-left: 1.1rem; }
ul { list-style: none; padding: 0; }
ul li { display: flex; align-items: center; gap: 0.5rem; margin-bottom: 0.35rem; }
.plan { margin-top: 0.8rem; font-size: 0.9rem; }
.ok { color: var(--green); }
.err { color: var(--red); }
</style>

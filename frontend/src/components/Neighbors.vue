<script setup>
import { ref } from 'vue'
import { api } from '../api'
import { state } from '../state'

const deviceIp = ref('')
const list = ref([])
const done = ref(false)
const err = ref('')
const busy = ref(false)

async function discover(demo = false) {
  busy.value = true
  err.value = ''
  done.value = false
  list.value = []
  try {
    const res = await api.neighbors(deviceIp.value, demo)
    list.value = res.neighbors || []
    done.value = true
  } catch (e) {
    if (e.status === 423) err.value = 'Coffre verrouillé.'
    else if (e.status === 400) err.value = 'Ajoutez une community/utilisateur SNMP.'
    else err.value = 'Découverte impossible : ' + e.message
  } finally {
    busy.value = false
  }
}
</script>

<template>
  <div class="nb">
    <h3>Voisinage LLDP / CDP</h3>
    <p class="muted">Interroge un équipement par SNMP pour lister ses voisins (switch-à-switch).</p>

    <button v-if="state.sim.enabled" class="demo" @click="discover(true)" :disabled="busy">
      Démo (switch simulé)
    </button>
    <div class="form">
      <input v-model="deviceIp" placeholder="IP de l'équipement (ex: 192.168.1.1)" @keyup.enter="discover(false)" />
      <button class="primary" @click="discover(false)" :disabled="busy">
        {{ busy ? '…' : 'Découvrir' }}
      </button>
    </div>

    <p v-if="err" class="err">{{ err }}</p>
    <div v-if="done && !list.length" class="muted">Aucun voisin LLDP/CDP détecté.</div>

    <ul v-if="list.length" class="list">
      <li v-for="(n, i) in list" :key="i">
        <span class="dot green"></span>
        <div>
          <strong>{{ n.remoteSysName || n.remoteChassisId }}</strong>
          <span class="tag muted">{{ n.source }}</span>
          <div class="muted">
            port local {{ n.localPort }} ↔ {{ n.remotePortId }}
          </div>
        </div>
      </li>
    </ul>
  </div>
</template>

<style scoped>
h3 { margin: 0 0 0.3rem; font-size: 0.95rem; }
p { margin: 0 0 0.7rem; }
.demo { width: 100%; margin-bottom: 0.6rem; font-size: 0.85rem; }
.form { display: flex; gap: 0.5rem; }
.form input { flex: 1; }
.err { color: var(--red); font-size: 0.9rem; }
.list { list-style: none; padding: 0; margin: 0.9rem 0 0; display: flex; flex-direction: column; gap: 0.6rem; }
.list li { display: flex; gap: 0.6rem; align-items: flex-start; }
.list .dot { margin-top: 5px; flex: 0 0 auto; }
</style>

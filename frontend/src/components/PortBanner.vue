<script setup>
import { ref } from 'vue'
import { api } from '../api'

const sentence = ref('')
const state = ref('idle') // idle | ok | warn
const busy = ref(false)

async function locate() {
  busy.value = true
  try {
    const loc = await api.portfinder({})
    sentence.value = loc.sentence || 'Localisation obtenue.'
    state.value = 'ok'
  } catch (e) {
    if (e.status === 423) {
      sentence.value = 'Coffre verrouillé — déverrouillez pour interroger le switch.'
    } else if (e.status === 400) {
      sentence.value = 'Ajoutez une community SNMP dans le coffre pour localiser votre port.'
    } else {
      sentence.value = 'Localisation impossible : ' + e.message
    }
    state.value = 'warn'
  } finally {
    busy.value = false
  }
}
</script>

<template>
  <div class="banner" :class="state">
    <div class="txt">
      <span v-if="state === 'ok'" class="dot green"></span>
      <span v-else-if="state === 'warn'" class="dot orange"></span>
      <span v-else class="dot"></span>
      <span v-if="sentence">{{ sentence }}</span>
      <span v-else class="muted">Où suis-je branché ? Lancez le Port-Finder pour identifier switch, port et VLAN.</span>
    </div>
    <button class="primary" @click="locate" :disabled="busy">
      {{ busy ? 'Recherche…' : 'Localiser mon port' }}
    </button>
  </div>
</template>

<style scoped>
.banner {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
  margin: 0 1rem;
  padding: 0.9rem 1.1rem;
  background: var(--panel);
  border: 1px solid var(--border);
  border-radius: var(--radius);
  flex-wrap: wrap;
}
.banner.ok { border-color: var(--green); }
.banner.warn { border-color: var(--orange); }
.txt {
  display: flex;
  align-items: center;
  gap: 0.6rem;
  font-size: 1.05rem;
}
.dot { background: var(--muted); }
</style>

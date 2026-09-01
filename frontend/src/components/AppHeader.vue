<script setup>
import { ref, onMounted } from 'vue'
import { api } from '../api'
import { state } from '../state'

defineEmits(['lock'])

const demoBusy = ref(false)
async function enableDemo() {
  demoBusy.value = true
  try {
    state.sim = await api.simEnable()
  } catch {
    /* ignoré */
  } finally {
    demoBusy.value = false
  }
}

const iface = ref(null)
const gateway = ref('')
const publicip = ref('')
const err = ref(false)

async function load() {
  try {
    const info = await api.networkInfo()
    iface.value = info.interface
    gateway.value = info.gateway
    err.value = false
  } catch {
    err.value = true
  }
  try {
    const p = await api.publicIP()
    publicip.value = p.ip || ''
  } catch {
    publicip.value = ''
  }
}
onMounted(load)
</script>

<template>
  <header class="hdr">
    <div class="brand">
      <span class="dot green"></span>
      <strong>Net-Companion</strong> <span class="muted">Lite</span>
    </div>

    <div class="status">
      <span v-if="iface" class="tag">
        <span class="dot green"></span>
        {{ iface.name }} · {{ iface.ipv4 }}
      </span>
      <span v-else class="tag">
        <span class="dot red"></span> interface indisponible
      </span>
      <span v-if="gateway" class="tag muted">GW {{ gateway }}</span>
      <span v-if="publicip" class="tag muted">WAN {{ publicip }}</span>

      <span v-if="state.sim.enabled" class="tag demo">
        <span class="dot orange"></span> Simulateur actif
      </span>
      <button v-else @click="enableDemo" :disabled="demoBusy" title="Démarrer un équipement simulé pour tester sans matériel">
        {{ demoBusy ? '…' : 'Mode démo' }}
      </button>

      <span class="tag"><span class="dot green"></span> Coffre déverrouillé</span>
      <button class="danger" @click="$emit('lock')">Verrouiller</button>
    </div>
  </header>
</template>

<style scoped>
.hdr {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
  padding: 0.75rem 1.25rem;
  background: var(--panel);
  border-bottom: 1px solid var(--border);
  flex-wrap: wrap;
}
.brand {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  font-size: 1.05rem;
}
.status {
  display: flex;
  align-items: center;
  gap: 0.6rem;
  flex-wrap: wrap;
}
.tag.demo { border-color: var(--orange); color: var(--orange); }
</style>

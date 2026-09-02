<script setup>
import { ref, onMounted } from 'vue'
import { api } from '../api'
import { friendlyError } from '../errors'
import { state } from '../state'

defineEmits(['lock'])

const demoBusy = ref(false)
const demoErr = ref('')
async function enableDemo() {
  demoBusy.value = true
  demoErr.value = ''
  try {
    state.sim = await api.simEnable()
  } catch (e) {
    demoErr.value = friendlyError(e, 'Activation de la démo impossible.')
  } finally {
    demoBusy.value = false
  }
}
async function disableDemo() {
  demoBusy.value = true
  try {
    state.sim = await api.simDisable()
  } catch {
    state.sim = { enabled: false, ssh: '', demoMac: '', user: '' }
  } finally {
    demoBusy.value = false
  }
}

const iface = ref(null)
const gateway = ref('')
const publicip = ref('')
const err = ref(false)

// Élévation (droits administrateur)
const sys = ref({ elevated: false, canElevate: false })
const elevating = ref(false)
async function loadSystem() {
  try {
    sys.value = await api.systemInfo()
  } catch {
    sys.value = { elevated: false, canElevate: false }
  }
}
async function elevate() {
  if (!window.confirm('Relancer Net-Companion en administrateur ? Une invite Windows (UAC) va s’afficher, puis une nouvelle fenêtre s’ouvrira.')) return
  elevating.value = true
  try {
    await api.elevate()
    // Le serveur va quitter puis redémarrer élevé et rouvrir le navigateur.
  } catch {
    elevating.value = false
    // UAC annulée ou erreur : on reste en mode standard.
  }
}

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
onMounted(() => {
  load()
  loadSystem()
})
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
      <button v-if="state.sim.enabled" class="danger" @click="disableDemo" :disabled="demoBusy"
        title="Arrêter le simulateur et revenir aux vraies données">
        {{ demoBusy ? '…' : 'Désactiver la démo' }}
      </button>
      <button v-else @click="enableDemo" :disabled="demoBusy"
        title="Démarrer un équipement simulé pour tester sans matériel">
        {{ demoBusy ? '…' : 'Mode démo' }}
      </button>
      <span v-if="demoErr" class="tag errtag"><span class="dot red"></span> {{ demoErr }}</span>

      <span v-if="sys.elevated" class="tag admin" title="Fonctions avancées (MAC spoofing) disponibles">
        <span class="dot green"></span> Administrateur
      </span>
      <button v-else-if="sys.canElevate" @click="elevate" :disabled="elevating"
        title="Relancer avec les droits administrateur (UAC) pour débloquer le MAC spoofing">
        {{ elevating ? 'Élévation…' : 'Élever (admin)' }}
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
.tag.errtag { border-color: var(--red); color: var(--red); }
.tag.admin { border-color: var(--green); color: var(--green); }
</style>

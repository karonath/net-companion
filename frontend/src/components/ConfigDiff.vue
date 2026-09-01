<script setup>
import { ref, watch } from 'vue'
import { api } from '../api'
import { state } from '../state'

const deviceIp = ref('')

// Pré-remplissage depuis le détail d'hôte.
watch(
  () => state.prefill.configDiffIp,
  (ip) => {
    if (ip) deviceIp.value = ip
  }
)

function useDemo() {
  if (state.sim.enabled) deviceIp.value = state.sim.ssh
}
const lines = ref([])
const err = ref('')
const busy = ref(false)
const done = ref(false)

async function run() {
  if (!deviceIp.value) return
  busy.value = true
  err.value = ''
  done.value = false
  lines.value = []
  try {
    const res = await api.configdiff(deviceIp.value)
    lines.value = res.lines || []
    done.value = true
  } catch (e) {
    if (e.status === 423) err.value = 'Coffre verrouillé.'
    else if (e.status === 400) err.value = 'Ajoutez un identifiant SSH dans le coffre.'
    else err.value = 'Échec SSH : ' + e.message
  } finally {
    busy.value = false
  }
}
</script>

<template>
  <div class="cd">
    <h3>Config-Diff (running vs startup)</h3>
    <button v-if="state.sim.enabled" class="demo" @click="useDemo">
      Utiliser l'équipement de démo ({{ state.sim.ssh }})
    </button>
    <div class="form">
      <input v-model="deviceIp" placeholder="IP de l'équipement (ex: 192.168.1.1)"
             @keyup.enter="run" />
      <button class="primary" @click="run" :disabled="busy">
        {{ busy ? 'SSH…' : 'Comparer' }}
      </button>
    </div>

    <p v-if="err" class="err">{{ err }}</p>

    <div v-if="done && !lines.length" class="muted">
      Aucune différence : running-config = startup-config.
    </div>

    <pre v-if="lines.length" class="diff"><code
      v-for="(l, i) in lines" :key="i" :class="l.op"
    >{{ l.op === 'add' ? '+ ' : l.op === 'del' ? '- ' : '  ' }}{{ l.text }}
</code></pre>
  </div>
</template>

<style scoped>
h3 { margin: 0 0 0.6rem; font-size: 0.95rem; }
.demo { width: 100%; margin-bottom: 0.6rem; font-size: 0.85rem; }
.form { display: flex; gap: 0.5rem; margin-bottom: 0.8rem; }
.form input { flex: 1; }
.err { color: var(--red); font-size: 0.9rem; }
.diff {
  margin: 0;
  padding: 0.6rem;
  background: var(--bg);
  border: 1px solid var(--border);
  border-radius: 8px;
  overflow-x: auto;
  font-size: 0.82rem;
  line-height: 1.35;
}
.diff code { display: block; white-space: pre; }
.diff code.add { color: var(--green); }
.diff code.del { color: var(--red); }
.diff code.same { color: var(--muted); }
</style>

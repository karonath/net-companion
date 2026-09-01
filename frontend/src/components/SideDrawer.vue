<script setup>
import { ref, watch } from 'vue'
import VaultManager from './VaultManager.vue'
import ConfigDiff from './ConfigDiff.vue'
import NacPanel from './NacPanel.vue'
import Diagnostics from './Diagnostics.vue'
import { state } from '../state'

const tab = ref('diag')

// Un module peut demander l'ouverture d'un onglet (depuis le détail d'hôte).
watch(
  () => state.prefill.tab,
  (t) => {
    if (t) tab.value = t
  }
)
const tabs = [
  { id: 'diag', label: 'Diag' },
  { id: 'vault', label: 'Coffre' },
  { id: 'diff', label: 'Config-Diff' },
  { id: 'nac', label: 'NAC' },
]
</script>

<template>
  <aside class="drawer panel">
    <nav class="tabs">
      <button
        v-for="t in tabs"
        :key="t.id"
        :class="{ active: tab === t.id }"
        @click="tab = t.id"
      >
        {{ t.label }}
      </button>
    </nav>
    <div class="content">
      <Diagnostics v-show="tab === 'diag'" />
      <VaultManager v-show="tab === 'vault'" />
      <ConfigDiff v-show="tab === 'diff'" />
      <NacPanel v-show="tab === 'nac'" />
    </div>
  </aside>
</template>

<style scoped>
.drawer {
  display: flex;
  flex-direction: column;
  overflow: hidden;
  min-height: 0;
}
.tabs {
  display: flex;
  border-bottom: 1px solid var(--border);
}
.tabs button {
  flex: 1;
  border: none;
  border-radius: 0;
  background: transparent;
  color: var(--muted);
  padding: 0.75rem;
}
.tabs button.active {
  color: var(--text);
  box-shadow: inset 0 -2px 0 var(--accent);
}
.content {
  padding: 1rem;
  overflow-y: auto;
}
</style>

<script setup>
import { ref } from 'vue'
import VaultManager from './VaultManager.vue'
import ConfigDiff from './ConfigDiff.vue'
import NacPanel from './NacPanel.vue'

const tab = ref('vault')
const tabs = [
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

<script setup>
import { ref, onMounted } from 'vue'
import { api } from './api'
import VaultGate from './components/VaultGate.vue'
import Dashboard from './components/Dashboard.vue'

const status = ref(null)
const loading = ref(true)

async function refresh() {
  try {
    status.value = await api.vaultStatus()
  } catch {
    status.value = { initialized: false, unlocked: false }
  } finally {
    loading.value = false
  }
}

function onUnlocked() {
  refresh()
}

async function onLock() {
  await api.vaultLock()
  refresh()
}

onMounted(refresh)
</script>

<template>
  <div v-if="loading" class="boot">Chargement…</div>
  <VaultGate
    v-else-if="!status.unlocked"
    :initialized="status.initialized"
    @unlocked="onUnlocked"
  />
  <Dashboard v-else @lock="onLock" />
</template>

<style scoped>
.boot {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--muted);
}
</style>

<script setup>
import { onMounted } from 'vue'
import AppHeader from './AppHeader.vue'
import PortBanner from './PortBanner.vue'
import TopologyGraph from './TopologyGraph.vue'
import SideDrawer from './SideDrawer.vue'
import HostDetail from './HostDetail.vue'
import { api } from '../api'
import { state } from '../state'

defineEmits(['lock'])

onMounted(async () => {
  try {
    state.sim = await api.sim()
  } catch {
    // simulateur indisponible : ignoré
  }
})
</script>

<template>
  <div class="app">
    <AppHeader @lock="$emit('lock')" />
    <PortBanner />
    <div class="body">
      <main class="center">
        <TopologyGraph />
        <HostDetail />
      </main>
      <SideDrawer />
    </div>
  </div>
</template>

<style scoped>
.app {
  height: 100vh;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}
.body {
  flex: 1;
  min-height: 0;
  display: grid;
  grid-template-columns: 1fr 380px;
  gap: 1rem;
  padding: 1rem;
}
.center {
  min-width: 0;
  min-height: 0;
  position: relative;
}
@media (max-width: 900px) {
  .app {
    height: auto;
    min-height: 100vh;
    overflow: visible;
  }
  .body {
    grid-template-columns: 1fr;
  }
  .center {
    height: 70vh;
  }
}
</style>

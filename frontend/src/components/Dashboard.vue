<script setup>
import AppHeader from './AppHeader.vue'
import PortBanner from './PortBanner.vue'
import TopologyGraph from './TopologyGraph.vue'
import SideDrawer from './SideDrawer.vue'

defineEmits(['lock'])
</script>

<template>
  <div class="app">
    <AppHeader @lock="$emit('lock')" />
    <PortBanner />
    <div class="body">
      <main class="center">
        <TopologyGraph />
      </main>
      <SideDrawer />
    </div>
  </div>
</template>

<style scoped>
.app {
  min-height: 100vh;
  display: flex;
  flex-direction: column;
}
.body {
  flex: 1;
  display: grid;
  grid-template-columns: 1fr 380px;
  gap: 1rem;
  padding: 1rem;
  min-height: 0;
}
.center {
  min-width: 0;
}
@media (max-width: 900px) {
  .body {
    grid-template-columns: 1fr;
  }
}
</style>

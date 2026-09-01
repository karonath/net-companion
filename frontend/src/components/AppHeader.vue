<script setup>
import { ref, onMounted } from 'vue'
import { api } from '../api'

defineEmits(['lock'])

const iface = ref(null)
const gateway = ref('')
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
</style>

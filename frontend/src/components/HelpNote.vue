<script setup>
import { ref } from 'vue'
const open = ref(false)
</script>

<template>
  <div class="help">
    <button class="toggle" @click="open = !open">
      <span class="i">ⓘ</span> {{ open ? 'Masquer l\'aide' : 'À quoi ça sert ?' }}
    </button>
    <p v-if="open" class="body"><slot /></p>
  </div>
</template>

<style scoped>
.help { margin-bottom: 0.7rem; }
.toggle {
  border: none; background: transparent; color: var(--accent);
  padding: 0; font-size: 0.82rem; cursor: pointer;
}
.i { opacity: 0.9; }
.body {
  margin: 0.5rem 0 0; font-size: 0.85rem; color: var(--muted);
  border-left: 2px solid var(--border); padding-left: 0.6rem;
}
</style>

<script setup>
</script>

<template>
  <main class="nc-hello">
    <h1>Net-Companion&nbsp;Lite</h1>
    <p class="status">
      <span class="dot"></span> Serveur local opérationnel — Étape&nbsp;1
    </p>
    <p class="sub">Binaire unique · frontend embarqué via go:embed</p>
  </main>
</template>

<style>
:root { color-scheme: dark; }
body { margin: 0; }
.nc-hello {
  min-height: 100vh;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 0.75rem;
  background: #0d1117;
  color: #e6edf3;
  font-family: system-ui, -apple-system, "Segoe UI", sans-serif;
}
.nc-hello h1 { margin: 0; font-size: 2.5rem; letter-spacing: -0.02em; }
.status { display: flex; align-items: center; gap: 0.5rem; color: #3fb950; font-weight: 600; }
.dot { width: 10px; height: 10px; border-radius: 50%; background: #3fb950; box-shadow: 0 0 8px #3fb950; }
.sub { color: #8b949e; font-size: 0.9rem; margin: 0; }
</style>

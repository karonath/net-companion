<script setup>
import { ref, onMounted } from 'vue'
import { api } from '../api'

const snmp = ref([])
const ssh = ref([])
const err = ref('')

const newSnmp = ref({ label: '', community: '', version: 'v2c' })
const newSsh = ref({ label: '', username: '', password: '' })

async function load() {
  try {
    snmp.value = await api.listSNMP()
    ssh.value = await api.listSSH()
    err.value = ''
  } catch (e) {
    err.value = e.message
  }
}

async function addSnmp() {
  if (!newSnmp.value.community) return
  try {
    await api.addSNMP({ ...newSnmp.value })
    newSnmp.value = { label: '', community: '', version: 'v2c' }
    load()
  } catch (e) {
    err.value = e.message
  }
}
async function delSnmp(id) {
  await api.delSNMP(id)
  load()
}
async function addSsh() {
  if (!newSsh.value.username) return
  try {
    await api.addSSH({ ...newSsh.value })
    newSsh.value = { label: '', username: '', password: '' }
    load()
  } catch (e) {
    err.value = e.message
  }
}
async function delSsh(id) {
  await api.delSSH(id)
  load()
}

onMounted(load)
</script>

<template>
  <div class="vault">
    <p v-if="err" class="err">{{ err }}</p>

    <section>
      <h3>Communities SNMP</h3>
      <ul>
        <li v-for="c in snmp" :key="c.id">
          <span><strong>{{ c.label || c.community }}</strong>
            <span class="muted"> · {{ c.community }} ({{ c.version }})</span></span>
          <button class="danger sm" @click="delSnmp(c.id)">✕</button>
        </li>
        <li v-if="!snmp.length" class="muted empty">Aucune community.</li>
      </ul>
      <div class="form">
        <input v-model="newSnmp.label" placeholder="Libellé (ex: prod)" />
        <input v-model="newSnmp.community" placeholder="Community (ex: public)" />
        <select v-model="newSnmp.version">
          <option value="v2c">v2c</option>
          <option value="v1">v1</option>
        </select>
        <button class="primary" @click="addSnmp">Ajouter</button>
      </div>
    </section>

    <section>
      <h3>Identifiants SSH</h3>
      <ul>
        <li v-for="c in ssh" :key="c.id">
          <span><strong>{{ c.label || c.username }}</strong>
            <span class="muted"> · {{ c.username }}</span></span>
          <button class="danger sm" @click="delSsh(c.id)">✕</button>
        </li>
        <li v-if="!ssh.length" class="muted empty">Aucun identifiant.</li>
      </ul>
      <div class="form">
        <input v-model="newSsh.label" placeholder="Libellé (ex: core)" />
        <input v-model="newSsh.username" placeholder="Utilisateur" />
        <input v-model="newSsh.password" type="password" placeholder="Mot de passe" />
        <button class="primary" @click="addSsh">Ajouter</button>
      </div>
    </section>
  </div>
</template>

<style scoped>
section { margin-bottom: 1.5rem; }
h3 { margin: 0 0 0.6rem; font-size: 0.95rem; }
ul { list-style: none; padding: 0; margin: 0 0 0.7rem; display: flex; flex-direction: column; gap: 0.4rem; }
li {
  display: flex; align-items: center; justify-content: space-between;
  gap: 0.5rem; background: var(--panel-2); border: 1px solid var(--border);
  border-radius: 8px; padding: 0.45rem 0.65rem; font-size: 0.9rem;
}
li.empty { justify-content: center; font-style: italic; }
.form { display: flex; flex-direction: column; gap: 0.5rem; }
button.sm { padding: 0.2rem 0.5rem; }
.err { color: var(--red); font-size: 0.9rem; }
</style>

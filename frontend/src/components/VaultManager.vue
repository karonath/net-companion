<script setup>
import { ref, onMounted } from 'vue'
import { api } from '../api'
import { friendlyError } from '../errors'
import HelpNote from './HelpNote.vue'

const snmp = ref([])
const ssh = ref([])
const err = ref('')

const emptySnmp = () => ({
  label: '', version: 'v2c', community: '',
  securityName: '', securityLevel: 'authPriv',
  authProtocol: 'SHA', authPassphrase: '',
  privProtocol: 'AES', privPassphrase: '',
})
const newSnmp = ref(emptySnmp())
const newSsh = ref({ label: '', username: '', password: '' })

const testIp = ref('')
const testResults = ref({})
async function testCred(type, id) {
  if (!testIp.value) {
    testResults.value = { ...testResults.value, [id]: { ok: false, detail: "renseignez l'IP de test ci-dessus" } }
    return
  }
  testResults.value = { ...testResults.value, [id]: { pending: true } }
  try {
    const r = await api.vaultTest(type, id, testIp.value)
    testResults.value = { ...testResults.value, [id]: r }
  } catch (e) {
    testResults.value = { ...testResults.value, [id]: { ok: false, detail: e.status === 423 ? 'coffre verrouillé' : e.message } }
  }
}

async function load() {
  try {
    snmp.value = await api.listSNMP()
    ssh.value = await api.listSSH()
    err.value = ''
  } catch (e) {
    err.value = friendlyError(e, 'Chargement du coffre impossible.')
  }
}

async function addSnmp() {
  const c = newSnmp.value
  if (c.version === 'v2c' && !c.community) {
    err.value = 'Renseignez la community (SNMP v2c).'
    return
  }
  if (c.version === 'v3' && !c.securityName) {
    err.value = "Renseignez le nom d'utilisateur (SNMP v3)."
    return
  }
  try {
    err.value = ''
    await api.addSNMP({ ...c })
    newSnmp.value = emptySnmp()
    load()
  } catch (e) {
    err.value = friendlyError(e, "Ajout de l'identifiant SNMP impossible.")
  }
}
async function delSnmp(id) {
  if (!window.confirm('Supprimer définitivement cet identifiant SNMP ?')) return
  try {
    err.value = ''
    await api.delSNMP(id)
    await load()
  } catch (e) {
    err.value = friendlyError(e, 'Suppression impossible.')
  }
}
async function addSsh() {
  if (!newSsh.value.username) {
    err.value = "Renseignez l'utilisateur SSH."
    return
  }
  try {
    err.value = ''
    await api.addSSH({ ...newSsh.value })
    newSsh.value = { label: '', username: '', password: '' }
    load()
  } catch (e) {
    err.value = friendlyError(e, "Ajout de l'identifiant SSH impossible.")
  }
}
async function delSsh(id) {
  if (!window.confirm('Supprimer définitivement cet identifiant SSH ?')) return
  try {
    err.value = ''
    await api.delSSH(id)
    await load()
  } catch (e) {
    err.value = friendlyError(e, 'Suppression impossible.')
  }
}

onMounted(load)
</script>

<template>
  <div class="vault">
    <HelpNote>
      Trousseau chiffré (AES-256, déverrouillé par ton PIN) : stocke tes
      communities SNMP (v2c/v3) et identifiants SSH. Ils sont réutilisés
      automatiquement par le Port-Finder, les Voisins et le Config-Diff.
      Renseigne une « IP de test » puis « Tester » pour valider un credential
      avant d'aller sur le terrain.
    </HelpNote>
    <p v-if="err" class="err">{{ err }}</p>

    <div class="testip">
      <input v-model="testIp" placeholder="IP de test (pour « Tester »)" />
    </div>

    <section>
      <h3>Communities SNMP</h3>
      <ul>
        <li v-for="c in snmp" :key="c.id" class="entry">
          <div class="row">
            <span><strong>{{ c.label || c.community || c.securityName }}</strong>
              <span class="muted"> · {{ c.version === 'v3' ? c.securityName + ' (v3 ' + c.securityLevel + ')' : c.community + ' (v2c)' }}</span></span>
            <span class="acts">
              <button class="sm" @click="testCred('snmp', c.id)">Tester</button>
              <button class="danger sm" @click="delSnmp(c.id)">✕</button>
            </span>
          </div>
          <div v-if="testResults[c.id]" class="res" :class="testResults[c.id].ok ? 'ok' : 'ko'">
            {{ testResults[c.id].pending ? 'test…' : testResults[c.id].detail }}
          </div>
        </li>
        <li v-if="!snmp.length" class="muted empty">Aucun identifiant SNMP.</li>
      </ul>
      <div class="form">
        <input v-model="newSnmp.label" placeholder="Libellé (ex: prod)" />
        <select v-model="newSnmp.version">
          <option value="v2c">SNMP v2c</option>
          <option value="v3">SNMP v3</option>
        </select>

        <template v-if="newSnmp.version === 'v2c'">
          <input v-model="newSnmp.community" placeholder="Community (ex: public)" />
        </template>

        <template v-else>
          <input v-model="newSnmp.securityName" placeholder="Nom d'utilisateur (securityName)" />
          <select v-model="newSnmp.securityLevel">
            <option value="noAuthNoPriv">noAuthNoPriv</option>
            <option value="authNoPriv">authNoPriv</option>
            <option value="authPriv">authPriv</option>
          </select>
          <template v-if="newSnmp.securityLevel !== 'noAuthNoPriv'">
            <select v-model="newSnmp.authProtocol">
              <option>SHA</option><option>SHA256</option><option>SHA512</option><option>MD5</option>
            </select>
            <input v-model="newSnmp.authPassphrase" type="password" placeholder="Passphrase auth" />
          </template>
          <template v-if="newSnmp.securityLevel === 'authPriv'">
            <select v-model="newSnmp.privProtocol">
              <option>AES</option><option>AES256</option><option>DES</option>
            </select>
            <input v-model="newSnmp.privPassphrase" type="password" placeholder="Passphrase priv" />
          </template>
        </template>

        <button class="primary" @click="addSnmp">Ajouter</button>
      </div>
    </section>

    <section>
      <h3>Identifiants SSH</h3>
      <ul>
        <li v-for="c in ssh" :key="c.id" class="entry">
          <div class="row">
            <span><strong>{{ c.label || c.username }}</strong>
              <span class="muted"> · {{ c.username }}</span></span>
            <span class="acts">
              <button class="sm" @click="testCred('ssh', c.id)">Tester</button>
              <button class="danger sm" @click="delSsh(c.id)">✕</button>
            </span>
          </div>
          <div v-if="testResults[c.id]" class="res" :class="testResults[c.id].ok ? 'ok' : 'ko'">
            {{ testResults[c.id].pending ? 'test…' : testResults[c.id].detail }}
          </div>
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
li.entry { flex-direction: column; align-items: stretch; gap: 0.4rem; }
.row { display: flex; align-items: center; justify-content: space-between; gap: 0.5rem; }
.acts { display: flex; gap: 0.4rem; flex: 0 0 auto; }
.res { font-size: 0.82rem; }
.res.ok { color: var(--green); }
.res.ko { color: var(--red); }
.testip { margin-bottom: 1rem; }
.form { display: flex; flex-direction: column; gap: 0.5rem; }
button.sm { padding: 0.2rem 0.5rem; }
.err { color: var(--red); font-size: 0.9rem; }
</style>

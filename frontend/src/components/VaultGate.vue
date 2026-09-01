<script setup>
import { ref, computed } from 'vue'
import { api } from '../api'

const props = defineProps({
  initialized: { type: Boolean, required: true },
})
const emit = defineEmits(['unlocked'])

const pin = ref('')
const pin2 = ref('')
const error = ref('')
const busy = ref(false)

const isCreate = computed(() => !props.initialized)

async function submit() {
  error.value = ''
  if (pin.value.length < 4) {
    error.value = 'Le PIN doit comporter au moins 4 caractères.'
    return
  }
  if (isCreate.value && pin.value !== pin2.value) {
    error.value = 'Les deux PIN ne correspondent pas.'
    return
  }
  busy.value = true
  try {
    if (isCreate.value) {
      await api.vaultInit(pin.value)
    } else {
      await api.vaultUnlock(pin.value)
    }
    emit('unlocked')
  } catch (e) {
    error.value = isCreate.value
      ? 'Création impossible : ' + e.message
      : 'PIN incorrect.'
  } finally {
    busy.value = false
  }
}
</script>

<template>
  <div class="gate">
    <div class="card panel">
      <div class="logo">
        <span class="dot green"></span> Net-Companion&nbsp;Lite
      </div>
      <h1>{{ isCreate ? 'Créer votre code PIN' : 'Déverrouiller le coffre' }}</h1>
      <p class="muted">
        {{ isCreate
          ? 'Ce PIN chiffrera vos identifiants (SNMP / SSH) en AES-256 sur cette machine.'
          : 'Saisissez votre PIN pour déverrouiller le trousseau en mémoire.' }}
      </p>

      <form @submit.prevent="submit">
        <label>PIN</label>
        <input v-model="pin" type="password" inputmode="numeric" autofocus
               placeholder="••••" @keyup.enter="submit" />

        <template v-if="isCreate">
          <label>Confirmer le PIN</label>
          <input v-model="pin2" type="password" inputmode="numeric"
                 placeholder="••••" @keyup.enter="submit" />
        </template>

        <p v-if="error" class="err">{{ error }}</p>

        <button class="primary" type="submit" :disabled="busy">
          {{ busy ? '…' : (isCreate ? 'Créer et déverrouiller' : 'Déverrouiller') }}
        </button>
      </form>
    </div>
  </div>
</template>

<style scoped>
.gate {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 1rem;
}
.card {
  width: 100%;
  max-width: 380px;
  padding: 2rem;
}
.logo {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  font-weight: 600;
  color: var(--muted);
  margin-bottom: 1.5rem;
}
h1 {
  font-size: 1.4rem;
  margin: 0 0 0.5rem;
}
label {
  display: block;
  font-size: 0.85rem;
  color: var(--muted);
  margin: 1rem 0 0.35rem;
}
button.primary {
  width: 100%;
  margin-top: 1.4rem;
}
.err {
  color: var(--red);
  font-size: 0.9rem;
  margin: 0.8rem 0 0;
}
</style>

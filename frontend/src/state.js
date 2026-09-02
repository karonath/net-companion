import { reactive } from 'vue'

// État partagé léger : infos simulateur, hôte sélectionné, et pré-remplissages
// pour router une action depuis le graphe vers un panneau du drawer.
export const state = reactive({
  sim: { enabled: false, ssh: '', demoMac: '', user: '' },
  selectedHost: null, // { ip, mac, vendor, isGateway }
  selectedIface: '', // '' = auto (Mode Universel : nom d'interface pour forcer)
  // seq est incrémenté à chaque navigation pour re-déclencher les watchers même
  // quand l'IP ciblée est identique (sinon un même clic ne se propage pas).
  prefill: { configDiffIp: '', diagHost: '', tab: '', seq: 0 },
})

export function selectHost(h) {
  state.selectedHost = h
}

export function clearHost() {
  state.selectedHost = null
}

export function gotoConfigDiff(ip) {
  state.prefill.configDiffIp = ip
  state.prefill.tab = 'diff'
  state.prefill.seq++
}

export function gotoDiag(host) {
  state.prefill.diagHost = host
  state.prefill.tab = 'diag'
  state.prefill.seq++
}

// copyText copie du texte dans le presse-papiers (best-effort).
export function copyText(text) {
  try {
    navigator.clipboard.writeText(text)
  } catch {
    /* ignoré */
  }
}

import { reactive } from 'vue'

// État partagé léger : infos simulateur, hôte sélectionné, et pré-remplissages
// pour router une action depuis le graphe vers un panneau du drawer.
export const state = reactive({
  sim: { enabled: false, ssh: '', demoMac: '', user: '' },
  selectedHost: null, // { ip, mac, vendor, isGateway }
  prefill: { configDiffIp: '', diagHost: '', tab: '' },
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
}

export function gotoDiag(host) {
  state.prefill.diagHost = host
  state.prefill.tab = 'diag'
}

// Client API centralisé : un wrapper par route du backend Net-Companion.

// reqRaw récupère une réponse texte brute (rapport HTML/JSON) avec le jeton en
// en-tête — évite d'exposer le jeton dans l'URL.
async function reqRaw(method, path) {
  const opts = { method, headers: {} }
  if (typeof window !== 'undefined' && window.__NC_TOKEN__) {
    opts.headers['X-NC-Token'] = window.__NC_TOKEN__
  }
  const res = await fetch(path, opts)
  const text = await res.text()
  if (!res.ok) {
    let data = null
    try {
      data = JSON.parse(text)
    } catch {
      /* réponse non-JSON */
    }
    const err = new Error((data && data.error) || res.statusText || `HTTP ${res.status}`)
    err.status = res.status
    err.data = data
    throw err
  }
  return text
}

async function req(method, path, body) {
  const opts = { method, headers: {} }
  // Jeton de session injecté par le serveur dans la page (anti-CSRF/rebinding).
  if (typeof window !== 'undefined' && window.__NC_TOKEN__) {
    opts.headers['X-NC-Token'] = window.__NC_TOKEN__
  }
  if (body !== undefined) {
    opts.headers['Content-Type'] = 'application/json'
    opts.body = JSON.stringify(body)
  }
  const res = await fetch(path, opts)
  const text = await res.text()
  let data = null
  if (text) {
    try {
      data = JSON.parse(text)
    } catch {
      data = text
    }
  }
  if (!res.ok) {
    const msg = (data && data.error) || res.statusText || `HTTP ${res.status}`
    const err = new Error(msg)
    err.status = res.status
    err.data = data
    throw err
  }
  return data
}

export const api = {
  vaultStatus: () => req('GET', '/api/vault/status'),
  vaultInit: (pin) => req('POST', '/api/vault/init', { pin }),
  vaultUnlock: (pin) => req('POST', '/api/vault/unlock', { pin }),
  vaultLock: () => req('POST', '/api/vault/lock'),

  listSNMP: () => req('GET', '/api/vault/secrets/snmp'),
  addSNMP: (c) => req('POST', '/api/vault/secrets/snmp', c),
  delSNMP: (id) => req('DELETE', '/api/vault/secrets/snmp/' + id),
  listSSH: () => req('GET', '/api/vault/secrets/ssh'),
  addSSH: (c) => req('POST', '/api/vault/secrets/ssh', c),
  delSSH: (id) => req('DELETE', '/api/vault/secrets/ssh/' + id),
  vaultTest: (type, id, deviceIp) => req('POST', '/api/vault/test', { type, id, deviceIp }),

  networkInfo: () => req('GET', '/api/network/info'),
  networkHost: (ip) => req('GET', '/api/network/host?ip=' + encodeURIComponent(ip)),
  publicIP: () => req('GET', '/api/network/publicip'),
  radar: (iface, opts = {}) => {
    const p = new URLSearchParams()
    if (iface) p.set('iface', iface)
    if (opts.quick) p.set('quick', '1')
    const qs = p.toString()
    return req('GET', '/api/network/radar' + (qs ? '?' + qs : ''))
  },
  networkInterfaces: () => req('GET', '/api/network/interfaces'),
  portfinder: (b) => req('POST', '/api/network/portfinder', b),
  neighbors: (deviceIp, demo) =>
    req('POST', '/api/network/neighbors', demo ? { demo: true } : { deviceIp }),
  sim: () => req('GET', '/api/sim'),
  simEnable: () => req('POST', '/api/sim/enable'),
  simDisable: () => req('POST', '/api/sim/disable'),

  lldp: () => req('GET', '/api/nac/lldp'),
  spoof: (b) => req('POST', '/api/nac/spoof', b),

  configdiff: (deviceIp) => req('POST', '/api/configdiff', { deviceIp }),

  diag: () => req('GET', '/api/diag'),
  diagHost: (host) => req('POST', '/api/diag/host', { host }),
  diagPort: (host, port) => req('POST', '/api/diag/port', { host, port }),
  diagTraceroute: (target) => req('POST', '/api/diag/traceroute', { target }),

  checkup: (label, notes) => req('POST', '/api/checkup', { label, notes }),

  configBackup: (devices) => req('POST', '/api/config/backup', { devices }),
  configDevices: () => req('GET', '/api/config/devices'),
  configHistory: (device) => req('GET', '/api/config/history?device=' + encodeURIComponent(device)),
  configBaseline: (device, id) => req('POST', '/api/config/baseline', { device, id }),
  configDrift: (device) => req('GET', '/api/config/drift?device=' + encodeURIComponent(device)),
  history: () => req('GET', '/api/history'),
  clearHistory: () => req('DELETE', '/api/history'),

  // Rapport : récupéré par en-tête (jeton hors URL) puis ouvert/téléchargé via
  // un Blob local.
  async openReport(id) {
    const html = await reqRaw('GET', '/api/report/' + encodeURIComponent(id))
    const url = URL.createObjectURL(new Blob([html], { type: 'text/html' }))
    window.open(url, '_blank')
    setTimeout(() => URL.revokeObjectURL(url), 60000)
  },
  async downloadReportJson(id) {
    const txt = await reqRaw('GET', '/api/report/' + encodeURIComponent(id) + '?format=json')
    const url = URL.createObjectURL(new Blob([txt], { type: 'application/json' }))
    const a = document.createElement('a')
    a.href = url
    a.download = 'net-companion-' + id + '.json'
    document.body.appendChild(a)
    a.click()
    a.remove()
    setTimeout(() => URL.revokeObjectURL(url), 60000)
  },
}

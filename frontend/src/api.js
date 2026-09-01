// Client API centralisé : un wrapper par route du backend Net-Companion.

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
  radar: () => req('GET', '/api/network/radar'),
  portfinder: (b) => req('POST', '/api/network/portfinder', b),
  neighbors: (deviceIp, demo) =>
    req('POST', '/api/network/neighbors', demo ? { demo: true } : { deviceIp }),
  sim: () => req('GET', '/api/sim'),

  lldp: () => req('GET', '/api/nac/lldp'),
  spoof: (b) => req('POST', '/api/nac/spoof', b),

  configdiff: (deviceIp) => req('POST', '/api/configdiff', { deviceIp }),

  diag: () => req('GET', '/api/diag'),
  diagPort: (host, port) => req('POST', '/api/diag/port', { host, port }),
  diagTraceroute: (target) => req('POST', '/api/diag/traceroute', { target }),

  checkup: (label, notes) => req('POST', '/api/checkup', { label, notes }),
  history: () => req('GET', '/api/history'),
  reportUrl: (id) => '/api/report/' + encodeURIComponent(id),
  reportJsonUrl: (id) => '/api/report/' + encodeURIComponent(id) + '?format=json',
}

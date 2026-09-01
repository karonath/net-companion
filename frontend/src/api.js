// Client API centralisé : un wrapper par route du backend Net-Companion.

async function req(method, path, body) {
  const opts = { method, headers: {} }
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

  networkInfo: () => req('GET', '/api/network/info'),
  radar: () => req('GET', '/api/network/radar'),
  portfinder: (b) => req('POST', '/api/network/portfinder', b),

  lldp: () => req('GET', '/api/nac/lldp'),
  spoof: (b) => req('POST', '/api/nac/spoof', b),

  configdiff: (deviceIp) => req('POST', '/api/configdiff', { deviceIp }),

  diag: () => req('GET', '/api/diag'),
  diagPort: (host, port) => req('POST', '/api/diag/port', { host, port }),
  diagTraceroute: (target) => req('POST', '/api/diag/traceroute', { target }),
}

const { request } = require('./request')

let cachedValue = false
let cacheExpiresAt = 0
let pendingPromise = null

function normalizeEntryVisibility(payload) {
  if (payload && typeof payload.showVipEntry === 'boolean') {
    return payload.showVipEntry
  }
  return false
}

function getVIPEntryVisibility(forceRefresh) {
  const now = Date.now()
  if (!forceRefresh && cacheExpiresAt > now) {
    return Promise.resolve(cachedValue)
  }
  if (pendingPromise) {
    return pendingPromise
  }
  pendingPromise = request({ url: '/api/vip/entry-config', method: 'GET' })
    .then((payload) => {
      cachedValue = normalizeEntryVisibility(payload)
      cacheExpiresAt = Date.now() + 30000
      pendingPromise = null
      return cachedValue
    })
    .catch(() => {
      pendingPromise = null
      return cachedValue
    })
  return pendingPromise
}

module.exports = {
  getVIPEntryVisibility
}
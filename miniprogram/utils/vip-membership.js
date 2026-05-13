const { request } = require('./request')
const { getAuthUser } = require('./storage')

let cachedUserID = ''
let cachedMembership = null
let cacheExpiresAt = 0
let pendingPromise = null

function getCurrentServerUserID() {
  const user = getAuthUser()
  if (!user || user.storageMode !== 'server' || !user.id) {
    return ''
  }
  return String(user.id)
}

function normalizeMembership(payload) {
  if (!payload || (!payload.productId && !payload.productName && !payload.statusText)) {
    return null
  }
  return payload
}

function getVIPMembership(forceRefresh) {
  const userID = getCurrentServerUserID()
  if (!userID) {
    cachedUserID = ''
    cachedMembership = null
    cacheExpiresAt = 0
    return Promise.resolve(null)
  }

  const now = Date.now()
  if (!forceRefresh && userID === cachedUserID && cacheExpiresAt > now) {
    return Promise.resolve(cachedMembership)
  }
  if (pendingPromise && userID === cachedUserID) {
    return pendingPromise
  }

  cachedUserID = userID
  pendingPromise = request({
    url: '/api/vip/membership',
    method: 'POST',
    data: { userId: userID },
    timeout: 5000
  }).then((payload) => {
    cachedMembership = normalizeMembership(payload)
    cacheExpiresAt = Date.now() + 30000
    pendingPromise = null
    return cachedMembership
  }).catch(() => {
    pendingPromise = null
    cachedMembership = null
    cacheExpiresAt = 0
    return null
  })
  return pendingPromise
}

function hasActiveVIP(forceRefresh) {
  return getVIPMembership(forceRefresh).then((membership) => {
    return !!(membership && membership.active && Number(membership.endAt || 0) > Date.now())
  })
}

module.exports = {
  getVIPMembership,
  hasActiveVIP
}
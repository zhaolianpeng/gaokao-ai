const { request } = require('./request')
const { hasActiveVIP } = require('./vip-membership')
const { getAuthUser } = require('./storage')

const SHARE_GATE_UNLOCK_KEY = 'shareGateUnlockState'
const SHARE_GATE_TOKEN_KEY = 'shareGateUnlockTokenState'
const SHARE_GATE_UNLOCK_TTL = 24 * 60 * 60 * 1000
const SHARE_GATE_TOKEN_TTL = 10 * 60 * 1000

let cachedValue = {
  requireShareForAiReport: false,
  requireShareForCollegeMajor: false,
  requireShareForRecommendResult: false,
  requireShareForPlanCompare: false
}
let cacheExpiresAt = 0
let pendingPromise = null

function getShareGateUserScope() {
  const user = getAuthUser()
  if (!user) {
    return 'guest'
  }
  if (user.storageMode === 'server' && user.id) {
    return `server:${String(user.id)}`
  }
  if (user.phone) {
    return `phone:${String(user.phone)}`
  }
  if (user.openid) {
    return `openid:${String(user.openid)}`
  }
  return 'guest'
}

function buildUnlockStateKey(gateKey) {
  return `${getShareGateUserScope()}::${gateKey}`
}

function getStoredUnlockState() {
  return wx.getStorageSync(SHARE_GATE_UNLOCK_KEY) || {}
}

function saveStoredUnlockState(state) {
  wx.setStorageSync(SHARE_GATE_UNLOCK_KEY, state)
}

function getStoredTokenState() {
  return wx.getStorageSync(SHARE_GATE_TOKEN_KEY) || {}
}

function saveStoredTokenState(state) {
  wx.setStorageSync(SHARE_GATE_TOKEN_KEY, state)
}

function hasRecentShareGateUnlock(gateKey) {
  const state = getStoredUnlockState()
  const unlockAt = Number(state[buildUnlockStateKey(gateKey)] || 0)
  return unlockAt > 0 && (Date.now() - unlockAt) < SHARE_GATE_UNLOCK_TTL
}

function markShareGateUnlocked(gateKey) {
  const state = getStoredUnlockState()
  state[buildUnlockStateKey(gateKey)] = Date.now()
  saveStoredUnlockState(state)
}

function createShareGateToken(gateKey) {
  const token = `${Date.now()}_${Math.random().toString(36).slice(2, 10)}`
  const state = getStoredTokenState()
  state[buildUnlockStateKey(gateKey)] = {
    token,
    createdAt: Date.now()
  }
  saveStoredTokenState(state)
  return token
}

function hasMatchingShareGateToken(gateKey, token) {
  if (!token) {
    return false
  }
  const state = getStoredTokenState()
  const record = state[buildUnlockStateKey(gateKey)] || null
  if (!record || record.token !== token) {
    return false
  }
  return (Date.now() - Number(record.createdAt || 0)) < SHARE_GATE_TOKEN_TTL
}

function normalizeShareGateConfig(payload) {
  return {
    requireShareForAiReport: !!(payload && payload.requireShareForAiReport),
    requireShareForCollegeMajor: !!(payload && payload.requireShareForCollegeMajor),
    requireShareForRecommendResult: !!(payload && payload.requireShareForRecommendResult),
    requireShareForPlanCompare: !!(payload && payload.requireShareForPlanCompare)
  }
}

function getGateFlag(config, gateKey) {
  if (!config) {
    return false
  }
  if (gateKey === 'aiReport') {
    return !!config.requireShareForAiReport
  }
  if (gateKey === 'collegeMajor') {
    return !!config.requireShareForCollegeMajor
  }
  if (gateKey === 'recommendResult') {
    return !!config.requireShareForRecommendResult
  }
  if (gateKey === 'planCompare') {
    return !!config.requireShareForPlanCompare
  }
  return false
}

function getShareGateConfig(forceRefresh) {
  const now = Date.now()
  if (!forceRefresh && cacheExpiresAt > now) {
    return Promise.resolve(cachedValue)
  }
  if (pendingPromise) {
    return pendingPromise
  }
  pendingPromise = request({ url: '/api/share-gate-config', method: 'POST', data: {}, timeout: 5000 })
    .then((payload) => {
      cachedValue = normalizeShareGateConfig(payload)
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

function shouldRequireShareGate(gateKey, forceRefresh, shareToken) {
  if (hasMatchingShareGateToken(gateKey, shareToken)) {
    markShareGateUnlocked(gateKey)
    return Promise.resolve(false)
  }
  if (hasRecentShareGateUnlock(gateKey)) {
    return Promise.resolve(false)
  }
  return Promise.all([
    getShareGateConfig(forceRefresh),
    hasActiveVIP(forceRefresh)
  ]).then(([config, activeVIP]) => {
    if (activeVIP) {
      return false
    }
    return getGateFlag(config, gateKey)
  })
}

module.exports = {
	getShareGateConfig,
  shouldRequireShareGate,
  markShareGateUnlocked,
  createShareGateToken
}
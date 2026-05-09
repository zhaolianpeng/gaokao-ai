const HISTORY_KEY = 'recommendHistory'
const REPORT_KEY = 'reportHistory'
const FAVORITE_GROUP_KEY = 'favoriteProgramGroups'
const APPLICATION_PLAN_KEY = 'applicationPlan'
const PLAN_SCENARIO_KEY = 'planScenarios'
const AUTH_USER_KEY = 'authUser'
const USER_PROFILE_KEY = 'userProfile'
const NETWORK_DIAGNOSTIC_KEY = 'networkDiagnostics'
const PENDING_RECOMMEND_KEY = 'pendingRecommendPayload'
const PENDING_EXPLORE_SUBJECT_KEY = 'pendingExploreSubject'
const PENDING_EXPLORE_FILTERS_KEY = 'pendingExploreFilters'

function getList(key) {
  return wx.getStorageSync(key) || []
}

function saveList(key, list) {
  wx.setStorageSync(key, list)
}

function buildRecord(student, result) {
  const now = Date.now()
  return {
    id: String(now),
    createdAt: now,
    student,
    result
  }
}

function compactRecommendBucket(list) {
  return (Array.isArray(list) ? list : []).slice(0, 12).map((item) => ({
    college_id: item.college_id,
    college_name: item.college_name,
    group_code: item.group_code,
    group_name: item.group_name,
    batch: item.batch,
    subject_requirement: item.subject_requirement,
    matched_major: item.matched_major,
    major: item.major,
    plan_count: item.plan_count,
    major_count: item.major_count,
    min_rank: item.min_rank,
    min_score: item.min_score,
    probability: item.probability,
    recommendation_reason: item.recommendation_reason
  }))
}

function compactRecommendResult(result) {
  const payload = result || {}
  return {
    chong: compactRecommendBucket(payload.chong),
    wen: compactRecommendBucket(payload.wen),
    bao: compactRecommendBucket(payload.bao)
  }
}

function saveRecommendHistory(student, result) {
  const list = getList(HISTORY_KEY)
  const record = buildRecord(student, compactRecommendResult(result))
  const next = [record, ...list.filter((item) => item.student.rank !== student.rank || item.student.score !== student.score)].slice(0, 10)
  saveList(HISTORY_KEY, next)
  return record
}

function getRecommendHistory() {
  return getList(HISTORY_KEY)
}

function savePendingRecommendPayload(student, result) {
  const compactPayload = {
    student: student || {},
    result: compactRecommendResult(result),
    updatedAt: Date.now()
  }
  const payload = {
    student: student || {},
    result: result || {},
    updatedAt: Date.now()
  }
  try {
    const app = typeof getApp === 'function' ? getApp() : null
    if (app && app.globalData) {
      app.globalData.pendingRecommendPayload = compactPayload
    }
  } catch (err) {
  }
  try {
    wx.setStorageSync(PENDING_RECOMMEND_KEY, compactPayload)
  } catch (err) {
  }
  return compactPayload
}

function getPendingRecommendPayload() {
  try {
    const app = typeof getApp === 'function' ? getApp() : null
    if (app && app.globalData && app.globalData.pendingRecommendPayload) {
      return app.globalData.pendingRecommendPayload
    }
  } catch (err) {
  }
  const result = wx.getStorageSync(PENDING_RECOMMEND_KEY) || null
  if (!result) {
    return null
  }
  return result
}

function clearPendingRecommendPayload() {
  try {
    const app = typeof getApp === 'function' ? getApp() : null
    if (app && app.globalData) {
      delete app.globalData.pendingRecommendPayload
    }
  } catch (err) {
  }
  wx.removeStorageSync(PENDING_RECOMMEND_KEY)
}

function savePendingExploreSubject(subject) {
  const value = subject || '历史'
  try {
    wx.setStorageSync(PENDING_EXPLORE_SUBJECT_KEY, value)
  } catch (err) {
  }
  return value
}

function consumePendingExploreSubject() {
  const value = wx.getStorageSync(PENDING_EXPLORE_SUBJECT_KEY) || ''
  if (value) {
    wx.removeStorageSync(PENDING_EXPLORE_SUBJECT_KEY)
  }
  return value
}

function savePendingExploreFilters(payload) {
  const value = {
    subject: payload && payload.subject ? payload.subject : '历史',
    keyword: payload && payload.keyword ? payload.keyword : '',
    updatedAt: Date.now()
  }
  try {
    wx.setStorageSync(PENDING_EXPLORE_FILTERS_KEY, value)
  } catch (err) {
  }
  return value
}

function consumePendingExploreFilters() {
  const value = wx.getStorageSync(PENDING_EXPLORE_FILTERS_KEY) || null
  if (value) {
    wx.removeStorageSync(PENDING_EXPLORE_FILTERS_KEY)
  }
  return value
}

function saveReportHistory(payload) {
  const list = getList(REPORT_KEY)
  const next = [
    {
      id: String(Date.now()),
      createdAt: Date.now(),
      ...payload
    },
    ...list
  ].slice(0, 10)
  saveList(REPORT_KEY, next)
}

function getReportHistory() {
  return getList(REPORT_KEY)
}

function buildGroupKey(item) {
  return [
    item.college_id || 0,
    item.group_code || '',
    item.batch || '',
    item.subject_requirement || '',
    item.min_rank || 0
  ].join('::')
}

function buildProgramGroupRecord(student, item, source) {
  const now = Date.now()
  return {
    id: buildGroupKey(item),
    createdAt: now,
    source: source || 'result',
    student: student || {},
    item
  }
}

function compactScenarioItem(item, index) {
  return {
    college_id: item.college_id || 0,
    college_name: item.college_name || '',
    province: item.province || '',
    group_code: item.group_code || '',
    group_name: item.group_name || '',
    batch: item.batch || '',
    subject_requirement: item.subject_requirement || '',
    plan_count: item.plan_count || 0,
    major_count: item.major_count || 0,
    major: item.major || '',
    matched_major: item.matched_major || '',
    majorPreview: item.majorPreview || item.matched_major || item.major || '',
    recommendation_reason: item.recommendation_reason || '',
    min_score: item.min_score || 0,
    min_rank: item.min_rank || 0,
    avg_score: item.avg_score || 0,
    probability: item.probability || 0,
    probabilityText: item.probabilityText || '',
    groupLabel: item.groupLabel || '',
    tag: item.tag || 'other',
    target_hit: item.target_hit || 0,
    sortIndex: index || 0
  }
}

function buildScenarioMetrics(items) {
  const list = Array.isArray(items) ? items : []
  const metrics = {
    total: list.length,
    chong: 0,
    wen: 0,
    bao: 0,
    targetHits: 0,
    topColleges: []
  }
  const collegeMap = {}
  list.forEach((item) => {
    if (item.tag === 'chong') {
      metrics.chong += 1
    } else if (item.tag === 'wen') {
      metrics.wen += 1
    } else if (item.tag === 'bao') {
      metrics.bao += 1
    }
    if (item.matched_major || item.target_hit) {
      metrics.targetHits += 1
    }
    if (item.college_name && !collegeMap[item.college_name]) {
      collegeMap[item.college_name] = true
      metrics.topColleges.push(item.college_name)
    }
  })
  metrics.topColleges = metrics.topColleges.slice(0, 3)
  return metrics
}

function savePlanScenario(payload) {
  const list = getList(PLAN_SCENARIO_KEY)
  const student = payload && payload.student ? payload.student : {}
  const items = Array.isArray(payload && payload.items) ? payload.items : []
  const scenarioKey = [
    payload && payload.strategyKey ? payload.strategyKey : 'default',
    student.subject || '',
    student.score || 0,
    student.rank || 0
  ].join('::')
  const now = Date.now()
  const compactItems = items.slice(0, 12).map((item, index) => compactScenarioItem(item, index))
  const record = {
    id: scenarioKey,
    createdAt: now,
    updatedAt: now,
    strategyKey: payload && payload.strategyKey ? payload.strategyKey : 'default',
    title: (payload && payload.title) || '未命名方案',
    desc: (payload && payload.desc) || '',
    focus: (payload && payload.focus) || '',
    note: (payload && payload.note) || '',
    student,
    items: compactItems,
    metrics: buildScenarioMetrics(compactItems)
  }
  const next = [record, ...list.filter((entry) => entry.id !== scenarioKey)].slice(0, 12)
  saveList(PLAN_SCENARIO_KEY, next)
  return record
}

function getPlanScenarios() {
  return getList(PLAN_SCENARIO_KEY)
}

function removePlanScenario(id) {
  saveList(PLAN_SCENARIO_KEY, getList(PLAN_SCENARIO_KEY).filter((item) => item.id !== id))
}

function applyPlanScenario(id) {
  const scenarios = getList(PLAN_SCENARIO_KEY)
  const scenario = scenarios.find((item) => item.id === id)
  if (!scenario || !Array.isArray(scenario.items) || !scenario.items.length) {
    return 0
  }
  const list = scenario.items.map((item) => buildProgramGroupRecord(scenario.student || {}, item, `scenario:${scenario.strategyKey || 'default'}`))
  saveList(APPLICATION_PLAN_KEY, list.slice(0, 100))
  return list.length
}

function toggleFavoriteProgramGroup(student, item) {
  const list = getList(FAVORITE_GROUP_KEY)
  const record = buildProgramGroupRecord(student, item, 'favorite')
  const index = list.findIndex((entry) => entry.id === record.id)
  if (index >= 0) {
    saveList(FAVORITE_GROUP_KEY, list.filter((entry) => entry.id !== record.id))
    return false
  }
  saveList(FAVORITE_GROUP_KEY, [record, ...list].slice(0, 100))
  return true
}

function getFavoriteProgramGroups() {
  return getList(FAVORITE_GROUP_KEY)
}

function addApplicationPlanItem(student, item, source) {
  const list = getList(APPLICATION_PLAN_KEY)
  const record = buildProgramGroupRecord(student, item, source || 'direct')
  const next = [record, ...list.filter((entry) => entry.id !== record.id)].slice(0, 100)
  saveList(APPLICATION_PLAN_KEY, next)
  return record
}

function buildApplicationPlanFromFavorites(student) {
  const favorites = getFavoriteProgramGroups()
  if (!favorites.length) {
    return 0
  }
  const current = getList(APPLICATION_PLAN_KEY)
  const merged = [...current]
  favorites.forEach((favorite) => {
    const baseStudent = Object.keys(student || {}).length ? student : (favorite.student || {})
    const record = buildProgramGroupRecord(baseStudent, favorite.item, 'favorite')
    const index = merged.findIndex((entry) => entry.id === record.id)
    if (index >= 0) {
      merged.splice(index, 1)
    }
    merged.unshift(record)
  })
  saveList(APPLICATION_PLAN_KEY, merged.slice(0, 100))
  return favorites.length
}

function getApplicationPlan() {
  return getList(APPLICATION_PLAN_KEY)
}

function removeApplicationPlanItem(id) {
  saveList(APPLICATION_PLAN_KEY, getList(APPLICATION_PLAN_KEY).filter((item) => item.id !== id))
}

function clearApplicationPlan() {
  saveList(APPLICATION_PLAN_KEY, [])
}

function getAuthUser() {
  return wx.getStorageSync(AUTH_USER_KEY) || null
}

function saveAuthUser(user) {
  if (!user) {
    wx.removeStorageSync(AUTH_USER_KEY)
    return null
  }
  const loginType = user.loginType || 'wechat-phone'
  const storageMode = user.storageMode || 'server'
  const serverUserID = user.id === 0 ? '0' : (user.id ? String(user.id) : '')
  const payload = {
    id: storageMode === 'server' ? serverUserID : '',
    openid: user.openid || '',
    nickname: user.nickname || '考生用户',
    phone: user.phone || '',
    avatarUrl: user.avatarUrl || '',
    avatarLocalPath: user.avatarLocalPath || '',
    loginType,
    storageMode,
    created: user.created || Date.now(),
    loggedInAt: Date.now()
  }
  wx.setStorageSync(AUTH_USER_KEY, payload)
  return payload
}

function clearAuthUser() {
  wx.removeStorageSync(AUTH_USER_KEY)
}

function getUserProfile() {
  return wx.getStorageSync(USER_PROFILE_KEY) || null
}

function saveUserProfile(profile) {
  const current = getUserProfile() || {}
  const next = {
    ...current,
    ...(profile || {}),
    updatedAt: Date.now()
  }
  wx.setStorageSync(USER_PROFILE_KEY, next)
  return next
}

function clearUserProfile() {
  wx.removeStorageSync(USER_PROFILE_KEY)
}

function saveNetworkDiagnostic(payload) {
  const current = getList(NETWORK_DIAGNOSTIC_KEY)
  const record = {
    id: String(Date.now()),
    createdAt: Date.now(),
    ...payload
  }
  saveList(NETWORK_DIAGNOSTIC_KEY, [record, ...current].slice(0, 20))
  return record
}

function getNetworkDiagnostics() {
  return getList(NETWORK_DIAGNOSTIC_KEY)
}

function clearNetworkDiagnostics() {
  saveList(NETWORK_DIAGNOSTIC_KEY, [])
}

module.exports = {
  getRecommendHistory,
  saveRecommendHistory,
  savePendingRecommendPayload,
  getPendingRecommendPayload,
  clearPendingRecommendPayload,
  savePendingExploreSubject,
  consumePendingExploreSubject,
  savePendingExploreFilters,
  consumePendingExploreFilters,
  getReportHistory,
  saveReportHistory,
  toggleFavoriteProgramGroup,
  getFavoriteProgramGroups,
  addApplicationPlanItem,
  buildApplicationPlanFromFavorites,
  getApplicationPlan,
  removeApplicationPlanItem,
  clearApplicationPlan,
  savePlanScenario,
  getPlanScenarios,
  removePlanScenario,
  applyPlanScenario,
  getAuthUser,
  saveAuthUser,
  clearAuthUser,
  getUserProfile,
  saveUserProfile,
  clearUserProfile,
  saveNetworkDiagnostic,
  getNetworkDiagnostics,
  clearNetworkDiagnostics
}
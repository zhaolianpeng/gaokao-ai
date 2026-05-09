const https = require('https')
const cloud = require('wx-server-sdk')

cloud.init({ env: cloud.DYNAMIC_CURRENT_ENV })

const db = cloud.database()
const command = db.command

const COLLECTIONS = {
  COLLEGE: 'college',
  PROGRAM_GROUP: 'college_program_group',
  ENROLLMENT_PLAN: 'college_enrollment_plan',
  MAJOR: 'college_major',
  MAJOR_STAT: 'college_major_admission_stat',
  PROVINCE_SCORE_LINE: 'province_score_line',
  SCORE_RANK: 'score_rank',
  AGENT_TASK: 'agent_recommend_task'
}

const DEEPSEEK_SOFT_TIMEOUT_MS = 1800
const AGENT_TASK_STALE_MS = 5000

function nowMs() {
  return Date.now()
}

function getDeepSeekApiKey() {
  return process.env.DEEPSEEK_API_KEY || ''
}

function summarizePayload(payload) {
  if (!payload || typeof payload !== 'object') {
    return payload
  }
  const summary = {}
  Object.keys(payload).slice(0, 8).forEach((key) => {
    const value = payload[key]
    if (Array.isArray(value)) {
      summary[key] = `[array:${value.length}]`
      return
    }
    if (value && typeof value === 'object') {
      summary[key] = '[object]'
      return
    }
    summary[key] = value
  })
  return summary
}

function logStage(route, stage, extra = {}) {
  console.log(`[gaokaoApi][${route}][${stage}]`, JSON.stringify(extra))
}

exports.main = async (event) => {
  const startedAt = nowMs()
  try {
    const route = String(event.route || '').trim()
    const method = String(event.method || 'GET').toUpperCase()
    const query = event.query || {}
    const body = event.body || {}

    logStage(route || 'unknown', 'start', {
      method,
      query: summarizePayload(query),
      body: summarizePayload(body)
    })

    let data
    switch (route) {
      case '/api/dashboard/overview':
        data = await getDashboardOverview(query)
        break
      case '/api/province-lines':
        data = await getProvinceLines(query)
        break
      case '/api/score-rank':
        data = await lookupScoreRank(query)
        break
      case '/api/colleges':
        data = await listColleges(query)
        break
      case '/api/recommend':
        ensureMethod(method, 'POST')
        data = await recommend(body)
        break
      case '/api/analyze':
        ensureMethod(method, 'POST')
        data = await analyze(body)
        break
      case '/api/agent-recommend':
        ensureMethod(method, 'POST')
        data = await createAgentRecommendTask(body)
        break
      case '/api/agent-recommend/task':
        data = await getAgentRecommendTask(query)
        break
      default:
        if (/^\/api\/colleges\/\d+$/.test(route)) {
          data = await getCollegeDetail(route, query)
          break
        }
        throw new Error(`unsupported route: ${route}`)
    }

    logStage(route || 'unknown', 'end', { durationMs: nowMs() - startedAt })

    return { ok: true, data }
  } catch (error) {
    const normalizedError = normalizeCloudError(error)
    logStage(String(event.route || 'unknown').trim() || 'unknown', 'error', {
      durationMs: nowMs() - startedAt,
      message: normalizedError.message,
      stack: error && error.stack ? String(error.stack).split('\n').slice(0, 4) : []
    })
    return {
      ok: false,
      error: normalizedError.message,
      details: normalizedError.details
    }
  }
}

function normalizeCloudError(error) {
  const message = String((error && error.message) || 'cloud function error')
  const details = error && error.details ? error.details : null
  const collectionMatch = message.match(/Db or Table not exist:\s*([a-zA-Z0-9_]+)/i)
  if (collectionMatch) {
    const collectionName = collectionMatch[1]
    return {
      message: `云数据库集合不存在：${collectionName}。请先在云开发数据库中创建并导入该集合数据。`,
      details: details || { missingCollection: collectionName }
    }
  }
  return {
    message,
    details
  }
}

function ensureMethod(method, expected) {
  if (method !== expected) {
    throw new Error(`invalid method: ${method}`)
  }
}

function normalizeLookupSubject(year, subject) {
  const text = String(subject || '').trim()
  const numericYear = toNumber(year)
  if (numericYear > 0 && numericYear <= 2023) {
    if (text === '历史' || text === '文科' || text === '历史类') {
      return '文科'
    }
    if (text === '物理' || text === '理科' || text === '物理类') {
      return '理科'
    }
  }
  if (text === '历史' || text === '文科') {
    return '历史类'
  }
  if (text === '物理' || text === '理科') {
    return '物理类'
  }
  return text
}

function toNumber(value, fallback = 0) {
  const parsed = Number(value)
  return Number.isFinite(parsed) ? parsed : fallback
}

function unique(values) {
  return Array.from(new Set(values.filter((item) => item !== undefined && item !== null && item !== '')))
}

function uniqueObjects(items) {
  return Array.from(new Set((items || []).map((item) => JSON.stringify(item)))).map((item) => JSON.parse(item))
}

function chunk(list, size = 100) {
  const parts = []
  for (let index = 0; index < list.length; index += size) {
    parts.push(list.slice(index, index + size))
  }
  return parts
}

function minPositive(values) {
  const valid = values.map((item) => toNumber(item)).filter((item) => item > 0)
  return valid.length ? Math.min(...valid) : 0
}

function buildMajorReason(targetMajor, matchedMajor, minRank) {
  if (targetMajor && matchedMajor) {
    return '命中意向专业，优先保留该专业组'
  }
  if (minRank > 0) {
    return '按黑龙江 2025 专业组最低位次匹配'
  }
  return '当前专业组缺少有效位次，按计划与专业数保留为备选'
}

function estimateProbability(rankDiff) {
  if (rankDiff > 5000) {
    return 0.9
  }
  if (rankDiff > 2000) {
    return 0.7
  }
  if (rankDiff >= 0) {
    return 0.5
  }
  if (rankDiff >= -1000) {
    return 0.35
  }
  return 0.3
}

function buildRecommendBands(rank) {
  const base = Math.max(800, Math.min(5000, Math.round(rank * 0.12)))
  const wenUpper = Math.max(1200, Math.min(8000, Math.round(rank * 0.18)))
  const baoUpper = Math.max(2500, Math.min(15000, Math.round(rank * 0.35)))
  return {
    chongLower: -base,
    wenUpper,
    baoUpper
  }
}

function classifyByRankDiff(rankDiff, bands) {
  if (rankDiff < bands.chongLower) {
    return ''
  }
  if (rankDiff < 0) {
    return 'chong'
  }
  if (rankDiff <= bands.wenUpper) {
    return 'wen'
  }
  if (rankDiff <= bands.baoUpper) {
    return 'bao'
  }
  return ''
}

function absRankDiff(minRank, userRank) {
  return Math.abs(toNumber(minRank) - toNumber(userRank))
}

function trim(items, size) {
  return items.length <= size ? items : items.slice(0, size)
}

function includesKeyword(value, keyword) {
  if (!keyword) {
    return false
  }
  return String(value || '').toLowerCase().includes(keyword)
}

function escapeRegExp(value) {
  return String(value || '').replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
}

function normalizeAgentStudent(student) {
  const current = student || {}
  return {
    province: String(current.province || '黑龙江').trim() || '黑龙江',
    subject: String(current.subject || '历史').trim() || '历史',
    analysisYear: String(current.analysisYear || current.year || '2025').trim() || '2025',
    score: toNumber(current.score),
    rank: toNumber(current.rank),
    targetMajor: String(current.targetMajor || '').trim(),
    notes: String(current.notes || '').trim()
  }
}

function normalizeAgentTemplates(templates) {
  return unique((Array.isArray(templates) ? templates : []).map((item) => String(item || '').trim()).filter(Boolean))
}

function buildAgentExploreSuggestions(student, demand, templates) {
  const subject = student.subject || '历史'
  const templateKeywords = {
    '留省内': '黑龙江',
    '优先哈尔滨': '哈尔滨',
    '优先公办': '公办',
    '冲 211': '211',
    '偏计算机': '计算机'
  }
  const suggestions = []
  const pushSuggestion = (title, keyword) => {
    if (!keyword) {
      return
    }
    suggestions.push({
      id: `${title}-${keyword}`,
      title,
      keyword,
      subject
    })
  }

  if (student.targetMajor) {
    pushSuggestion('按意向专业筛选', student.targetMajor)
  }
  if (/哈尔滨/.test(student.notes)) {
    pushSuggestion('查看哈尔滨院校', '哈尔滨')
  }
  if (/哈尔滨/.test(demand)) {
    pushSuggestion('优先哈尔滨', '哈尔滨')
  }
  if (/计算机|软件|电子信息/.test(demand)) {
    pushSuggestion('查看计算机方向', '计算机')
  }
  templates.forEach((item) => {
    const keyword = templateKeywords[item]
    if (keyword) {
      pushSuggestion(item, keyword)
    }
  })

  return uniqueObjects(suggestions).slice(0, 5)
}

function buildAgentTaskPayload(body) {
  const student = normalizeAgentStudent(body.student)
  const demand = String(body.demand || '').trim()
  const templates = normalizeAgentTemplates(body.templates)
  if (!demand) {
    throw new Error('invalid demand')
  }
  return {
    title: 'AI 智能体报考建议',
    student,
    demand,
    templates,
    suggestions: buildAgentExploreSuggestions(student, demand, templates)
  }
}

async function loadAgentTask(taskId) {
  try {
    const res = await db.collection(COLLECTIONS.AGENT_TASK).doc(taskId).get()
    return res.data || null
  } catch (error) {
    if (String(error && error.message || '').includes('document.get:fail')) {
      return null
    }
    throw error
  }
}

function shouldProcessAgentTask(task) {
  if (!task) {
    return false
  }
  if (task.status === 'pending') {
    return true
  }
  if (task.status === 'processing' && nowMs() - toNumber(task.updatedAt) > AGENT_TASK_STALE_MS) {
    return true
  }
  return false
}

function formatAgentTask(task) {
  return {
    taskId: task._id,
    title: task.title || 'AI 智能体报考建议',
    status: task.status || 'pending',
    ready: task.status === 'succeeded',
    failed: task.status === 'failed',
    report: task.status === 'succeeded' ? String(task.report || '') : '',
    student: task.student || null,
    suggestions: Array.isArray(task.suggestions) ? task.suggestions : [],
    provider: String(task.provider || ''),
    errorMessage: String(task.errorMessage || ''),
    createdAt: toNumber(task.createdAt),
    updatedAt: toNumber(task.updatedAt),
    completedAt: toNumber(task.completedAt)
  }
}

async function createAgentRecommendTask(body) {
  const payload = buildAgentTaskPayload(body)
  const timestamp = nowMs()
  const result = await db.collection(COLLECTIONS.AGENT_TASK).add({
    data: {
      ...payload,
      status: 'pending',
      report: '',
      provider: '',
      errorMessage: '',
      createdAt: timestamp,
      updatedAt: timestamp,
      startedAt: 0,
      completedAt: 0,
      attemptCount: 0
    }
  })

  return {
    taskId: result._id,
    title: payload.title,
    status: 'pending',
    ready: false,
    createdAt: timestamp
  }
}

async function getAgentRecommendTask(query) {
  const taskId = String(query.taskId || '').trim()
  if (!taskId) {
    throw new Error('missing taskId')
  }

  let task = await loadAgentTask(taskId)
  if (!task) {
    throw new Error('agent task not found')
  }

  if (shouldProcessAgentTask(task)) {
    task = await processAgentTask(task)
  }

  return formatAgentTask(task)
}

async function processAgentTask(task) {
  const claimedAt = nowMs()
  const claimResult = await db.collection(COLLECTIONS.AGENT_TASK).where({
    _id: task._id,
    status: task.status
  }).update({
    data: {
      status: 'processing',
      updatedAt: claimedAt,
      startedAt: toNumber(task.startedAt) || claimedAt,
      attemptCount: command.inc(1),
      errorMessage: ''
    }
  })

  if (!claimResult.stats || !claimResult.stats.updated) {
    const latest = await loadAgentTask(task._id)
    return latest || task
  }

  try {
    const result = await generateAgentReport(task.student || {}, String(task.demand || ''), normalizeAgentTemplates(task.templates))
    const completedAt = nowMs()
    await db.collection(COLLECTIONS.AGENT_TASK).doc(task._id).update({
      data: {
        status: 'succeeded',
        report: result.report,
        provider: result.provider,
        updatedAt: completedAt,
        completedAt,
        errorMessage: ''
      }
    })
  } catch (error) {
    const failedAt = nowMs()
    await db.collection(COLLECTIONS.AGENT_TASK).doc(task._id).update({
      data: {
        status: 'failed',
        updatedAt: failedAt,
        completedAt: failedAt,
        errorMessage: String(error && error.message || 'agent task failed').slice(0, 500)
      }
    })
  }

  return await loadAgentTask(task._id)
}

function buildReason(req, item, rankDiff) {
  let reason = item.recommendation_reason || '按黑龙江专业组最低位次匹配'
  if (req.targetMajor && item.matched_major) {
    reason += `；命中意向专业：${item.matched_major}`
  }
  if (rankDiff < 0) {
    reason += '；当前定位偏冲刺'
  } else if (rankDiff <= buildRecommendBands(req.rank).wenUpper) {
    reason += '；当前定位偏稳妥'
  } else {
    reason += '；当前定位偏保底'
  }
  return reason
}

async function fetchAll(collectionName, where = {}, options = {}) {
  const limit = options.limit || 100
  const max = options.max || 5000
  const result = []
  let skip = 0
  while (result.length < max) {
    let query = db.collection(collectionName).where(where)
    ;(options.orderBy || []).forEach((item) => {
      query = query.orderBy(item.field, item.order)
    })
    const res = await query.skip(skip).limit(limit).get()
    const items = res.data || []
    result.push(...items)
    if (items.length < limit) {
      break
    }
    skip += items.length
  }
  return result.slice(0, max)
}

async function fetchByIds(collectionName, field, ids, extraWhere = {}) {
  const values = unique(ids)
  if (!values.length) {
    return []
  }
  const items = []
  for (const part of chunk(values, 100)) {
    const res = await db.collection(collectionName).where({
      ...extraWhere,
      [field]: command.in(part)
    }).get()
    items.push(...(res.data || []))
  }
  return items
}

function buildPlanFilter({ province, year, subject, collegeId }) {
  const where = {}
  if (province) {
    where.province = province
  }
  if (year) {
    where.year = year
  }
  if (subject) {
    where.subject = subject
  }
  if (collegeId) {
    where.college_id = collegeId
  }
  return where
}

async function fetchRegexMatches(collectionName, where, limit = 200) {
  const res = await db.collection(collectionName).where(where).limit(limit).get()
  return res.data || []
}

async function fetchCollegeKeywordContext(filter, keyword) {
  const escapedKeyword = escapeRegExp(keyword)
  const regex = db.RegExp({
    regexp: `.*${escapedKeyword}.*`,
    options: 'i'
  })
  const planBaseWhere = {
    province: filter.province,
    year: filter.year,
    subject: filter.subject
  }
  const planFields = ['major_name', 'major_full_name', 'major_remark', 'major_category']
  const planMap = new Map()
  for (const field of planFields) {
    const plans = await fetchRegexMatches(COLLECTIONS.ENROLLMENT_PLAN, {
      ...planBaseWhere,
      [field]: regex
    }, 300)
    plans.forEach((plan) => {
      planMap.set(String(plan.id), plan)
    })
  }

  const collegeMap = new Map()
  const collegeFields = ['name', 'city']
  for (const field of collegeFields) {
    const colleges = await fetchRegexMatches(COLLECTIONS.COLLEGE, {
      [field]: regex
    }, 150)
    colleges.forEach((college) => {
      collegeMap.set(String(college.id), college)
    })
  }

  const matchedCollegeIds = new Set([
    ...Array.from(planMap.values()).map((plan) => plan.college_id),
    ...Array.from(collegeMap.values()).map((college) => college.id)
  ])
  const planGroupIds = unique(Array.from(planMap.values()).map((plan) => plan.program_group_id)).filter((item) => toNumber(item) > 0)
  const groupsByPlan = planGroupIds.length
    ? await fetchByIds(COLLECTIONS.PROGRAM_GROUP, 'id', planGroupIds, planBaseWhere)
    : []
  const groupsByCollege = matchedCollegeIds.size
    ? await fetchByIds(COLLECTIONS.PROGRAM_GROUP, 'college_id', Array.from(matchedCollegeIds), planBaseWhere)
    : []

  const groupMap = new Map()
  const groupKeywordMap = new Map()
  ;[...groupsByPlan, ...groupsByCollege].forEach((group) => {
    groupMap.set(String(group.id), group)
    groupKeywordMap.set(group.id, includesKeyword(group.group_name, keyword) || includesKeyword(group.group_code, keyword))
  })

  return {
    groups: Array.from(groupMap.values()),
    groupKeywordMap,
    matchedCollegeIds,
    planCount: planMap.size
  }
}

function buildGroupedCollegeSummary(groups) {
  const grouped = new Map()
  groups.forEach((group) => {
    const key = String(group.college_id)
    const current = grouped.get(key) || {
      id: group.college_id,
      programGroupIds: new Set(),
      minGroupScoreValues: [],
      minGroupRankValues: [],
      totalPlanCount: 0
    }
    current.programGroupIds.add(group.id)
    if (toNumber(group.group_min_score) > 0) {
      current.minGroupScoreValues.push(group.group_min_score)
    }
    if (toNumber(group.group_min_rank) > 0) {
      current.minGroupRankValues.push(group.group_min_rank)
    }
    current.totalPlanCount += toNumber(group.group_plan_count)
    grouped.set(key, current)
  })
  return grouped
}

function sortCollegeItems(items) {
  items.sort((left, right) => {
    const leftRank = toNumber(left.min_group_rank) || Number.MAX_SAFE_INTEGER
    const rightRank = toNumber(right.min_group_rank) || Number.MAX_SAFE_INTEGER
    if (leftRank !== rightRank) {
      return leftRank - rightRank
    }
    if (toNumber(left.min_group_score) !== toNumber(right.min_group_score)) {
      return toNumber(right.min_group_score) - toNumber(left.min_group_score)
    }
    return toNumber(left.id) - toNumber(right.id)
  })
  return items
}

function buildCollegeItems(colleges, grouped, keyword, groupKeywordMap, matchedCollegeIds) {
  return colleges
    .filter((college) => {
      if (!keyword) {
        return true
      }
      if (includesKeyword(college.name, keyword) || includesKeyword(college.city, keyword)) {
        return true
      }

      const entry = grouped.get(String(college.id))
      const hasGroupHit = entry
        ? Array.from(entry.programGroupIds).some((groupId) => groupKeywordMap.get(groupId))
        : false

      return hasGroupHit || matchedCollegeIds.has(college.id)
    })
    .map((college) => {
      const entry = grouped.get(String(college.id)) || {
        programGroupIds: new Set(),
        minGroupScoreValues: [],
        minGroupRankValues: [],
        totalPlanCount: 0
      }
      return {
        id: college.id,
        name: college.name || '',
        province: college.province || '',
        city: college.city || '',
        level: college.level || '',
        tags: Array.isArray(college.tags) ? college.tags : [],
        school_level_tags: Array.isArray(college.school_level_tags) ? college.school_level_tags : [],
        recommended_postgraduate_rate: String(college.recommended_postgraduate_rate || ''),
        ranking: college.ranking || '',
        group_count: entry.programGroupIds.size,
        major_count: entry.totalPlanCount,
        min_group_score: minPositive(entry.minGroupScoreValues),
        min_group_rank: minPositive(entry.minGroupRankValues)
      }
    })
}

async function fallbackCollegeKeywordSearch(filter, keyword) {
  const plans = await fetchAll(COLLECTIONS.ENROLLMENT_PLAN, buildPlanFilter(filter), { max: 1200 })
  const matchedPlans = plans.filter((plan) => (
    includesKeyword(plan.major_name, keyword)
      || includesKeyword(plan.major_full_name, keyword)
      || includesKeyword(plan.major_remark, keyword)
      || includesKeyword(plan.major_category, keyword)
  ))
  if (!matchedPlans.length) {
    return { items: [], planCount: plans.length, groupCount: 0 }
  }

  const matchedCollegeIds = new Set(matchedPlans.map((item) => item.college_id))
  const programGroupIds = unique(matchedPlans.map((item) => item.program_group_id)).filter((item) => toNumber(item) > 0)
  const groups = programGroupIds.length
    ? await fetchByIds(COLLECTIONS.PROGRAM_GROUP, 'id', programGroupIds, {
      province: filter.province,
      year: filter.year,
      subject: filter.subject
    })
    : []
  const grouped = buildGroupedCollegeSummary(groups)
  const collegeIds = unique([
    ...matchedPlans.map((item) => item.college_id),
    ...groups.map((item) => item.college_id)
  ])
  const colleges = await fetchByIds(COLLECTIONS.COLLEGE, 'id', collegeIds)
  const items = sortCollegeItems(buildCollegeItems(colleges, grouped, keyword, new Map(), matchedCollegeIds))
  return {
    items,
    planCount: plans.length,
    groupCount: groups.length
  }
}

async function fetchRecommendRankedGroups(req, bands) {
  const where = {
    province: req.province,
    year: req.year,
    subject: req.subject
  }
  const batchSize = 100
  const maxGroups = 1600
  const bucketTargets = { chong: 40, wen: 70, bao: 70 }
  const rankedGroups = []
  const bucketed = { chong: [], wen: [], bao: [] }
  let skip = 0
  let scannedCount = 0
  let lastRank = 0

  while (scannedCount < maxGroups) {
    let query = db.collection(COLLECTIONS.PROGRAM_GROUP).where(where)
    query = query.orderBy('group_min_rank', 'asc').orderBy('group_min_score', 'desc')
    const res = await query.skip(skip).limit(batchSize).get()
    const batch = res.data || []
    if (!batch.length) {
      break
    }

    scannedCount += batch.length
    skip += batch.length
    lastRank = toNumber(batch[batch.length - 1].group_min_rank)

    batch.forEach((group) => {
      const minRank = toNumber(group.group_min_rank)
      if (minRank <= 0) {
        return
      }
      const diff = minRank - req.rank
      const tag = classifyByRankDiff(diff, bands)
      if (!tag) {
        return
      }
      const item = { group, diff, tag }
      rankedGroups.push(item)
      bucketed[tag].push(item)
    })

    if (batch.length < batchSize) {
      break
    }

    const enoughCandidates = bucketed.chong.length >= bucketTargets.chong
      && bucketed.wen.length >= bucketTargets.wen
      && bucketed.bao.length >= bucketTargets.bao
    const coveredSafeRange = lastRank > req.rank + bands.baoUpper
    if (enoughCandidates && coveredSafeRange) {
      break
    }
  }

  const sorter = (left, right) => {
    const rankDiff = absRankDiff(left.group.group_min_rank, req.rank) - absRankDiff(right.group.group_min_rank, req.rank)
    if (rankDiff !== 0) {
      return rankDiff
    }
    return toNumber(right.group.group_plan_count) - toNumber(left.group.group_plan_count)
  }

  rankedGroups.sort(sorter)
  Object.keys(bucketed).forEach((key) => bucketed[key].sort(sorter))

  return {
    rankedGroups,
    bucketed,
    scannedCount,
    lastRank
  }
}

async function getDashboardOverview(query) {
  const province = query.province || ''
  const year = toNumber(query.year, 2025)
  const subject = query.subject || ''
  const plans = await fetchAll(COLLECTIONS.ENROLLMENT_PLAN, buildPlanFilter({ province, year, subject }), { max: 6000 })
  const planIds = unique(plans.map((item) => item.id))
  const stats = planIds.length ? await fetchByIds(COLLECTIONS.MAJOR_STAT, 'enrollment_plan_id', planIds) : []
  const programGroupCount = unique(plans.map((item) => item.program_group_id)).filter((item) => toNumber(item) > 0).length
  return {
    province,
    year,
    subject,
    college_count: unique(plans.map((item) => item.college_id)).length,
    program_group_count: programGroupCount,
    enrollment_count: plans.length,
    major_count: unique(plans.map((item) => item.major_name)).length,
    stat_count: stats.length
  }
}

async function getProvinceLines(query) {
  const province = query.province || '黑龙江'
  const year = toNumber(query.year, 2025)
  const subject = normalizeLookupSubject(year, query.subject || '')
  const items = await fetchAll(COLLECTIONS.PROVINCE_SCORE_LINE, {
    province,
    year,
    subject
  }, { max: 300 })

  items.sort((left, right) => {
    if (left.subject !== right.subject) {
      return String(left.subject).localeCompare(String(right.subject), 'zh-Hans-CN')
    }
    if (toNumber(left.score) !== toNumber(right.score)) {
      return toNumber(right.score) - toNumber(left.score)
    }
    return String(left.batch || '').localeCompare(String(right.batch || ''), 'zh-Hans-CN')
  })
  return { items }
}

async function lookupScoreRank(query) {
  const province = query.province || '黑龙江'
  const year = toNumber(query.year, 2025)
  const subject = normalizeLookupSubject(year, query.subject || '')
  const score = toNumber(query.score)
  if (score <= 0) {
    throw new Error('invalid score')
  }
  if (!subject) {
    throw new Error('invalid subject')
  }

  const items = await fetchAll(COLLECTIONS.SCORE_RANK, { province, year, subject }, { max: 2000 })
  if (!items.length) {
    return {
      province,
      year,
      subject,
      query_score: score,
      matched_score: 0,
      rank: 0,
      count: 0,
      diff: 0,
      exact: false,
      available: false
    }
  }

  items.sort((left, right) => {
    const leftPriority = toNumber(left.score) === score ? 0 : (toNumber(left.score) < score ? 1 : 2)
    const rightPriority = toNumber(right.score) === score ? 0 : (toNumber(right.score) < score ? 1 : 2)
    if (leftPriority !== rightPriority) {
      return leftPriority - rightPriority
    }
    const diff = Math.abs(toNumber(left.score) - score) - Math.abs(toNumber(right.score) - score)
    if (diff !== 0) {
      return diff
    }
    return toNumber(right.score) - toNumber(left.score)
  })

  const best = items[0]
  return {
    province: best.province || province,
    year: toNumber(best.year, year),
    subject: best.subject || subject,
    query_score: score,
    matched_score: toNumber(best.score),
    rank: toNumber(best.rank),
    count: toNumber(best.count),
    diff: Math.abs(toNumber(best.score) - score),
    exact: toNumber(best.score) === score,
    available: true
  }
}

async function listColleges(query) {
  const stageStart = nowMs()
  const filter = {
    province: query.province || '黑龙江',
    year: toNumber(query.year, 2025),
    subject: query.subject || '',
    keyword: String(query.keyword || '').trim(),
    limit: Math.min(Math.max(toNumber(query.limit, 20), 1), 100)
  }
  const keyword = filter.keyword.toLowerCase()
  let groups = []
  let groupKeywordMap = new Map()
  let matchedCollegeIds = new Set()
  let planCount = 0

  if (keyword) {
    const keywordContext = await fetchCollegeKeywordContext(filter, keyword)
    groups = keywordContext.groups
    groupKeywordMap = keywordContext.groupKeywordMap
    matchedCollegeIds = keywordContext.matchedCollegeIds
    planCount = keywordContext.planCount
  } else {
    groups = await fetchAll(COLLECTIONS.PROGRAM_GROUP, {
      province: filter.province,
      year: filter.year,
      subject: filter.subject
    }, {
      max: 1200,
      orderBy: [
        { field: 'group_min_rank', order: 'asc' },
        { field: 'group_min_score', order: 'desc' }
      ]
    })
  }
  logStage('/api/colleges', 'groups-fetched', {
    durationMs: nowMs() - stageStart,
    groupCount: groups.length,
    keyword: filter.keyword
  })
  logStage('/api/colleges', 'plans-fetched', {
    durationMs: nowMs() - stageStart,
    planCount,
    keyword: filter.keyword
  })

  const grouped = buildGroupedCollegeSummary(groups)

  const collegeIds = Array.from(grouped.values()).map((item) => item.id)
  const colleges = await fetchByIds(COLLECTIONS.COLLEGE, 'id', collegeIds)
  let items = sortCollegeItems(buildCollegeItems(colleges, grouped, keyword, groupKeywordMap, matchedCollegeIds))

  if (keyword && !items.length) {
    const fallback = await fallbackCollegeKeywordSearch(filter, keyword)
    items = fallback.items
    planCount = Math.max(planCount, fallback.planCount)
    logStage('/api/colleges', 'keyword-fallback', {
      durationMs: nowMs() - stageStart,
      fallbackPlanCount: fallback.planCount,
      fallbackGroupCount: fallback.groupCount,
      fallbackItemCount: fallback.items.length,
      keyword: filter.keyword
    })
  }

  logStage('/api/colleges', 'assembled', {
    durationMs: nowMs() - stageStart,
    collegeCount: colleges.length,
    itemCount: items.length,
    limit: filter.limit
  })

  return { items: items.slice(0, filter.limit) }
}

async function getCollegeDetail(route, query) {
  const collegeId = toNumber(route.split('/').pop())
  if (collegeId <= 0) {
    throw new Error('invalid college id')
  }
  const province = query.province || '黑龙江'
  const year = toNumber(query.year, 2025)
  const subject = query.subject || ''

  const collegeList = await fetchAll(COLLECTIONS.COLLEGE, { id: collegeId }, { max: 1 })
  if (!collegeList.length) {
    throw new Error('college not found')
  }
  const college = collegeList[0]
  const programGroups = await fetchAll(COLLECTIONS.PROGRAM_GROUP, {
    college_id: collegeId,
    province,
    year,
    subject
  }, { max: 500 })
  const plans = await fetchAll(COLLECTIONS.ENROLLMENT_PLAN, buildPlanFilter({ province, year, subject, collegeId }), { max: 1000 })
  const programGroupMap = new Map(programGroups.map((item) => [item.id, item]))
  const majors = await fetchAll(COLLECTIONS.MAJOR, { college_id: collegeId }, { max: 1000 })
  const majorMap = new Map(majors.map((item) => [item.major_name, item]))
  const stats = plans.length ? await fetchByIds(COLLECTIONS.MAJOR_STAT, 'enrollment_plan_id', plans.map((item) => item.id)) : []
  const statMap = new Map()
  stats.forEach((item) => {
    const list = statMap.get(item.enrollment_plan_id) || []
    list.push({
      year: toNumber(item.stat_year),
      legacy_batch: item.legacy_batch || '',
      plan_count: toNumber(item.plan_count),
      admitted_count: toNumber(item.admitted_count),
      min_score: toNumber(item.min_score),
      min_rank: toNumber(item.min_rank),
      max_score: toNumber(item.max_score),
      max_rank: toNumber(item.max_rank)
    })
    statMap.set(item.enrollment_plan_id, list)
  })

  const historicalYears = unique(stats.map((item) => toNumber(item.stat_year))).sort((left, right) => right - left)

  const detail = {
    id: college.id,
    name: college.name || '',
    province: college.province || '',
    city: college.city || '',
    city_level: college.city_level || '',
    level: college.level || '',
    tags: Array.isArray(college.tags) ? college.tags : [],
    school_level_tags: Array.isArray(college.school_level_tags) ? college.school_level_tags : [],
    affiliation: college.affiliation || '',
    school_type: college.school_type || '',
    ownership_type: college.ownership_type || '',
    recommended_postgraduate_rate: String(college.recommended_postgraduate_rate || ''),
    ranking: college.ranking || '',
    transfer_policy: college.transfer_policy || '',
    admissions_charter_url: college.admissions_charter_url || '',
    softscience_grade: college.softscience_grade || '',
    softscience_ranking: college.softscience_ranking || '',
    discipline_evaluation: college.discipline_evaluation || '',
    master_major_count: toNumber(college.master_major_count),
    master_major_list: Array.isArray(college.master_major_list) ? college.master_major_list : [],
    doctor_major_count: toNumber(college.doctor_major_count),
    doctor_major_list: Array.isArray(college.doctor_major_list) ? college.doctor_major_list : [],
    program_groups: programGroups
      .map((item) => ({
        group_code: item.group_code || '',
        group_name: item.group_name || '',
        batch: item.batch || '',
        batch_remark: item.batch_remark || '',
        category: item.category || '',
        subject_requirement: item.subject_requirement || '',
        plan_count: toNumber(item.group_plan_count),
        min_score: toNumber(item.group_min_score),
        min_rank: toNumber(item.group_min_rank)
      }))
      .sort((left, right) => {
        const leftRank = toNumber(left.min_rank) || Number.MAX_SAFE_INTEGER
        const rightRank = toNumber(right.min_rank) || Number.MAX_SAFE_INTEGER
        if (leftRank !== rightRank) {
          return leftRank - rightRank
        }
        return String(left.group_code).localeCompare(String(right.group_code), 'zh-Hans-CN')
      }),
    major_plans: plans.map((plan) => {
      const group = programGroupMap.get(plan.program_group_id) || {}
      const major = majorMap.get(plan.major_name) || {}
      const admissionStats = (statMap.get(plan.id) || []).sort((left, right) => right.year - left.year)
      return {
        id: plan.id,
        major_code: plan.major_code || '',
        major_name: plan.major_name || '',
        major_full_name: plan.major_full_name || '',
        batch: plan.batch || '',
        batch_remark: plan.batch_remark || '',
        group_code: group.group_code || '',
        group_name: group.group_name || '',
        subject_requirement: group.subject_requirement || plan.subject_requirement || '',
        plan_count: toNumber(plan.plan_count),
        study_years: plan.study_years || '',
        tuition_fee: plan.tuition_fee || '',
        major_category: plan.major_category || '',
        discipline_category: plan.discipline_category || '',
        major_strength: major.major_strength || '',
        master_points: Array.isArray(major.master_points) ? major.master_points : [],
        doctor_points: Array.isArray(major.doctor_points) ? major.doctor_points : [],
        admission_stats: admissionStats
      }
    }),
    historical_stats_available: historicalYears
  }

  return detail
}

async function recommend(body) {
  const stageStart = nowMs()
  const req = {
    province: body.province || '黑龙江',
    score: toNumber(body.score),
    rank: toNumber(body.rank),
    subject: body.subject || '',
    year: toNumber(body.year, 2025),
    targetMajor: String(body.targetMajor || '').trim(),
    notes: body.notes || ''
  }
  if (!req.province || !req.subject || req.score <= 0 || req.rank <= 0) {
    throw new Error('invalid recommend payload')
  }

  const bands = buildRecommendBands(req.rank)
  const targetKeyword = req.targetMajor.toLowerCase()
  const rankedGroupContext = await fetchRecommendRankedGroups(req, bands)
  const groups = rankedGroupContext.rankedGroups.map((item) => item.group)
  logStage('/api/recommend', 'groups-fetched', {
    durationMs: nowMs() - stageStart,
    groupCount: groups.length,
    scannedCount: rankedGroupContext.scannedCount,
    lastRank: rankedGroupContext.lastRank,
    rank: req.rank,
    targetMajor: req.targetMajor
  })

  const rankedGroups = rankedGroupContext.rankedGroups
  const bucketedCandidates = rankedGroupContext.bucketed
  const candidateGroups = [
    ...trim(bucketedCandidates.chong, 30),
    ...trim(bucketedCandidates.wen, 50),
    ...trim(bucketedCandidates.bao, 50)
  ]
  logStage('/api/recommend', 'candidates-selected', {
    durationMs: nowMs() - stageStart,
    rankedCount: rankedGroups.length,
    candidateCount: candidateGroups.length,
    chongCount: bucketedCandidates.chong.length,
    wenCount: bucketedCandidates.wen.length,
    baoCount: bucketedCandidates.bao.length
  })

  const candidateGroupIds = candidateGroups.map((item) => item.group.id)
  const colleges = await fetchByIds(COLLECTIONS.COLLEGE, 'id', candidateGroups.map((item) => item.group.college_id))
  const collegeMap = new Map(colleges.map((item) => [item.id, item]))
  const plans = candidateGroupIds.length
    ? await fetchByIds(COLLECTIONS.ENROLLMENT_PLAN, 'program_group_id', candidateGroupIds, {
      province: req.province,
      year: req.year,
      subject: req.subject
    })
    : []
  logStage('/api/recommend', 'plans-fetched', {
    durationMs: nowMs() - stageStart,
    collegeCount: colleges.length,
    planCount: plans.length
  })
  const planMap = new Map()
  plans.forEach((plan) => {
    const list = planMap.get(plan.program_group_id) || []
    list.push(plan)
    planMap.set(plan.program_group_id, list)
  })

  const items = candidateGroups.map(({ group, diff, tag }) => {
    const college = collegeMap.get(group.college_id) || {}
    const relatedPlans = planMap.get(group.id) || []
    const majors = unique(relatedPlans.map((item) => item.major_name))
    const matchedMajor = targetKeyword
      ? (majors.find((name) => String(name || '').toLowerCase().includes(targetKeyword)) || '')
      : ''
    return {
      college_id: group.college_id,
      college_name: college.name || '',
      province: group.province || req.province,
      group_code: group.group_code || '',
      group_name: group.group_name || '',
      batch: group.batch || '',
      subject_requirement: group.subject_requirement || '不限',
      plan_count: toNumber(group.group_plan_count),
      major_count: majors.length,
      major: majors.slice(0, 8).join('、'),
      matched_major: matchedMajor,
      recommendation_reason: buildReason(req, {
        recommendation_reason: buildMajorReason(req.targetMajor, matchedMajor, toNumber(group.group_min_rank)),
        matched_major: matchedMajor
      }, diff),
      min_score: toNumber(group.group_min_score),
      min_rank: toNumber(group.group_min_rank),
      avg_score: toNumber(group.group_min_score),
      probability: estimateProbability(diff),
      tag,
      target_hit: matchedMajor ? 1 : 0
    }
  })

  items.sort((left, right) => {
    if (toNumber(right.target_hit) !== toNumber(left.target_hit)) {
      return toNumber(right.target_hit) - toNumber(left.target_hit)
    }
    const leftRank = toNumber(left.min_rank) || Number.MAX_SAFE_INTEGER
    const rightRank = toNumber(right.min_rank) || Number.MAX_SAFE_INTEGER
    if (leftRank !== rightRank) {
      return leftRank - rightRank
    }
    if (toNumber(left.min_score) !== toNumber(right.min_score)) {
      return toNumber(right.min_score) - toNumber(left.min_score)
    }
    if (toNumber(left.plan_count) !== toNumber(right.plan_count)) {
      return toNumber(right.plan_count) - toNumber(left.plan_count)
    }
    return toNumber(left.college_id) - toNumber(right.college_id)
  })

  const buckets = { chong: [], wen: [], bao: [] }
  items.forEach((item) => {
    if (toNumber(item.min_rank) <= 0) {
      return
    }
    const diff = toNumber(item.min_rank) - req.rank
    const tag = classifyByRankDiff(diff, bands)
    if (!tag) {
      return
    }
    const nextItem = {
      ...item,
      probability: estimateProbability(diff),
      recommendation_reason: buildReason(req, item, diff),
      tag
    }
    buckets[tag].push(nextItem)
  })

  Object.keys(buckets).forEach((key) => {
    buckets[key].sort((left, right) => absRankDiff(left.min_rank, req.rank) - absRankDiff(right.min_rank, req.rank))
  })

  logStage('/api/recommend', 'assembled', {
    durationMs: nowMs() - stageStart,
    chong: buckets.chong.length,
    wen: buckets.wen.length,
    bao: buckets.bao.length
  })

  return {
    chong: trim(buckets.chong, 10),
    wen: trim(buckets.wen, 20),
    bao: trim(buckets.bao, 20)
  }
}

async function analyze(body) {
  const stageStart = nowMs()
  const student = body.student || {}
  const recommendResult = body.recommend || {}
  const prompt = buildPrompt(student, recommendResult)
  logStage('/api/analyze', 'prompt-built', {
    durationMs: nowMs() - stageStart,
    promptLength: prompt.length
  })
  const apiKey = getDeepSeekApiKey()
  const baseUrl = (process.env.DEEPSEEK_BASE_URL || 'https://api.deepseek.com').replace(/\/$/, '')

  if (!apiKey) {
    return { report: `当前服务未配置 DEEPSEEK_API_KEY，先返回本地模板报告。\n\n${prompt}` }
  }

  const payload = JSON.stringify({
    model: 'deepseek-chat',
    messages: [
      { role: 'system', content: '你是中国高考志愿填报专家。' },
      { role: 'user', content: prompt }
    ],
    temperature: 0.3
  })

  const response = await requestJson(`${baseUrl}/chat/completions`, payload, {
    Accept: 'application/json',
    Authorization: `Bearer ${apiKey}`,
    'Content-Type': 'application/json'
  })
  logStage('/api/analyze', 'deepseek-returned', {
    durationMs: nowMs() - stageStart
  })
  const content = response && response.choices && response.choices[0] && response.choices[0].message
    ? String(response.choices[0].message.content || '').trim()
    : ''
  if (!content) {
    throw new Error('deepseek returned empty content')
  }
  return { report: content }
}

async function generateAgentReport(student, demand, templates) {
  const stageStart = nowMs()
  const prompt = buildAgentPrompt(student, demand, templates)
  logStage('/api/agent-recommend', 'prompt-built', {
    durationMs: nowMs() - stageStart,
    promptLength: prompt.length
  })

  const apiKey = getDeepSeekApiKey()
  const baseUrl = (process.env.DEEPSEEK_BASE_URL || 'https://api.deepseek.com').replace(/\/$/, '')
  if (!apiKey) {
    return { report: buildLocalAgentAdvice(student, demand, templates, 'missing_key'), provider: 'local' }
  }

  const payload = JSON.stringify({
    model: 'deepseek-chat',
    messages: [
      { role: 'system', content: '你是中国高考志愿填报智能体，擅长将考生需求转化为可执行的志愿策略。' },
      { role: 'user', content: prompt }
    ],
    temperature: 0.4
  })

  const response = await requestJson(`${baseUrl}/chat/completions`, payload, {
    Accept: 'application/json',
    Authorization: `Bearer ${apiKey}`,
    'Content-Type': 'application/json'
  }, DEEPSEEK_SOFT_TIMEOUT_MS).catch((error) => {
    logStage('/api/agent-recommend', 'deepseek-fallback', {
      durationMs: nowMs() - stageStart,
      reason: String(error && error.message ? error.message : error).slice(0, 120)
    })
    return null
  })
  if (!response) {
    return { report: buildLocalAgentAdvice(student, demand, templates, 'timeout'), provider: 'local-fallback' }
  }
  logStage('/api/agent-recommend', 'deepseek-returned', {
    durationMs: nowMs() - stageStart
  })
  const content = response && response.choices && response.choices[0] && response.choices[0].message
    ? String(response.choices[0].message.content || '').trim()
    : ''
  if (!content) {
    throw new Error('deepseek returned empty content')
  }
  return { report: content, provider: 'deepseek' }
}

function buildPrompt(student, recommendResult) {
  const formatList = (items = []) => {
    if (!items.length) {
      return '无'
    }
    return items.map((item) => (
      `- ${item.college_name} ${item.group_code || ''}${item.group_name || ''} | 批次:${item.batch || ''} | 选科:${item.subject_requirement || ''} | 计划:${item.plan_count || 0} | 组最低位次:${item.min_rank || 0} | 概率:${Math.round((item.probability || 0) * 100)}% | 组内专业:${item.major || ''} | 推荐理由:${item.recommendation_reason || ''}`
    )).join('\n')
  }

  return `你现在要为黑龙江考生生成 2025 年专业组口径的志愿填报建议。\n\n学生信息：\n省份：${student.province || '黑龙江'}\n分数：${student.score || 0}\n排名：${student.rank || 0}\n科类：${student.subject || ''}\n意向专业：${student.targetMajor || ''}\n补充偏好：${student.notes || ''}\n\n推荐专业组：\n\n冲刺组：\n${formatList(recommendResult.chong)}\n\n稳妥组：\n${formatList(recommendResult.wen)}\n\n保底组：\n${formatList(recommendResult.bao)}\n\n请输出一份黑龙江专版志愿报告，必须包含：\n1. 黑龙江当前分数/位次在所选科类中的总体判断\n2. 冲稳保三档专业组怎么排，为什么这样排\n3. 对意向专业的匹配度分析，哪些组最贴近目标专业\n4. 需要警惕的风险：位次倒挂、计划过少、是否接受调剂、组内专业跨度\n5. 一个可执行的正式填报策略，直接给出志愿梯度建议`
}

function buildAgentPrompt(student, demand, templates) {
  const filledInfo = [
    `省份：${student.province || '黑龙江'}`,
    `科类：${student.subject || '未填写'}`,
    `查询年份：${student.analysisYear || student.year || 2025}`,
    `分数：${student.score || '未填写'}`,
    `排名：${student.rank || '未填写'}`,
    `意向专业：${student.targetMajor || '未填写'}`,
    `补充偏好：${student.notes || '未填写'}`
  ].join('\n')
  const templateText = templates.length ? templates.map((item) => `- ${item}`).join('\n') : '无'

  return `你现在要作为黑龙江高考志愿智能体，为考生输出一份“需求驱动”的报考分析。\n\n以下是用户当前已经填写的信息，请全部纳入分析，不要忽略：\n${filledInfo}\n\n用户本次选择/使用的常用需求模板：\n${templateText}\n\n本次用户输入的核心需求：\n${demand}\n\n请直接输出一份可执行建议，必须包含：\n1. 先用 2-4 句话总结这位考生当前最适合的报考方向\n2. 从城市、学校层次、专业方向、调剂接受度四个维度拆解用户需求\n3. 给出“优先级排序建议”，说明哪些条件必须优先，哪些条件需要妥协\n4. 给出冲稳保三档策略，但用自然语言描述，不要求列具体学校名单\n5. 给出后续操作建议，明确下一步应该去院校库重点查什么，最好点出 3-5 个可检索关键词\n6. 如果用户需求本身互相冲突，要明确指出冲突点和取舍方式\n\n请用中文输出，结构清晰，避免空话套话。`
}

function buildLocalAgentAdvice(student, demand, templates, fallbackReason) {
  const text = `${student.targetMajor || ''} ${student.notes || ''} ${demand} ${(templates || []).join(' ')}`.toLowerCase()
  const wantsHarbin = /哈尔滨/.test(text)
  const wantsInsideProvince = /省内|黑龙江/.test(text)
  const wantsPublic = /公办/.test(text)
  const acceptsAdjustment = /调剂|接受调剂/.test(text)
  const wants211 = /211|双一流/.test(text)
  const majorDirection = student.targetMajor || (/计算机|软件|电子信息/.test(text) ? '计算机/电子信息方向' : '未明确具体专业方向')
  const scoreText = student.score ? `${student.score} 分` : '未填写分数'
  const rankText = student.rank ? `${student.rank} 名` : '未填写排名'
  const yearText = student.analysisYear || student.year || 2025
  const keywords = []

  if (wantsHarbin) keywords.push('哈尔滨')
  if (wantsPublic) keywords.push('公办')
  if (/计算机|软件/.test(text)) keywords.push('计算机')
  if (/电子信息/.test(text)) keywords.push('电子信息')
  if (wants211) keywords.push('211')
  if (!keywords.length && student.targetMajor) keywords.push(student.targetMajor)

  const uniqueKeywords = unique(keywords).slice(0, 5)
  const summary = wantsHarbin
    ? '当前需求明显偏向“城市优先”，哈尔滨应当被放在第一筛选层。'
    : '当前需求更适合先按学校层次和专业方向做第一轮筛选。'
  const publicSummary = wantsPublic
    ? '你已经明确偏向公办院校，这会直接压缩可选范围，但能提高结果稳定性。'
    : '你没有把公办设为硬条件，后续可以把学校层次和专业匹配放在更前面。'
  const adjustSummary = acceptsAdjustment
    ? '你接受组内调剂，这对保住城市或学校层次是有帮助的。'
    : '你没有明确接受调剂，后续需要重点审查专业组内专业跨度。'
  const conflict = wantsHarbin && wants211 && /历史/.test(String(student.subject || ''))
    ? '“优先哈尔滨”与“冲 211/双一流”同时成立时，历史类下可选面会明显变窄，需要在城市和层次之间做取舍。'
    : wantsHarbin && wantsPublic
      ? '“优先哈尔滨”与“优先公办”可以同时成立，但会抬高筛选门槛，热门专业可能需要接受专业让步。'
      : '当前需求没有绝对冲突，但城市、层次、专业三者不一定能同时最优，需要接受局部妥协。'

  const fallbackIntro = fallbackReason === 'missing_key'
    ? '以下内容为本地模板建议。当前服务未配置 DEEPSEEK_API_KEY，所以先返回一份结构化可执行分析。'
    : fallbackReason === 'timeout'
      ? '以下内容为本地模板建议。当前智能体生成时间较长，先返回一份可执行分析；你也可以稍后继续重试获取更细化结果。'
      : '以下内容为本地模板建议。当前先返回一份结构化可执行分析。'

  return [
    fallbackIntro,
    '',
    '## 一、总体判断',
    `黑龙江 ${student.subject || '未填写科类'} ${yearText} 口径下，你当前已填写的信息是：${scoreText}、${rankText}、意向专业 ${student.targetMajor || '未填写'}。`,
    summary,
    publicSummary,
    adjustSummary,
    '',
    '## 二、需求拆解',
    `城市维度：${wantsHarbin ? '优先哈尔滨，应先在院校库用“哈尔滨”做首轮筛选。' : wantsInsideProvince ? '优先黑龙江省内，可先看省内院校，再做城市细分。' : '城市没有形成硬约束，可作为第二优先级。'} `,
    `学校层次：${wants211 ? '有明显冲层次诉求，建议把 211/双一流作为冲刺方向，而不是全部志愿的统一标准。' : wantsPublic ? '优先公办是硬条件，应先排除高收费和民办项目。' : '学校层次可以作为筛选条件之一，但不必压过专业匹配。'} `,
    `专业方向：当前重点应放在 ${majorDirection}，同时留意相近替代方向，如软件工程、网络空间安全、电子信息类。`,
    `调剂接受度：${acceptsAdjustment ? '接受组内调剂，适合优先保城市或学校层次。' : '不接受调剂或未明确接受调剂，填报时要重点核查组内专业构成。'} `,
    '',
    '## 三、优先级排序建议',
    `第一优先级：${wantsHarbin ? '城市与办学属性' : '专业方向与学校层次'}`,
    `第二优先级：${wantsPublic ? '公办属性与学费可接受度' : '城市偏好与调剂接受度'}`,
    `第三优先级：${acceptsAdjustment ? '组内专业结构是否可接受' : '是否需要为了保专业而牺牲城市或层次'}`,
    '建议不要把所有条件都设成硬门槛，否则结果会过窄。',
    '',
    '## 四、冲稳保策略',
    '冲刺：把城市、层次、专业三者中最看重的两项锁死，第三项允许有限妥协，用来尝试更高层次目标。',
    '稳妥：优先保住最核心需求，例如哈尔滨 + 公办，或者公办 + 计算机方向，再在这个范围里找匹配度更高的组。',
    '保底：城市、层次、专业三项里至少放松一项，重点保可录取性和可接受的组内专业结构。',
    '',
    '## 五、下一步怎么查',
    '建议先去院校库逐个检索这些关键词，并对比组内专业结构、最低位次和是否接受调剂：',
    ...uniqueKeywords.map((item, index) => `${index + 1}. ${item}`),
    ...(uniqueKeywords.length ? [] : ['1. 哈尔滨', '2. 公办', '3. 计算机', '4. 电子信息']),
    '',
    '## 六、需求冲突与取舍',
    conflict,
    '如果后续配置好 DeepSeek Key，这一页会返回更细化、更贴近你输入语义的分析结果。'
  ].join('\n')
}

function requestJson(url, body, headers, timeoutMs) {
  return new Promise((resolve, reject) => {
    const target = new URL(url)
    const req = https.request({
      protocol: target.protocol,
      hostname: target.hostname,
      port: target.port || 443,
      path: `${target.pathname}${target.search}`,
      method: 'POST',
      headers: {
        ...headers,
        'Content-Length': Buffer.byteLength(body)
      },
      timeout: timeoutMs || 120000
    }, (res) => {
      const chunks = []
      res.on('data', (chunkData) => chunks.push(chunkData))
      res.on('end', () => {
        const text = Buffer.concat(chunks).toString('utf8').trim()
        if (!text) {
          reject(new Error('deepseek returned empty response body'))
          return
        }
        let parsed
        try {
          parsed = JSON.parse(text)
        } catch (error) {
          reject(new Error(`deepseek invalid json response: ${text.slice(0, 400)}`))
          return
        }
        if (res.statusCode >= 300) {
          const err = new Error(parsed.error && parsed.error.message ? parsed.error.message : `deepseek http status=${res.statusCode}`)
          err.details = parsed
          reject(err)
          return
        }
        if (parsed.error) {
          const err = new Error(parsed.error.message || 'deepseek api error')
          err.details = parsed.error
          reject(err)
          return
        }
        resolve(parsed)
      })
    })

    req.on('error', reject)
    req.on('timeout', () => req.destroy(new Error('deepseek request timeout')))
    req.write(body)
    req.end()
  })
}
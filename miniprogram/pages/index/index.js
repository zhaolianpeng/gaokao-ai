const { request } = require('../../utils/request')
const { getRecommendHistory, getFavoriteProgramGroups, getApplicationPlan, getUserProfile, getAuthUser, saveUserProfile, savePendingRecommendPayload, savePendingExploreSubject, saveHomeFormDraft, getHomeFormDraft } = require('../../utils/storage')
const { getVIPEntryVisibility } = require('../../utils/vip-entry')

function mergeFormWithDraft(form, draft) {
  if (!draft) {
    return form
  }
  return {
    ...form,
    province: draft.province || form.province,
    subject: draft.subject || form.subject,
    analysisYear: draft.analysisYear || form.analysisYear,
    year: draft.year || form.year,
    score: draft.score || form.score,
    rank: draft.rank || form.rank,
    targetMajor: draft.targetMajor || form.targetMajor,
    notes: draft.notes || form.notes,
    schoolName: draft.schoolName || form.schoolName,
    schoolYear: draft.schoolYear || form.schoolYear,
    className: draft.className || form.className,
    fromRecommend: typeof draft.fromRecommend === 'boolean' ? draft.fromRecommend : form.fromRecommend
  }
}

function buildFormFromProfile(form, profile) {
  if (!profile) {
    return form
  }
  return {
    ...form,
    province: profile.province || form.province,
    subject: profile.subject || form.subject,
    analysisYear: profile.analysisYear || form.analysisYear,
    year: profile.year || form.year,
    score: profile.score ? String(profile.score) : form.score,
    rank: profile.rank ? String(profile.rank) : form.rank,
    targetMajor: profile.targetMajor || form.targetMajor,
    notes: profile.notes || form.notes,
    schoolName: profile.schoolName || form.schoolName,
    schoolYear: profile.schoolYear || form.schoolYear,
    className: profile.className || form.className,
    fromRecommend: typeof profile.fromRecommend === 'boolean' ? profile.fromRecommend : form.fromRecommend
  }
}

function normalizeLookupSubject(subject) {
  return subject === '物理' ? '物理类' : '历史类'
}

function isLegacyMappedYear(year) {
  return year === '2022' || year === '2023'
}

function buildYearContext(year, subject) {
  const legacyMapped = isLegacyMappedYear(year)
  if (year === '2023') {
    return {
      mappingTone: 'info',
      mappingText: `${year} 年普通类按老高考文科/理科口径发布，前端会自动映射到 ${subject} 查询。`,
      reliabilityTone: 'positive',
      reliabilityText: `${year} 年普通类一分一段已补齐，命中同分值时展示精确位次。`
    }
  }
  if (year === '2022') {
    return {
      mappingTone: 'info',
      mappingText: `${year} 年按老高考文科/理科口径发布，前端会自动映射到 ${subject} 查询。`,
      reliabilityTone: 'positive',
      reliabilityText: `${year} 年当前已支持精确分值命中，未命中时才回退最近分段。`
    }
  }
  return {
    mappingTone: 'info',
    mappingText: `${year} 年按黑龙江 ${subject} 新高考口径直接查询。`,
    reliabilityTone: legacyMapped ? 'positive' : 'positive',
    reliabilityText: `${year} 年按实际批次线与一分一段展示，命中同分值时返回精确位次。`
  }
}

function buildRankDisplay(result, form) {
  if (!result || !result.available) {
    return null
  }
  const lookupType = result.lookup_type || 'score'
  const queryScore = Number(form.score || 0)
  const queryRank = Number(form.rank || 0)
  const diff = Number(result.diff || 0)
  const exact = !!result.exact
  const legacyMapped = isLegacyMappedYear(form.analysisYear)
  let status = 'exact'
  let message = lookupType === 'rank'
    ? `命中对应位次分段，当前分数人数 ${result.count}`
    : `命中同分段，当前分数人数 ${result.count}`

  if (exact && legacyMapped) {
    message = lookupType === 'rank'
      ? `命中精确位次，当前分数人数 ${result.count}。`
      : `命中精确分值，当前分数人数 ${result.count}。`
  }

  if (!exact) {
    if (diff <= 5) {
      status = 'near'
      message = lookupType === 'rank'
        ? `未命中精确位次，已回退到最近的 ${result.rank} 名分段。`
        : `未命中精确分值，已回退到最近的 ${result.matched_score} 分分段。`
    } else {
      status = 'approx'
      message = lookupType === 'rank'
        ? `当前仅命中 ${result.rank} 名分段，和输入排名相差 ${diff} 名，结果仅供参考。`
        : `当前仅命中 ${result.matched_score} 分分段，和输入分数相差 ${diff} 分，结果仅供参考。`
    }
  }

  if (!exact && legacyMapped) {
    message += ' 当前年份按老高考文科/理科口径自动映射。'
  }

  return {
    ...result,
    lookupType,
    queryScore,
    queryRank,
    status,
    message
  }
}

function buildBatchLineDashboard(items) {
  const list = Array.isArray(items) ? items : []
  const maxScore = list.reduce((currentMax, item) => Math.max(currentMax, Number(item.score || 0)), 0)
  return list.map((item, index) => ({
    ...item,
    trendWidth: maxScore > 0 ? Math.max(18, Math.round((Number(item.score || 0) / maxScore) * 100)) : 18,
    tone: index === 0 ? 'strong' : index === 1 ? 'mid' : 'light'
  }))
}

function buildRankDashboard(preview) {
  if (!preview || !preview.available) {
    return null
  }
  const confidence = preview.status === 'exact' ? 96 : preview.status === 'near' ? 72 : 48
  return {
    ...preview,
    confidence,
    confidenceLabel: preview.status === 'exact' ? '精确匹配' : preview.status === 'near' ? '近似匹配' : '参考匹配'
  }
}

function exceedsTolerance(inputValue, suggestedValue) {
  const current = Number(inputValue || 0)
  const suggested = Number(suggestedValue || 0)
  if (current <= 0 || suggested <= 0) {
    return false
  }
  return Math.abs(current - suggested) / suggested > 0.1
}

function buildToleranceToast(fieldLabel, suggestedValue) {
  return `${fieldLabel}调整已超过10%，当前建议值是 ${suggestedValue}`
}

function decodeQueryValue(value, fallback) {
  if (!value && value !== 0) {
    return fallback
  }
  return decodeURIComponent(value)
}

function decodeBooleanQueryValue(value, fallback) {
  if (value === undefined || value === null || value === '') {
    return fallback
  }
  var text = String(value).trim().toLowerCase()
  if (text === '1' || text === 'true' || text === 'yes') {
    return true
  }
  if (text === '0' || text === 'false' || text === 'no') {
    return false
  }
  return fallback
}

function enableShareMenus() {
  if (wx.showShareMenu) {
    wx.showShareMenu({ menus: ['shareAppMessage', 'shareTimeline'] })
  }
}

function buildHomeShareQuery(form) {
  var safeForm = form || {}
  var pairs = []
  var fields = ['subject', 'analysisYear', 'score', 'rank', 'targetMajor', 'notes', 'schoolName', 'schoolYear', 'className']

  for (var i = 0; i < fields.length; i += 1) {
    var key = fields[i]
    var value = safeForm[key]
    if (value || value === 0) {
      pairs.push(`${key}=${encodeURIComponent(String(value))}`)
    }
  }

  if (safeForm.fromRecommend) {
    pairs.push('fromRecommend=true')
  }

  return pairs.join('&')
}

function buildArchiveText(student) {
  var safeStudent = student || {}
  var parts = [safeStudent.schoolName, safeStudent.schoolYear, safeStudent.className].filter(Boolean)
  return parts.length ? parts.join(' / ') : ''
}

function buildHomeShareTitle(form) {
  var safeForm = form || {}
  if (safeForm.score && safeForm.rank) {
    return `黑龙江${safeForm.subject || ''}${safeForm.score}分 / ${safeForm.rank}名志愿方案，帮我一起看看`
  }
  if (safeForm.targetMajor) {
    return `我在看黑龙江${safeForm.targetMajor}志愿填报，帮我一起参谋`
  }
  return '黑龙江高考志愿填报助手：查位次、看专业组、出方案'
}

function shouldAutoLoadRemoteInsights(form) {
  var safeForm = form || {}
  return !!((safeForm.score && Number(safeForm.score) > 0) || (safeForm.rank && Number(safeForm.rank) > 0))
}

const CLOUD_DATASET_OVERVIEW = {
  college_count: 1520,
  program_group_count: 6174,
  enrollment_count: 24128,
  major_count: 17596,
  stat_count: 73059
}

const HOME_DECISION_FLOW = [
  {
    key: 'data',
    title: '先查数据',
    eta: '通常 1 秒内',
    desc: '先把批次线、一分一段、院校库和专业组看清楚，再决定要不要进入推荐。'
  },
  {
    key: 'recommend',
    title: '再出方案',
    eta: '通常 1-2 秒',
    desc: '智能推荐先给出冲稳保梯度，方便你和家长快速比学校、比专业、比城市。'
  },
  {
    key: 'ai',
    title: '最后做 AI 深挖',
    eta: '通常 60-120 秒',
    desc: '深度 AI 报告单独异步处理，适合做多轮讨论、风险解释和正式填报前复盘。'
  }
]

const HOME_SERVICE_LAYERS = [
  {
    label: '免费先用',
    badge: '零门槛',
    desc: '院校查询、专业组浏览、批次线、一分一段、冲稳保推荐、家长沟通摘要。'
  },
  {
    label: 'AI 深度分析',
    badge: '异步完成',
    desc: '更适合需要解释推荐原因、比较多套路径、补齐风险提示的考生。'
  },
  {
    label: 'VIP 高频决策',
    badge: '重度用户',
    desc: '适合需要长期保存多套方案、反复和家长老师沟通、集中填报的人。'
  }
]

Page({
  data: {
    loading: false,
    dashboardLoading: false,
    lineLoading: false,
    rankLoading: false,
    subjectOptions: ['历史', '物理'],
    yearOptions: ['2025', '2024', '2023', '2022'],
    historyPreview: [],
    favoriteCount: 0,
    applicationCount: 0,
    analysisContext: buildYearContext('2025', '历史'),
    dashboard: CLOUD_DATASET_OVERVIEW,
    batchLinePreview: [],
    batchLineDashboard: [],
    scoreRankPreview: null,
    scoreRankDashboard: null,
    suggestedScore: 0,
    suggestedRank: 0,
    decisionFlow: HOME_DECISION_FLOW,
    serviceLayers: HOME_SERVICE_LAYERS,
    showVipEntry: false,
    quickActions: [
      { key: 'explore', title: '院校库', desc: '查学校、专业组、招生计划', action: 'openExplorePage' },
      { key: 'province-lines', title: '黑龙江批次线', desc: '查看 2025-2022 批次线', action: 'openProvinceLinesPage' },
      { key: 'score-rank', title: '一分一段', desc: '按分数查询全省位次', action: 'openScoreRankPage' },
      { key: 'recommend', title: '智能推荐', desc: '按位次生成冲稳保方案', action: 'onRecommend' },
      { key: 'agent', title: 'AI 智能体', desc: '输入需求生成报考分析', action: 'openAiAgentPage' },
      { key: 'plan-list', title: '正式志愿表', desc: '查看已生成的填报清单', action: 'openPlanListPage' },
      { key: 'materials', title: '资料库', desc: '特殊类型与政策资料', action: 'openMaterialsPage' }
    ],
    form: {
      province: '黑龙江',
      subject: '历史',
      analysisYear: '2025',
      year: '2025',
      score: '',
      rank: '',
      targetMajor: '',
      notes: '',
      schoolName: '',
      schoolYear: '',
      className: '',
      fromRecommend: false
    }
  },

  onLoad(query) {
    this.syncVIPEntryVisibility(true)
    var safeQuery = query || {}
    var hasSharePayload = !!(safeQuery.subject || safeQuery.score || safeQuery.rank || safeQuery.targetMajor || safeQuery.notes || safeQuery.analysisYear || safeQuery.schoolName || safeQuery.schoolYear || safeQuery.className || safeQuery.fromRecommend)
    if (!hasSharePayload) {
      return
    }

    var nextForm = {
      ...this.data.form,
      subject: decodeQueryValue(safeQuery.subject, this.data.form.subject),
      analysisYear: decodeQueryValue(safeQuery.analysisYear, this.data.form.analysisYear),
      score: decodeQueryValue(safeQuery.score, this.data.form.score),
      rank: decodeQueryValue(safeQuery.rank, this.data.form.rank),
      targetMajor: decodeQueryValue(safeQuery.targetMajor, this.data.form.targetMajor),
      notes: decodeQueryValue(safeQuery.notes, this.data.form.notes),
      schoolName: decodeQueryValue(safeQuery.schoolName, this.data.form.schoolName),
      schoolYear: decodeQueryValue(safeQuery.schoolYear, this.data.form.schoolYear),
      className: decodeQueryValue(safeQuery.className, this.data.form.className),
      fromRecommend: decodeBooleanQueryValue(safeQuery.fromRecommend, this.data.form.fromRecommend)
    }

    this.setData({
      form: nextForm,
      analysisContext: buildYearContext(nextForm.analysisYear, nextForm.subject)
    })
  },

  onShow() {
    enableShareMenus()
    this.syncVIPEntryVisibility(false)
    const history = getRecommendHistory().slice(0, 3).map((item) => ({
      ...item,
      archiveText: buildArchiveText(item.student),
      fromRecommendText: item && item.student && item.student.fromRecommend ? '推荐来源' : ''
    }))
    const authUser = getAuthUser()
    const profile = getUserProfile()
    const homeDraft = getHomeFormDraft()
    const draftForm = mergeFormWithDraft(this.data.form, homeDraft)
    const nextForm = authUser ? buildFormFromProfile(draftForm, profile) : draftForm
    this.setData({
      historyPreview: history,
      favoriteCount: getFavoriteProgramGroups().length,
      applicationCount: getApplicationPlan().length,
      form: nextForm,
      analysisContext: buildYearContext(nextForm.analysisYear, nextForm.subject)
    })
    this.loadDashboard()
    this.loadBatchLines()
    if (nextForm.score && Number(nextForm.score) > 0) {
      this.lookupScoreRank()
      return
    }
    if (nextForm.rank && Number(nextForm.rank) > 0) {
      this.lookupRankScore()
      return
    }
    this.setData({
      scoreRankPreview: null,
      scoreRankDashboard: null,
      suggestedScore: 0,
      suggestedRank: 0,
      lineLoading: false,
      rankLoading: false
    })
  },

  persistHomeDraft(nextForm) {
    saveHomeFormDraft(nextForm)
  },

  syncVIPEntryVisibility(forceRefresh) {
  return getVIPEntryVisibility(forceRefresh).then((showVipEntry) => {
		if (this.data.showVipEntry !== showVipEntry) {
			this.setData({ showVipEntry })
		}
  }).catch(() => false)
  },

  onSubjectChange(e) {
    const value = this.data.subjectOptions[e.detail.value]
    const nextForm = {
      ...this.data.form,
      subject: value
    }
    this.setData({
      form: nextForm,
      analysisContext: buildYearContext(nextForm.analysisYear, value)
    })
    this.persistHomeDraft(nextForm)
    this.loadInsightData()
  },

  onAnalysisYearChange(e) {
    const value = this.data.yearOptions[e.detail.value]
    const nextForm = {
      ...this.data.form,
      analysisYear: value
    }
    this.setData({
      form: nextForm,
      analysisContext: buildYearContext(value, nextForm.subject)
    })
    this.persistHomeDraft(nextForm)
    this.loadInsightData()
  },

  onScoreInput(e) {
    const nextForm = {
      ...this.data.form,
      score: e.detail.value
    }
    this.setData({ form: nextForm })
    this.persistHomeDraft(nextForm)
    this.scheduleScoreRankLookup()
  },

  onScoreBlur(e) {
    const nextScore = e.detail && e.detail.value ? e.detail.value : this.data.form.score
    if (exceedsTolerance(nextScore, this.data.suggestedScore)) {
      wx.showToast({ title: buildToleranceToast('分数', this.data.suggestedScore), icon: 'none' })
    }
  },

  onRankInput(e) {
    const nextForm = {
      ...this.data.form,
      rank: e.detail.value
    }
    this.setData({ form: nextForm })
    this.persistHomeDraft(nextForm)
    this.scheduleRankScoreLookup()
  },

  onRankBlur(e) {
    const nextRank = e.detail && e.detail.value ? e.detail.value : this.data.form.rank
    if (exceedsTolerance(nextRank, this.data.suggestedRank)) {
      wx.showToast({ title: buildToleranceToast('排名', this.data.suggestedRank), icon: 'none' })
    }
  },

  onTargetMajorInput(e) {
    const nextForm = {
      ...this.data.form,
      targetMajor: e.detail.value
    }
    this.setData({ form: nextForm })
    this.persistHomeDraft(nextForm)
  },

  onNotesInput(e) {
    const nextForm = {
      ...this.data.form,
      notes: e.detail.value
    }
    this.setData({ form: nextForm })
    this.persistHomeDraft(nextForm)
  },

  openHistoryPage() {
    wx.navigateTo({ url: '/pages/history/history' })
  },

  openMaterialsPage() {
    wx.navigateTo({ url: '/pages/materials/materials' })
  },

  openAboutPage() {
    wx.navigateTo({ url: '/pages/about/about' })
  },

  openExplorePage() {
    const { subject } = this.data.form
    savePendingExploreSubject(subject)
    wx.switchTab({ url: '/pages/explore/explore' })
  },

  openProvinceLinesPage() {
    const { subject, analysisYear } = this.data.form
    wx.navigateTo({
      url: `/pages/province-lines/province-lines?subject=${encodeURIComponent(subject)}&year=${analysisYear}`
    })
  },

  openScoreRankPage() {
    const { subject, analysisYear, score } = this.data.form
    wx.navigateTo({
      url: `/pages/score-rank/score-rank?subject=${encodeURIComponent(subject)}&year=${analysisYear}&score=${encodeURIComponent(score || '')}`
    })
  },

  openPlanListPage() {
    wx.navigateTo({ url: '/pages/plan-list/plan-list' })
  },

  openAiAgentPage() {
    const form = this.data.form
    wx.navigateTo({
      url: `/pages/ai-agent/ai-agent?subject=${encodeURIComponent(form.subject)}&score=${encodeURIComponent(form.score || '')}&rank=${encodeURIComponent(form.rank || '')}&targetMajor=${encodeURIComponent(form.targetMajor || '')}&notes=${encodeURIComponent(form.notes || '')}&analysisYear=${encodeURIComponent(form.analysisYear || '2025')}&schoolName=${encodeURIComponent(form.schoolName || '')}&schoolYear=${encodeURIComponent(form.schoolYear || '')}&className=${encodeURIComponent(form.className || '')}&fromRecommend=${encodeURIComponent(form.fromRecommend ? 'true' : 'false')}`
    })
  },

  openVipPage() {
    wx.navigateTo({ url: '/pages/vip/vip' })
  },

  handleQuickAction(e) {
    const action = e.currentTarget.dataset.action
    if (!action || typeof this[action] !== 'function') {
      return
    }
    this[action]()
  },

  loadDashboard() {
    this.setData({
      dashboardLoading: false,
      dashboard: CLOUD_DATASET_OVERVIEW
    })
  },

  async loadBatchLines() {
    const { province, subject, analysisYear } = this.data.form
    this.setData({ lineLoading: true })
    try {
      const data = await request({
        url: '/api/province-lines',
        method: 'POST',
        data: {
          province,
          subject: normalizeLookupSubject(subject),
          year: Number(analysisYear)
        }
      })
      const batchLinePreview = (data.items || []).slice(0, 3)
      this.setData({
        batchLinePreview,
        batchLineDashboard: buildBatchLineDashboard(batchLinePreview)
      })
    } catch (err) {
      this.setData({ batchLinePreview: [], batchLineDashboard: [] })
    } finally {
      this.setData({ lineLoading: false })
    }
  },

  scheduleScoreRankLookup() {
    if (this.rankScoreTimer) {
      clearTimeout(this.rankScoreTimer)
    }
    if (this.scoreRankTimer) {
      clearTimeout(this.scoreRankTimer)
    }
    this.scoreRankTimer = setTimeout(() => this.lookupScoreRank(), 250)
  },

  scheduleRankScoreLookup() {
    if (this.scoreRankTimer) {
      clearTimeout(this.scoreRankTimer)
    }
    if (this.rankScoreTimer) {
      clearTimeout(this.rankScoreTimer)
    }
    this.rankScoreTimer = setTimeout(() => this.lookupRankScore(), 250)
  },

  async lookupScoreRank() {
    const { province, subject, analysisYear, score } = this.data.form
    if (!score || Number(score) <= 0) {
      this.setData({ scoreRankPreview: null, scoreRankDashboard: null, suggestedRank: 0 })
      return
    }
    this.setData({ rankLoading: true })
    try {
      const result = await request({
        url: '/api/score-rank',
        method: 'POST',
        data: {
          province,
          subject: normalizeLookupSubject(subject),
          year: Number(analysisYear),
          score: Number(score)
        }
      })
      const scoreRankPreview = buildRankDisplay(result, this.data.form)
      const nextRank = result && result.available && Number(result.rank || 0) > 0 ? String(result.rank) : this.data.form.rank
      this.setData({
        'form.rank': nextRank,
        suggestedRank: Number(result && result.rank) || 0,
        scoreRankPreview,
        scoreRankDashboard: buildRankDashboard(scoreRankPreview)
      })
    } catch (err) {
      this.setData({ scoreRankPreview: null, scoreRankDashboard: null, suggestedRank: 0 })
    } finally {
      this.setData({ rankLoading: false })
    }
  },

  async lookupRankScore() {
    const { province, subject, analysisYear, rank } = this.data.form
    if (!rank || Number(rank) <= 0) {
      this.setData({ scoreRankPreview: null, scoreRankDashboard: null, suggestedScore: 0 })
      return
    }
    this.setData({ rankLoading: true })
    try {
      const result = await request({
        url: '/api/rank-score',
        method: 'POST',
        data: {
          province,
          subject: normalizeLookupSubject(subject),
          year: Number(analysisYear),
          rank: Number(rank)
        }
      })
      const nextForm = {
        ...this.data.form,
        score: result && result.available && Number(result.matched_score || 0) > 0 ? String(result.matched_score) : this.data.form.score,
        rank: result && result.available && Number(result.rank || 0) > 0 ? String(result.rank) : this.data.form.rank
      }
      const scoreRankPreview = buildRankDisplay(result, nextForm)
      this.setData({
        'form.score': nextForm.score,
        'form.rank': nextForm.rank,
        suggestedScore: Number(result && result.matched_score) || 0,
        scoreRankPreview,
        scoreRankDashboard: buildRankDashboard(scoreRankPreview)
      })
    } catch (err) {
      this.setData({ scoreRankPreview: null, scoreRankDashboard: null, suggestedScore: 0 })
    } finally {
      this.setData({ rankLoading: false })
    }
  },

  loadInsightData() {
    this.loadDashboard()
    this.loadBatchLines()
    const { score, rank } = this.data.form
    if (score && Number(score) > 0) {
      this.lookupScoreRank()
      return
    }
    if (rank && Number(rank) > 0) {
      this.lookupRankScore()
      return
    }
    this.setData({ scoreRankPreview: null, scoreRankDashboard: null })
  },

  useHistory(e) {
    const record = this.data.historyPreview[e.currentTarget.dataset.index]
    if (!record) {
      return
    }
    const student = record.student || {}
    this.setData({
      form: {
        province: student.province || '黑龙江',
        subject: student.subject || '历史',
        analysisYear: String(student.year || 2025),
        year: '2025',
        score: String(student.score || ''),
        rank: String(student.rank || ''),
        targetMajor: student.targetMajor || '',
        notes: student.notes || '',
        schoolName: student.schoolName || '',
        schoolYear: student.schoolYear || '',
        className: student.className || '',
        fromRecommend: !!student.fromRecommend
      },
      analysisContext: buildYearContext(String(student.year || 2025), student.subject || '历史')
    })
    this.loadInsightData()
  },

  validateForm() {
    const { score, rank } = this.data.form
    const scoreValue = Number(score)
    const rankValue = Number(rank)
    if (!score || !rank) {
      return '请填写分数和排名'
    }
    if (Number.isNaN(scoreValue) || scoreValue <= 0 || scoreValue > 750) {
      return '分数范围不合法'
    }
    if (Number.isNaN(rankValue) || rankValue <= 0) {
      return '排名必须大于 0'
    }

    const scoreRankPreview = this.data.scoreRankPreview
    if (scoreRankPreview && scoreRankPreview.available) {
      const expectedRank = Number(scoreRankPreview.rank || 0)
      const tolerance = Math.max(300, Math.round(expectedRank * 0.12))
      if (expectedRank > 0 && Math.abs(rankValue - expectedRank) > tolerance) {
        return `当前分数参考位次约 ${expectedRank}，请先核对排名`
      }
    }

    return ''
  },

  async onRecommend() {
    const message = this.validateForm()
    if (message) {
      wx.showToast({ title: message, icon: 'none' })
      return
    }

    this.setData({ loading: true })
    try {
      const { province, subject, score, rank, year, targetMajor, notes, schoolName, schoolYear, className, fromRecommend } = this.data.form
      const payload = {
        province,
        subject,
        score: Number(score),
        rank: Number(rank),
        year: Number(year),
        targetMajor,
        notes,
        schoolName,
        schoolYear,
        className,
        fromRecommend: !!fromRecommend
      }
      const result = await request({
        url: '/api/recommend',
        method: 'POST',
        data: payload
      })

      if (getAuthUser()) {
        const profile = saveUserProfile({
          province,
          subject,
          analysisYear: this.data.form.analysisYear,
          year: String(year),
          score: Number(score),
          rank: Number(rank),
          targetMajor,
          notes,
          schoolName,
          schoolYear,
          className,
          fromRecommend: !!fromRecommend
        })
        getApp().setProfile(profile)
      }

      const pendingPayload = savePendingRecommendPayload(payload, result)
      wx.navigateTo({
        url: '/pages/recommend/recommend',
        success(res) {
          if (res && res.eventChannel) {
            res.eventChannel.emit('acceptRecommendPayload', pendingPayload)
          }
        }
      })
    } catch (err) {
      if (!err || !err.handledByModal) {
        wx.showToast({ title: (err && err.error) || '推荐失败', icon: 'none' })
      }
    } finally {
      this.setData({ loading: false })
    }
  },

  onShareAppMessage() {
    var query = buildHomeShareQuery(this.data.form)
    return {
      title: buildHomeShareTitle(this.data.form),
      path: '/pages/index/index' + (query ? `?${query}` : ''),
      imageUrl: ''
    }
  },

  onShareTimeline() {
    return {
      title: buildHomeShareTitle(this.data.form),
      query: buildHomeShareQuery(this.data.form),
      imageUrl: ''
    }
  }
})

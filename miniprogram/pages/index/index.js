const { request } = require('../../utils/request')
const { getRecommendHistory, getFavoriteProgramGroups, getApplicationPlan, getUserProfile, getAuthUser, saveUserProfile, savePendingRecommendPayload, savePendingExploreSubject } = require('../../utils/storage')

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
    notes: profile.notes || form.notes
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
  const queryScore = Number(form.score || 0)
  const diff = Number(result.diff || 0)
  const exact = !!result.exact
  const legacyMapped = isLegacyMappedYear(form.analysisYear)
  let status = 'exact'
  let message = `命中同分段，当前分数人数 ${result.count}`

  if (exact && legacyMapped) {
    message = `命中精确分值，当前分数人数 ${result.count}。`
  }

  if (!exact) {
    if (diff <= 5) {
      status = 'near'
      message = `未命中精确分值，已回退到最近的 ${result.matched_score} 分分段。`
    } else {
      status = 'approx'
      message = `当前仅命中 ${result.matched_score} 分分段，和输入分数相差 ${diff} 分，结果仅供参考。`
    }
  }

  if (!exact && legacyMapped) {
    message += ' 当前年份按老高考文科/理科口径自动映射。'
  }

  return {
    ...result,
    queryScore,
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

const CLOUD_DATASET_OVERVIEW = {
  college_count: 1520,
  program_group_count: 6174,
  enrollment_count: 24128,
  major_count: 17596,
  stat_count: 73059
}

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
      score: '580',
      rank: '7500',
      targetMajor: '计算机类',
      notes: ''
    }
  },

  onShow() {
    const history = getRecommendHistory().slice(0, 3)
    const authUser = getAuthUser()
    const profile = getUserProfile()
    const nextForm = authUser ? buildFormFromProfile(this.data.form, profile) : this.data.form
    this.setData({
      historyPreview: history,
      favoriteCount: getFavoriteProgramGroups().length,
      applicationCount: getApplicationPlan().length,
      form: nextForm,
      analysisContext: buildYearContext(nextForm.analysisYear, nextForm.subject)
    })
    this.loadInsightData()
  },

  onSubjectChange(e) {
    const value = this.data.subjectOptions[e.detail.value]
    this.setData({
      'form.subject': value,
      analysisContext: buildYearContext(this.data.form.analysisYear, value)
    })
    this.loadInsightData()
  },

  onAnalysisYearChange(e) {
    const value = this.data.yearOptions[e.detail.value]
    this.setData({
      'form.analysisYear': value,
      analysisContext: buildYearContext(value, this.data.form.subject)
    })
    this.loadInsightData()
  },

  onScoreInput(e) {
    this.setData({ 'form.score': e.detail.value })
    this.scheduleScoreRankLookup()
  },

  onRankInput(e) {
    this.setData({ 'form.rank': e.detail.value })
  },

  onTargetMajorInput(e) {
    this.setData({ 'form.targetMajor': e.detail.value })
  },

  onNotesInput(e) {
    this.setData({ 'form.notes': e.detail.value })
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
      url: `/pages/ai-agent/ai-agent?subject=${encodeURIComponent(form.subject)}&score=${encodeURIComponent(form.score || '')}&rank=${encodeURIComponent(form.rank || '')}&targetMajor=${encodeURIComponent(form.targetMajor || '')}&notes=${encodeURIComponent(form.notes || '')}&analysisYear=${encodeURIComponent(form.analysisYear || '2025')}`
    })
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
    if (this.scoreRankTimer) {
      clearTimeout(this.scoreRankTimer)
    }
    this.scoreRankTimer = setTimeout(() => this.lookupScoreRank(), 250)
  },

  async lookupScoreRank() {
    const { province, subject, analysisYear, score } = this.data.form
    if (!score || Number(score) <= 0) {
      this.setData({ scoreRankPreview: null })
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
      this.setData({
        scoreRankPreview,
        scoreRankDashboard: buildRankDashboard(scoreRankPreview)
      })
    } catch (err) {
      this.setData({ scoreRankPreview: null, scoreRankDashboard: null })
    } finally {
      this.setData({ rankLoading: false })
    }
  },

  loadInsightData() {
    this.loadDashboard()
    this.loadBatchLines()
    this.lookupScoreRank()
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
        notes: student.notes || ''
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
      const { province, subject, score, rank, year, targetMajor, notes } = this.data.form
      const payload = {
        province,
        subject,
        score: Number(score),
        rank: Number(rank),
        year: Number(year),
        targetMajor,
        notes
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
          notes
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
  }
})

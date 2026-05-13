const {
  getApplicationPlan,
  getPlanScenarios,
  getPendingRecommendPayload,
  getUserProfile,
  saveUserProfile,
  getHomeFormDraft,
  saveHomeFormDraft,
  savePendingRecommendPayload,
  savePlanScenario,
  clearApplicationPlan
} = require('../../utils/storage')
const { request } = require('../../utils/request')

const MODE_CONFIGS = [
  {
    key: 'balanced',
    title: '标准模拟',
    desc: '适合大多数家庭，兼顾冲稳保结构。',
    aggressiveMin: 0.25,
    aggressiveMax: 0.5,
    safeMin: 0.25,
    notePrefix: '模拟策略：标准平衡，兼顾录取结果和院校层次。'
  },
  {
    key: 'safe',
    title: '保守模拟',
    desc: '优先录取把握，要求保底更充足。',
    aggressiveMin: 0.15,
    aggressiveMax: 0.35,
    safeMin: 0.35,
    notePrefix: '模拟策略：保守填报，优先录取把握和保底充足。'
  },
  {
    key: 'aggressive',
    title: '冲层次模拟',
    desc: '更强调冲高层次，但仍保留底线。',
    aggressiveMin: 0.4,
    aggressiveMax: 0.65,
    safeMin: 0.2,
    notePrefix: '模拟策略：冲层次填报，适当增加冲刺院校占比。'
  }
]

function getModeConfig(modeKey) {
  for (let i = 0; i < MODE_CONFIGS.length; i += 1) {
    if (MODE_CONFIGS[i].key === modeKey) {
      return MODE_CONFIGS[i]
    }
  }
  return MODE_CONFIGS[0]
}

function formatTime(timestamp) {
  if (!timestamp) {
    return ''
  }
  const date = new Date(timestamp)
  const month = `${date.getMonth() + 1}`.padStart(2, '0')
  const day = `${date.getDate()}`.padStart(2, '0')
  const hour = `${date.getHours()}`.padStart(2, '0')
  const minute = `${date.getMinutes()}`.padStart(2, '0')
  return `${month}-${day} ${hour}:${minute}`
}

function matchesCurrentStudent(form, pendingStudent) {
  if (!pendingStudent || !form) {
    return false
  }
  return String(form.subject || '') === String(pendingStudent.subject || '')
    && String(form.score || '') === String(pendingStudent.score || '')
    && String(form.rank || '') === String(pendingStudent.rank || '')
}

function countByTag(list) {
  const counters = {
    chong: 0,
    jiaochong: 0,
    wen: 0,
    jiaobao: 0,
    bao: 0,
    other: 0
  }
  ;(Array.isArray(list) ? list : []).forEach((entry) => {
    const tag = entry && entry.item && entry.item.tag ? entry.item.tag : 'other'
    if (Object.prototype.hasOwnProperty.call(counters, tag)) {
      counters[tag] += 1
    } else {
      counters.other += 1
    }
  })
  return counters
}

function buildGradientWarnings(list) {
  const sorted = (Array.isArray(list) ? list : [])
    .filter((entry) => entry && entry.item && entry.item.min_rank)
    .slice()
    .sort((left, right) => Number(left.item.min_rank || 0) - Number(right.item.min_rank || 0))

  let crowdedPairs = 0
  for (let i = 1; i < sorted.length; i += 1) {
    const prevRank = Number(sorted[i - 1].item.min_rank || 0)
    const nextRank = Number(sorted[i].item.min_rank || 0)
    if (!prevRank || !nextRank) {
      continue
    }
    const threshold = Math.max(120, Math.round(prevRank * 0.03))
    if (Math.abs(nextRank - prevRank) < threshold) {
      crowdedPairs += 1
    }
  }
  return crowdedPairs
}

function buildDuplicateCollegeCount(list) {
  const collegeMap = {}
  ;(Array.isArray(list) ? list : []).forEach((entry) => {
    const name = entry && entry.item && entry.item.college_name ? entry.item.college_name : ''
    if (!name) {
      return
    }
    collegeMap[name] = (collegeMap[name] || 0) + 1
  })
  return Object.keys(collegeMap).reduce((count, key) => count + Math.max(0, collegeMap[key] - 1), 0)
}

function buildTargetHitCount(list, targetMajor) {
  const keyword = String(targetMajor || '').trim()
  if (!keyword) {
    return 0
  }
  return (Array.isArray(list) ? list : []).filter((entry) => {
    const item = (entry && entry.item) || {}
    const text = [item.matched_major, item.majorPreview, item.major].join(' ')
    return text.indexOf(keyword) >= 0 || Number(item.target_hit || 0) > 0
  }).length
}

function buildRiskSummary(list, modeKey, targetMajor) {
  const total = Array.isArray(list) ? list.length : 0
  const mode = getModeConfig(modeKey)
  const counts = countByTag(list)
  const aggressive = counts.chong + counts.jiaochong
  const safe = counts.jiaobao + counts.bao
  const duplicateCount = buildDuplicateCollegeCount(list)
  const crowdedPairs = buildGradientWarnings(list)
  const targetHits = buildTargetHitCount(list, targetMajor)
  const warnings = []
  let deduction = 0

  if (!total) {
    return {
      score: 0,
      level: '未开始',
      levelClass: 'pending',
      total: 0,
      counts,
      targetHits: 0,
      ratioText: '先生成推荐并加入志愿表，系统才会开始评估这一轮模拟。',
      warnings: ['当前还没有加入正式志愿表的院校专业组。'],
      suggestions: ['先生成一轮推荐，至少加入 8-12 个志愿再评估风险。']
    }
  }

  const aggressiveRatio = aggressive / total
  const safeRatio = safe / total
  if (aggressiveRatio < mode.aggressiveMin || aggressiveRatio > mode.aggressiveMax) {
    warnings.push('冲刺占比和当前模拟模式不匹配，需要重新平衡冲刺与稳妥数量。')
    deduction += 15
  }
  if (safeRatio < mode.safeMin) {
    warnings.push('较保和保底数量偏少，保底缓冲不够。')
    deduction += 18
  }
  if (counts.bao < Math.max(2, Math.round(total * 0.15))) {
    warnings.push('纯保底志愿偏少，建议至少补 2 个确定性更强的选项。')
    deduction += 12
  }
  if (counts.wen < Math.max(2, Math.round(total * 0.2))) {
    warnings.push('稳妥层数量偏少，正式落表时中段承接不够。')
    deduction += 10
  }
  if (crowdedPairs >= 3) {
    warnings.push('相邻志愿位次过于密集，梯度容易扎堆。')
    deduction += 10
  }
  if (duplicateCount >= 2) {
    warnings.push('同一院校出现过多，可能挤占志愿位次。')
    deduction += 8
  }

  let score = Math.max(0, 100 - deduction)
  let level = '优秀'
  let levelClass = 'excellent'
  if (score < 50) {
    level = '风险高'
    levelClass = 'danger'
  } else if (score < 70) {
    level = '需改进'
    levelClass = 'warning'
  } else if (score < 90) {
    level = '良好'
    levelClass = 'good'
  }

  const suggestions = []
  if (safeRatio < mode.safeMin) {
    suggestions.push('从较保、保底层补充学校，先保证录取下限。')
  }
  if (aggressiveRatio > mode.aggressiveMax) {
    suggestions.push('减少冲刺组数量，把部分学校替换到稳妥组。')
  }
  if (aggressiveRatio < mode.aggressiveMin) {
    suggestions.push('可以适度加入更高层次学校，避免整张表过于保守。')
  }
  if (crowdedPairs >= 3) {
    suggestions.push('查看正式志愿表，拉开相邻学校的最低位次差。')
  }
  if (!suggestions.length) {
    suggestions.push('结构基本合理，下一步重点检查专业方向是否符合目标。')
  }

  return {
    score,
    level,
    levelClass,
    total,
    counts,
    targetHits,
    ratioText: `冲刺 ${aggressive} / 稳妥 ${counts.wen} / 保底 ${safe}，当前为${getModeConfig(modeKey).title}结构。`,
    warnings: warnings.length ? warnings : ['当前结构基本合理，可以继续细调院校和专业顺序。'],
    suggestions
  }
}

function buildStepCards(form, pendingPayload, applicationList) {
  const hasScore = !!form.score && !!form.rank
  const hasPending = matchesCurrentStudent(form, pendingPayload && pendingPayload.student)
  const hasPlan = Array.isArray(applicationList) && applicationList.length > 0
  return [
    {
      key: 'fill',
      title: '1. 填模拟条件',
      desc: '先填分数、位次、意向专业和补充偏好。',
      statusText: hasScore ? '已完成' : '待填写',
      done: hasScore
    },
    {
      key: 'rank',
      title: '2. 核对位次',
      desc: '如果分数和位次还没对准，先去单独查位次。',
      statusText: form.rank ? '已填写位次' : '建议先核对',
      done: !!form.rank
    },
    {
      key: 'recommend',
      title: '3. 生成 5 层推荐',
      desc: '直接生成冲刺、较冲、稳妥、较保、保底五层方案。',
      statusText: hasPending ? '已有本轮结果' : '待生成',
      done: hasPending
    },
    {
      key: 'plan',
      title: '4. 加入正式志愿表',
      desc: '在推荐结果页把合适学校加入志愿表，再回来统一看整张表。',
      statusText: hasPlan ? `已加入 ${applicationList.length} 项` : '待加入',
      done: hasPlan
    }
  ]
}

function buildSimulationExportText(form, modeKey, applicationList, riskSummary, scenarioCount) {
  const mode = getModeConfig(modeKey)
  const lines = [
    `黑龙江 2025 模拟报考结果`,
    `模拟策略：${mode.title}`,
    `考生信息：${form.subject || '未填科类'} · ${form.score || '未填分数'}分 / ${form.rank || '未填位次'}名`
  ]
  if (form.targetMajor) {
    lines.push(`目标专业：${form.targetMajor}`)
  }
  if (form.notes) {
    lines.push(`补充偏好：${form.notes}`)
  }
  if (riskSummary && riskSummary.summaryText) {
    lines.push(`风险评估：${riskSummary.level} ${riskSummary.score || 0}分，${riskSummary.summaryText}`)
  }
  if (scenarioCount > 0) {
    lines.push(`当前已保存方案数：${scenarioCount}`)
  }
  lines.push('')
  ;(Array.isArray(applicationList) ? applicationList : []).forEach((entry, index) => {
    const item = (entry && entry.item) || {}
    const tagText = item.tag === 'chong' ? '冲刺' : item.tag === 'jiaochong' ? '较冲' : item.tag === 'wen' ? '稳妥' : item.tag === 'jiaobao' ? '较保' : item.tag === 'bao' ? '保底' : '待定'
    lines.push(`${index + 1}. [${tagText}] ${item.college_name || ''} ${item.group_code || ''} ${item.group_name || ''}`)
    lines.push(`   最低位次 ${item.min_rank || '无'} / 最低分 ${item.min_score || '无'} / 专业 ${item.majorPreview || item.matched_major || item.major || '未提供'}`)
  })
  if (!(applicationList && applicationList.length)) {
    lines.push('当前还没有加入正式志愿表的专业组。')
  }
  return lines.join('\n')
}

function defaultForm() {
  return {
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
}

function mergeFormFromDraftAndProfile() {
  const draft = getHomeFormDraft() || {}
  const profile = getUserProfile() || {}
  return {
    ...defaultForm(),
    ...draft,
    ...profile,
    province: '黑龙江',
    analysisYear: '2025',
    year: '2025'
  }
}

Page({
  data: {
    loading: false,
    subjectOptions: ['历史', '物理'],
    simulationMode: 'balanced',
    modeCards: MODE_CONFIGS,
    form: mergeFormFromDraftAndProfile(),
    applicationCount: 0,
    scenarioCount: 0,
    stepCards: [],
    riskSummary: buildRiskSummary([], 'balanced', ''),
    pendingResultText: '',
    hasPendingResult: false,
    nextActionText: '先填条件，开始第一轮模拟。'
  },

  refreshDerivedState(form, simulationMode, nextActionText) {
    const applicationList = getApplicationPlan()
    const scenarios = getPlanScenarios()
    const pendingPayload = getPendingRecommendPayload() || null
    const riskSummary = buildRiskSummary(applicationList, simulationMode, form.targetMajor)
    this.setData({
      form,
      simulationMode,
      applicationCount: applicationList.length,
      scenarioCount: scenarios.length,
      stepCards: buildStepCards(form, pendingPayload, applicationList),
      riskSummary,
      hasPendingResult: matchesCurrentStudent(form, pendingPayload && pendingPayload.student),
      pendingResultText: pendingPayload && pendingPayload.updatedAt ? `最近一次生成：${formatTime(pendingPayload.updatedAt)}` : '',
      nextActionText: nextActionText || (applicationList.length
        ? (riskSummary.levelClass === 'excellent' || riskSummary.levelClass === 'good' ? '结构已经成形，下一步去正式志愿表微调顺序。' : '先根据风险提示补足稳妥和保底，再看整张志愿表。')
        : (pendingPayload ? '推荐结果已经生成，去推荐页把合适院校加入正式志愿表。' : '先填条件，开始第一轮模拟。'))
    })
  },

  onShow() {
    const form = mergeFormFromDraftAndProfile()
    this.refreshDerivedState(form, this.data.simulationMode)
  },

  persistForm(nextForm) {
    saveHomeFormDraft(nextForm)
    saveUserProfile({
      ...getUserProfile(),
      ...nextForm,
      province: '黑龙江',
      analysisYear: '2025',
      year: '2025'
    })
  },

  onSubjectChange(e) {
    const subject = this.data.subjectOptions[e.detail.value]
    const nextForm = {
      ...this.data.form,
      subject
    }
    this.persistForm(nextForm)
    this.refreshDerivedState(nextForm, this.data.simulationMode)
  },

  onChooseMode(e) {
    const simulationMode = e.currentTarget.dataset.mode
    if (!simulationMode || simulationMode === this.data.simulationMode) {
      return
    }
    const applicationList = getApplicationPlan()
    const riskSummary = buildRiskSummary(applicationList, simulationMode, this.data.form.targetMajor)
    this.setData({ riskSummary })
    this.refreshDerivedState(
      this.data.form,
      simulationMode,
      applicationList.length
        ? (riskSummary.levelClass === 'excellent' || riskSummary.levelClass === 'good' ? '当前结构和所选模拟模式基本匹配。' : '当前志愿表和所选模式不完全匹配，建议按风险提示调整。')
        : '模式已切换，接下来生成一轮新的模拟推荐。'
    )
  },

  onInput(e) {
    const field = e.currentTarget.dataset.field
    const nextForm = {
      ...this.data.form,
      [field]: e.detail.value
    }
    this.persistForm(nextForm)
    this.refreshDerivedState(nextForm, this.data.simulationMode)
  },

  validateForm() {
    const { score, rank } = this.data.form
    const scoreValue = Number(score)
    const rankValue = Number(rank)
    if (!score || !rank) {
      return '请先填写分数和排名'
    }
    if (Number.isNaN(scoreValue) || scoreValue <= 0 || scoreValue > 750) {
      return '分数范围不合法'
    }
    if (Number.isNaN(rankValue) || rankValue <= 0) {
      return '排名必须大于 0'
    }
    return ''
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

  exportSimulationResult() {
    const applicationList = getApplicationPlan()
    const text = buildSimulationExportText(
      this.data.form,
      this.data.simulationMode,
      applicationList,
      this.data.riskSummary,
      this.data.scenarioCount
    )
    wx.setClipboardData({
      data: text,
      success: () => wx.showToast({ title: '已复制本轮模拟结果', icon: 'none' })
    })
  },

  saveCurrentSimulationScenario() {
    const applicationList = getApplicationPlan()
    if (!applicationList.length) {
      wx.showToast({ title: '请先把推荐院校加入正式志愿表', icon: 'none' })
      return
    }
    const mode = getModeConfig(this.data.simulationMode)
    savePlanScenario({
      strategyKey: `simulate-${this.data.simulationMode}`,
      title: `${mode.title}模拟方案`,
      desc: `${this.data.form.subject || '未填科类'} ${this.data.form.score || '--'}分 / ${this.data.form.rank || '--'}名`,
      focus: `顺序评估 ${this.data.riskSummary.level}`,
      note: this.data.riskSummary.summaryText,
      student: {
        ...this.data.form,
        score: Number(this.data.form.score || 0),
        rank: Number(this.data.form.rank || 0)
      },
      items: applicationList.map((entry) => entry.item)
    })
    this.refreshDerivedState(this.data.form, this.data.simulationMode, '已保存到方案对比库，接下来可以去正式志愿表页并排比较。')
    wx.showToast({ title: '已保存本轮模拟方案', icon: 'success' })
  },

  openPendingRecommendPage() {
    const pendingPayload = getPendingRecommendPayload() || null
    if (!pendingPayload || !pendingPayload.student) {
      wx.showToast({ title: '当前没有可继续查看的推荐结果', icon: 'none' })
      return
    }
    wx.navigateTo({
      url: '/pages/recommend/recommend',
      success(res) {
        if (res && res.eventChannel) {
          res.eventChannel.emit('acceptRecommendPayload', pendingPayload)
        }
      }
    })
  },

  clearSimulationPlan() {
    clearApplicationPlan()
    this.refreshDerivedState(this.data.form, this.data.simulationMode, '已清空这一轮模拟，建议重新生成推荐并重新落表。')
    wx.showToast({ title: '已清空当前模拟志愿表', icon: 'none' })
  },

  async startSimulation() {
    const message = this.validateForm()
    if (message) {
      wx.showToast({ title: message, icon: 'none' })
      return
    }

    this.setData({ loading: true })
    try {
      const { province, subject, score, rank, year, targetMajor, notes, schoolName, schoolYear, className, fromRecommend } = this.data.form
      const modeConfig = getModeConfig(this.data.simulationMode)
      const payload = {
        province,
        subject,
        score: Number(score),
        rank: Number(rank),
        year: Number(year),
        targetMajor,
        notes: [modeConfig.notePrefix, notes].filter(Boolean).join('；'),
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
        wx.showToast({ title: (err && err.error) || '模拟报考失败', icon: 'none' })
      }
    } finally {
      this.setData({ loading: false })
    }
  }
})
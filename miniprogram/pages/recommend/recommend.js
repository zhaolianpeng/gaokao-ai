const { request } = require('../../utils/request')
const { getVIPEntryVisibility } = require('../../utils/vip-entry')
const {
  saveRecommendHistory,
  getPendingRecommendPayload,
  toggleFavoriteProgramGroup,
  getFavoriteProgramGroups,
  addApplicationPlanItem,
  clearApplicationPlan,
  getApplicationPlan,
  buildApplicationPlanFromFavorites,
  savePlanScenario,
  getPlanScenarios
} = require('../../utils/storage')

const ANALYZE_IDLE_TEXT = 'AI 报告通常需要 60-120 秒。生成期间请保持当前页面开启，避免重复点击。'
const ANALYZE_PROGRESS_TEXTS = [
  '正在整理你的分数、位次和专业组结果。',
  '正在生成黑龙江志愿分析，通常还需要一点时间。',
  '正在补充填报建议和风险提示，请继续等待。'
]
const ANALYZE_HELPER_IDLE = '基础推荐先秒级给出，深度 AI 报告改为异步生成。你可以先继续浏览当前推荐，结果完成后会自动打开。'

function enableShareMenus() {
  if (wx.showShareMenu) {
    wx.showShareMenu({ menus: ['shareAppMessage', 'shareTimeline'] })
  }
}

function buildRecommendShareQuery(student) {
  var safeStudent = student || {}
  var pairs = []
  var fields = ['subject', 'score', 'rank', 'targetMajor', 'notes', 'schoolName', 'schoolYear', 'className']

  for (var i = 0; i < fields.length; i += 1) {
    var key = fields[i]
    var value = safeStudent[key]
    if (value || value === 0) {
      pairs.push(`${key}=${encodeURIComponent(String(value))}`)
    }
  }

  if (safeStudent.fromRecommend) {
    pairs.push('fromRecommend=true')
  }

  pairs.push('analysisYear=2025')
  return pairs.join('&')
}

function buildRecommendShareTitle(student) {
  var safeStudent = student || {}
  if (safeStudent.score && safeStudent.rank) {
    return `黑龙江${safeStudent.subject || ''}${safeStudent.score}分 / ${safeStudent.rank}名志愿方案已生成`
  }
  if (safeStudent.targetMajor) {
    return `黑龙江${safeStudent.targetMajor}志愿推荐已生成，帮我一起看看`
  }
  return '黑龙江志愿推荐结果已生成，帮我一起看看'
}

const LENS_OPTIONS = [
  { key: 'school', title: '保学校', desc: '先保学校层次和录取结果，适合先定梯度，再微调专业。' },
  { key: 'major', title: '保专业', desc: '优先保证目标专业和相近方向，再看是否接受组内调剂。' },
  { key: 'city', title: '保城市', desc: '先守住目标城市或省内偏好，再决定学校和专业取舍。' }
]

function decodeJsonQuery(value, fallback) {
  if (!value) {
    return fallback
  }
  try {
    return JSON.parse(decodeURIComponent(value))
  } catch (error) {
    return fallback
  }
}

function hasKeys(obj) {
  return !!obj && Object.keys(obj).length > 0
}

function cloneItem(item) {
  return {
    college_id: item.college_id || 0,
    college_name: item.college_name || '',
    province: item.province || '',
    city: item.city || '',
    group_code: item.group_code || '',
    group_name: item.group_name || '',
    batch: item.batch || '',
    subject_requirement: item.subject_requirement || '',
    plan_count: item.plan_count || 0,
    major_count: item.major_count || 0,
    major: item.major || '',
    matched_major: item.matched_major || '',
    recommendation_reason: item.recommendation_reason || '',
    min_score: item.min_score || 0,
    min_rank: item.min_rank || 0,
    avg_score: item.avg_score || 0,
    probability: item.probability || 0,
    tag: item.tag || '',
    target_hit: item.target_hit || 0,
    itemKey: [item.college_id || 0, item.group_code || '', item.batch || '', item.subject_requirement || '', item.min_rank || 0].join('::'),
    probabilityText: Math.round((item.probability || 0) * 100) + '%',
    groupLabel: ((item.group_code || '') + ' ' + (item.group_name || '')).trim(),
    majorPreview: item.matched_major || item.major || '未提供组内专业',
    planText: (item.plan_count || 0) + '人 / ' + (item.major_count || 0) + '专业',
    favoriteActive: false,
    favoriteClass: '',
    favoriteText: '收藏专业组'
  }
}

function normalizeBucket(items, defaultTag) {
  var list = Array.isArray(items) ? items : []
  var result = []
  for (var i = 0; i < list.length; i += 1) {
    var normalized = cloneItem(list[i] || {})
    if (!normalized.tag) {
      normalized.tag = defaultTag || 'other'
    }
    result.push(normalized)
  }
  return result
}

function normalizePayload(payload) {
  var safePayload = payload || {}
  return {
    student: safePayload.student || {},
    result: {
      chong: normalizeBucket(safePayload.result && safePayload.result.chong, 'chong'),
      wen: normalizeBucket(safePayload.result && safePayload.result.wen, 'wen'),
      bao: normalizeBucket(safePayload.result && safePayload.result.bao, 'bao')
    }
  }
}

function pushUniqueScenarioItem(target, item, seen) {
  if (!item || !item.itemKey || seen[item.itemKey]) {
    return
  }
  seen[item.itemKey] = true
  target.push(item)
}

function buildScenarioItems(strategyKey, result) {
  var items = []
  var seen = {}
  var chong = result.chong || []
  var wen = result.wen || []
  var bao = result.bao || []
  var i

  if (strategyKey === 'balanced') {
    for (i = 0; i < 1 && i < chong.length; i += 1) {
      pushUniqueScenarioItem(items, chong[i], seen)
    }
    for (i = 0; i < 3 && i < wen.length; i += 1) {
      pushUniqueScenarioItem(items, wen[i], seen)
    }
    for (i = 0; i < 2 && i < bao.length; i += 1) {
      pushUniqueScenarioItem(items, bao[i], seen)
    }
  } else if (strategyKey === 'major') {
    var source = [].concat(wen, chong, bao)
    for (i = 0; i < source.length; i += 1) {
      if (source[i].matched_major || source[i].target_hit) {
        pushUniqueScenarioItem(items, source[i], seen)
      }
      if (items.length >= 4) {
        break
      }
    }
    for (i = 0; i < 2 && i < wen.length; i += 1) {
      pushUniqueScenarioItem(items, wen[i], seen)
    }
    for (i = 0; i < 2 && i < bao.length; i += 1) {
      pushUniqueScenarioItem(items, bao[i], seen)
    }
  } else {
    for (i = 0; i < 3 && i < chong.length; i += 1) {
      pushUniqueScenarioItem(items, chong[i], seen)
    }
    for (i = 0; i < 2 && i < wen.length; i += 1) {
      pushUniqueScenarioItem(items, wen[i], seen)
    }
    for (i = 0; i < 1 && i < bao.length; i += 1) {
      pushUniqueScenarioItem(items, bao[i], seen)
    }
  }

  var fallback = [].concat(wen, chong, bao)
  for (i = 0; i < fallback.length && items.length < 6; i += 1) {
    pushUniqueScenarioItem(items, fallback[i], seen)
  }
  return items.slice(0, 6)
}

function buildAnalyzeStatusText(status, elapsedSeconds) {
  if (status === 'succeeded') {
    return 'AI 报告已经生成完成，正在为你打开结果页。'
  }
  if (status === 'failed') {
    return 'AI 报告生成失败，请稍后重试。'
  }
  if (status === 'processing') {
    if (elapsedSeconds >= 60) {
      return ANALYZE_PROGRESS_TEXTS[2]
    }
    if (elapsedSeconds >= 18) {
      return ANALYZE_PROGRESS_TEXTS[1]
    }
  }
  return ANALYZE_PROGRESS_TEXTS[0]
}

function buildAnalyzeProgressPercent(status, elapsedSeconds) {
  if (status === 'succeeded') {
    return 100
  }
  if (status === 'failed') {
    return 0
  }
  if (status === 'processing') {
    if (elapsedSeconds >= 60) {
      return 88
    }
    if (elapsedSeconds >= 18) {
      return 62
    }
    return 35
  }
  return 15
}

function getTopItem(list) {
  var items = Array.isArray(list) ? list : []
  return items.length ? items[0] : null
}

function buildStrategyCards(student, result) {
  var chongTop = getTopItem(result.chong)
  var wenTop = getTopItem(result.wen)
  var baoTop = getTopItem(result.bao)
  var majorMatches = []
  var source = [].concat(result.chong || [], result.wen || [], result.bao || [])
  for (var i = 0; i < source.length; i += 1) {
    if (source[i].matched_major || source[i].target_hit) {
      majorMatches.push(source[i])
    }
  }
  return [
    {
      key: 'balanced',
      title: '稳妥优先',
      desc: '先把主力志愿落在稳妥组，再用少量冲刺组抬上限。',
      focus: wenTop ? `${wenTop.college_name} ${wenTop.groupLabel}` : '优先从稳妥组前 3 所开始排表',
      note: '适合希望先保结果、再看层次提升的考生。'
    },
    {
      key: 'major',
      title: '专业优先',
      desc: '优先审查命中意向专业或相近方向的专业组，再决定是否接受组内调剂。',
      focus: majorMatches.length ? `${majorMatches[0].college_name} ${majorMatches[0].majorPreview}` : '当前推荐里没有明显命中专业，需要扩大相近方向搜索',
      note: `当前命中意向专业/相近方向 ${majorMatches.length} 组。`
    },
    {
      key: 'tier',
      title: '冲层次优先',
      desc: '把冲刺组当成抬学校层次的窗口，但要留出足够稳妥和保底仓位。',
      focus: chongTop ? `${chongTop.college_name} ${chongTop.groupLabel}` : '当前冲刺组较少，建议先稳住志愿梯度',
      note: baoTop ? `保底兜底建议至少保留 ${baoTop.college_name} 这一类选择。` : '保底组仍需补足。'
    }
  ]
}

function getLensMeta(activeLens) {
  for (var i = 0; i < LENS_OPTIONS.length; i += 1) {
    if (LENS_OPTIONS[i].key === activeLens) {
      return LENS_OPTIONS[i]
    }
  }
  return LENS_OPTIONS[0]
}

function findPreferredCity(student) {
  var text = `${student.notes || ''}\n${student.targetMajor || ''}`
  var namedCities = ['哈尔滨', '齐齐哈尔', '佳木斯', '大庆', '牡丹江', '北京', '上海', '广州', '深圳', '杭州', '南京', '武汉', '西安', '成都', '重庆', '天津', '长沙']
  for (var i = 0; i < namedCities.length; i += 1) {
    if (text.indexOf(namedCities[i]) >= 0) {
      return namedCities[i]
    }
  }
  var match = text.match(/([一-龥]{2,5})(?:市|地区|州)/)
  return match ? match[1] : ''
}

function mergeUniqueItems(result) {
  var merged = []
  var seen = {}
  var buckets = ['wen', 'chong', 'bao']
  for (var i = 0; i < buckets.length; i += 1) {
    var list = result[buckets[i]] || []
    for (var j = 0; j < list.length; j += 1) {
      pushUniqueScenarioItem(merged, list[j], seen)
    }
  }
  return merged
}

function sortByProbability(items) {
  return items.slice().sort(function(a, b) {
    if ((b.probability || 0) !== (a.probability || 0)) {
      return (b.probability || 0) - (a.probability || 0)
    }
    if ((a.min_rank || 0) !== (b.min_rank || 0)) {
      return (a.min_rank || 0) - (b.min_rank || 0)
    }
    return (b.target_hit || 0) - (a.target_hit || 0)
  })
}

function sortByMajorPriority(items) {
  return items.slice().sort(function(a, b) {
    var aScore = (a.matched_major ? 2 : 0) + (a.target_hit ? 1 : 0)
    var bScore = (b.matched_major ? 2 : 0) + (b.target_hit ? 1 : 0)
    if (bScore !== aScore) {
      return bScore - aScore
    }
    if ((b.probability || 0) !== (a.probability || 0)) {
      return (b.probability || 0) - (a.probability || 0)
    }
    return (a.min_rank || 0) - (b.min_rank || 0)
  })
}

function decorateSectionItems(items, favoriteMap) {
  var next = []
  for (var i = 0; i < items.length; i += 1) {
    var item = items[i]
    item.favoriteActive = !!favoriteMap[item.itemKey]
    item.favoriteClass = item.favoriteActive ? 'item-action-active' : ''
    item.favoriteText = item.favoriteActive ? '已收藏' : '收藏专业组'
    next.push(item)
  }
  return next
}

function createLensSection(config, items, favoriteMap) {
  var decorated = decorateSectionItems(items, favoriteMap)
  return {
    key: config.key,
    title: config.title,
    subtitle: config.subtitle,
    expanded: !!config.expanded,
    arrowText: config.expanded ? '收起' : '展开',
    itemCount: decorated.length,
    items: decorated
  }
}

function buildSchoolLensSections(result, favoriteMap) {
  var configs = [
    { key: 'school-chong', title: '可冲学校层次', subtitle: '优先看有机会抬层次的学校和专业组。', expanded: true, items: result.chong || [] },
    { key: 'school-wen', title: '主力学校池', subtitle: '作为主力填报区间，兼顾把握和学校层次。', expanded: true, items: result.wen || [] },
    { key: 'school-bao', title: '保底录取池', subtitle: '兜住录取结果，避免整张表整体过冲。', expanded: false, items: result.bao || [] }
  ]
  var sections = []
  for (var i = 0; i < configs.length; i += 1) {
    sections.push(createLensSection(configs[i], configs[i].items, favoriteMap))
  }
  return sections
}

function buildMajorLensSections(result, favoriteMap) {
  var all = mergeUniqueItems(result)
  var matched = []
  var related = []
  var backup = []
  for (var i = 0; i < all.length; i += 1) {
    var item = all[i]
    if (item.matched_major || item.target_hit) {
      matched.push(item)
    } else if (item.tag === 'bao') {
      backup.push(item)
    } else {
      related.push(item)
    }
  }
  return [
    createLensSection({ key: 'major-match', title: '优先保专业', subtitle: '优先保住意向专业或相近方向。', expanded: true }, sortByMajorPriority(matched), favoriteMap),
    createLensSection({ key: 'major-related', title: '相近专业备选', subtitle: '专业方向相近，但需要你进一步核查组内专业结构。', expanded: true }, sortByMajorPriority(related), favoriteMap),
    createLensSection({ key: 'major-backup', title: '保录取兜底', subtitle: '当专业命中不足时，用保底组先兜住录取。', expanded: false }, sortByMajorPriority(backup), favoriteMap)
  ]
}

function buildCityLensSections(student, result, favoriteMap) {
  var all = mergeUniqueItems(result)
  var preferredCity = findPreferredCity(student)
  var sameCity = []
  var sameProvince = []
  var outsideProvince = []
  for (var i = 0; i < all.length; i += 1) {
    var item = all[i]
    if (preferredCity && item.city && item.city.indexOf(preferredCity) >= 0) {
      sameCity.push(item)
    } else if ((item.province || '') === (student.province || '黑龙江')) {
      sameProvince.push(item)
    } else {
      outsideProvince.push(item)
    }
  }

  if (!preferredCity) {
    return [
      createLensSection({ key: 'city-province', title: '省内优先池', subtitle: '没有明确城市偏好时，先保省内学校。', expanded: true }, sortByProbability(sameProvince), favoriteMap),
      createLensSection({ key: 'city-expand', title: '外省拓展池', subtitle: '如果省内选择不够，再看外省成熟城市的学校。', expanded: true }, sortByProbability(outsideProvince), favoriteMap)
    ]
  }

  return [
    createLensSection({ key: 'city-target', title: `优先保 ${preferredCity}`, subtitle: `先筛出 ${preferredCity} 城市的学校和专业组。`, expanded: true }, sortByProbability(sameCity), favoriteMap),
    createLensSection({ key: 'city-province', title: '同省备选', subtitle: '如果目标城市不够，再从黑龙江省内学校补强。', expanded: true }, sortByProbability(sameProvince), favoriteMap),
    createLensSection({ key: 'city-outside', title: '跨城拓展', subtitle: '最后再看外省城市，平衡城市接受度和录取结果。', expanded: false }, sortByProbability(outsideProvince), favoriteMap)
  ]
}

function buildLensSections(activeLens, student, result, favoriteMap) {
  if (activeLens === 'major') {
    return buildMajorLensSections(result, favoriteMap)
  }
  if (activeLens === 'city') {
    return buildCityLensSections(student, result, favoriteMap)
  }
  return buildSchoolLensSections(result, favoriteMap)
}

function buildLensComparisonRows(sections) {
  var rows = []
  for (var i = 0; i < sections.length; i += 1) {
    var item = getTopItem(sections[i].items)
    if (!item) {
      continue
    }
    rows.push({
      label: sections[i].title,
      college: item.college_name,
      city: item.city || item.province || '城市待补充',
      groupLabel: item.groupLabel,
      majorPreview: item.majorPreview,
      probabilityText: item.probabilityText,
      rankText: item.min_rank ? `最低位次 ${item.min_rank}` : '最低位次待补充'
    })
  }
  return rows
}

function buildLensDecisionSteps(activeLens, student, sections) {
  var majorText = student.targetMajor || '你的目标专业'
  if (activeLens === 'major') {
    return [
      { title: '先锁定命中专业', desc: `先从“优先保专业”里找能真正覆盖 ${majorText} 的专业组。` },
      { title: '再看相近方向', desc: '如果完全命中的组不多，再看名称接近、培养方向相近的专业组。' },
      { title: '最后补保底', desc: '保底组只负责兜录取，不要用它来承担主要专业诉求。' },
      { title: '明确调剂边界', desc: '和家长先讲清楚哪些专业可以接受，哪些方向坚决不接受。' }
    ]
  }
  if (activeLens === 'city') {
    return [
      { title: '先定城市优先级', desc: '先明确是保目标城市、保省内，还是允许跨省换城市。' },
      { title: '再看城市里的学校', desc: '同一城市里再比较学校层次、组内专业和录取概率。' },
      { title: '给外省留备份', desc: '如果目标城市供给不足，要给外省城市留少量备选。' },
      { title: '最后核查通勤与成本', desc: '城市选择不仅是地理偏好，也要核查生活成本和家庭接受度。' }
    ]
  }
  return [
    { title: '先定学校梯度', desc: sections.length > 1 && sections[1].itemCount ? `先从“${sections[1].title}”里挑主力学校，稳住录取把握。` : '先从主力学校池里定好稳妥学校。' },
    { title: '再保专业不跑偏', desc: `围绕 ${majorText} 逐一核查组内专业，避免学校合适但专业读偏。` },
    { title: '最后留足保底', desc: '保底池至少保留 2-3 个接受度高的学校，避免整张表过冲。' },
    { title: '统一家庭决策口径', desc: '先把保学校、保专业、保城市的排序讲清楚，再排正式志愿表。' }
  ]
}

function buildLensFamilySummary(student, activeLens, sections) {
  var focusText = activeLens === 'major' ? '当前这版方案优先保专业。' : activeLens === 'city' ? '当前这版方案优先保城市。' : '当前这版方案优先保学校层次和录取结果。'
  var top = sections.length ? getTopItem(sections[0].items) : null
  var archiveText = [student.schoolName, student.schoolYear, student.className].filter(Boolean).join(' / ')
  var lines = [
    `黑龙江 ${student.subject || ''} 考生，分数 ${student.score || ''}，位次 ${student.rank || ''}。`,
    focusText,
    '正式填报前，建议家长和考生先统一保学校 / 保专业 / 保城市三者顺序。'
  ]
  if (archiveText) {
    lines.splice(1, 0, `档案上下文：${archiveText}。`)
  }
  if (student.fromRecommend) {
    lines.splice(archiveText ? 2 : 1, 0, '当前考生已标记为通过推荐链路进入。')
  }
  if (top) {
    lines.push(`当前优先讨论的专业组：${top.college_name}${top.city ? '（' + top.city + '）' : ''} ${top.groupLabel}。`)
  }
  return lines.join('\n')
}

Page({
  data: {
    loading: false,
    loadError: '',
    analyzingText: ANALYZE_IDLE_TEXT,
    analyzeButtonText: '生成黑龙江 AI 报考报告',
    student: {},
    result: { chong: [], wen: [], bao: [] },
    summary: [],
    sections: [],
    favoriteMap: {},
    applicationMap: {},
    favoriteCount: 0,
    applicationCount: 0,
    scenarioCount: 0,
    lensOptions: LENS_OPTIONS,
    activeLens: 'school',
    activeLensMeta: LENS_OPTIONS[0],
    topSummary: [],
    strategyCards: [],
    comparisonRows: [],
    decisionSteps: [],
    familySummary: '',
    analyzeTaskId: '',
    analyzeTaskStatus: '',
    analyzeProgressPercent: 0,
    analyzeHelperText: ANALYZE_HELPER_IDLE,
    showVipEntry: false
  },

  onLoad(query) {
    enableShareMenus()
    this.syncVIPEntryVisibility(true)
    var safeQuery = query || {}
    this.bindEventChannelPayload()
    try {
      var pendingPayload = getPendingRecommendPayload() || {}
      var directPayload = {
        student: decodeJsonQuery(safeQuery.student, pendingPayload.student || {}),
        result: decodeJsonQuery(safeQuery.result, pendingPayload.result || {})
      }
      this.applyPayload(directPayload)
    } catch (error) {
      this.setData({
        loadError: '推荐结果加载失败，请返回首页重新生成',
        result: { chong: [], wen: [], bao: [] },
        summary: [],
        sections: [],
        topSummary: []
      })
    }
  },

  onShow() {
    enableShareMenus()
    this.syncVIPEntryVisibility(true)
    if (!hasKeys(this.data.student)) {
      var pendingPayload = getPendingRecommendPayload() || null
      if (pendingPayload) {
        this.applyPayload(pendingPayload)
        return
      }
    }
    this.refreshCollections()
    if (this.data.analyzeTaskId && this.data.loading && this.data.analyzeTaskStatus !== 'failed') {
      this.startAnalyzePolling(this.data.analyzeTaskId)
    }
  },

  onUnload() {
    this.clearAnalyzeTimers()
    this.stopAnalyzePolling()
  },

  syncVIPEntryVisibility(forceRefresh) {
  return getVIPEntryVisibility(forceRefresh).then((showVipEntry) => {
		if (this.data.showVipEntry !== showVipEntry) {
			this.setData({ showVipEntry })
		}
	})
  },

  buildSummary(result) {
    return [
      { label: '冲刺组', value: (result.chong || []).length, type: 'chong' },
      { label: '稳妥组', value: (result.wen || []).length, type: 'wen' },
      { label: '保底组', value: (result.bao || []).length, type: 'bao' }
    ]
  },

  buildTopSummary(result) {
    function makeText(list) {
      var items = Array.isArray(list) ? list : []
      var names = []
      for (var i = 0; i < items.length && i < 2; i += 1) {
        names.push(items[i].college_name)
      }
      return names.join('、') || '当前暂无推荐'
    }

    return [
      { label: '优先看冲刺', text: makeText(result.chong) },
      { label: '主力填报', text: makeText(result.wen) },
      { label: '保底兜底', text: makeText(result.bao) }
    ]
  },

  applyViewModel() {
    var result = this.data.result || { chong: [], wen: [], bao: [] }
    var favoriteMap = this.data.favoriteMap || {}
    var student = this.data.student || {}
    var activeLens = this.data.activeLens || 'school'
    var sections = buildLensSections(activeLens, student, result, favoriteMap)
    this.setData({
      summary: this.buildSummary(result),
      topSummary: this.buildTopSummary(result),
      sections: sections,
      activeLensMeta: getLensMeta(activeLens),
      strategyCards: buildStrategyCards(student, result),
      comparisonRows: buildLensComparisonRows(sections),
      decisionSteps: buildLensDecisionSteps(activeLens, student, sections),
      familySummary: buildLensFamilySummary(student, activeLens, sections)
    })
  },

  bindEventChannelPayload() {
    try {
      if (typeof this.getOpenerEventChannel !== 'function') {
        return
      }
      var eventChannel = this.getOpenerEventChannel()
      if (!eventChannel || typeof eventChannel.on !== 'function') {
        return
      }
      var that = this
      eventChannel.on('acceptRecommendPayload', function(payload) {
        that.applyPayload(payload)
      })
    } catch (error) {
    }
  },

  applyPayload(payload) {
    var normalized = normalizePayload(payload)
    if (!hasKeys(normalized.student)) {
      this.setData({ loadError: '推荐结果已失效，请重新生成' })
      return
    }

    try {
      saveRecommendHistory(normalized.student, normalized.result)
    } catch (err) {
    }

    this.setData({
      loadError: '',
      student: normalized.student,
      result: normalized.result
    })
    this.refreshCollections()
  },

  openHistoryPage() {
    wx.navigateTo({ url: '/pages/history/history' })
  },

  openPlanList() {
    wx.navigateTo({ url: '/pages/plan-list/plan-list' })
  },

  openVip() {
    wx.navigateTo({ url: '/pages/vip/vip' })
  },

  onSwitchLens(e) {
    var lens = e.currentTarget.dataset.lens
    if (!lens || lens === this.data.activeLens) {
      return
    }
    this.setData({ activeLens: lens })
    this.applyViewModel()
  },

  onBuildPlanFromCurrentLens() {
    var sections = this.data.sections || []
    var student = this.data.student || {}
    var items = []
    var seen = {}

    for (var i = 0; i < sections.length; i += 1) {
      var sectionItems = sections[i].items || []
      for (var j = 0; j < sectionItems.length; j += 1) {
        var item = sectionItems[j]
        if (!item || !item.itemKey || seen[item.itemKey]) {
          continue
        }
        seen[item.itemKey] = true
        items.push(item)
      }
    }

    if (!items.length) {
      wx.showToast({ title: '当前视角没有可生成的志愿表', icon: 'none' })
      return
    }

    clearApplicationPlan()
    for (var index = items.length - 1; index >= 0; index -= 1) {
      addApplicationPlanItem(student, items[index], `lens:${this.data.activeLens || 'school'}`)
    }

    this.refreshCollections()
    wx.showToast({ title: `已按当前视角生成 ${items.length} 项`, icon: 'none' })
    wx.navigateTo({ url: '/pages/plan-list/plan-list' })
  },

  copyFamilySummary() {
    if (!this.data.familySummary) {
      return
    }
    wx.setClipboardData({
      data: this.data.familySummary,
      success: function() {
        wx.showToast({ title: '已复制家长沟通摘要', icon: 'success' })
      }
    })
  },

  refreshCollections() {
    var favorites = getFavoriteProgramGroups()
    var applications = getApplicationPlan()
    var scenarios = getPlanScenarios()
    var favoriteMap = {}
    var applicationMap = {}
    var i

    for (i = 0; i < favorites.length; i += 1) {
      favoriteMap[favorites[i].id] = true
    }
    for (i = 0; i < applications.length; i += 1) {
      applicationMap[applications[i].id] = true
    }

    this.setData({
      favoriteMap: favoriteMap,
      applicationMap: applicationMap,
      favoriteCount: favorites.length,
      applicationCount: applications.length,
      scenarioCount: scenarios.length
    })
    this.applyViewModel()
  },

  saveStrategyScenario(e) {
    var strategyKey = e.currentTarget.dataset.strategyKey
    var strategyCards = this.data.strategyCards || []
    var selectedCard = null
    var i
    for (i = 0; i < strategyCards.length; i += 1) {
      if (strategyCards[i].key === strategyKey) {
        selectedCard = strategyCards[i]
        break
      }
    }
    if (!selectedCard) {
      return
    }
    var items = buildScenarioItems(strategyKey, this.data.result || {})
    if (!items.length) {
      wx.showToast({ title: '当前没有可保存的方案', icon: 'none' })
      return
    }
    savePlanScenario({
      strategyKey: strategyKey,
      title: selectedCard.title,
      desc: selectedCard.desc,
      focus: selectedCard.focus,
      note: selectedCard.note,
      student: this.data.student,
      items: items
    })
    this.refreshCollections()
    wx.showToast({ title: '已保存到方案对比库', icon: 'success' })
  },

  getItemByDataset(e) {
    var itemKey = e.currentTarget.dataset.itemKey
    var buckets = ['chong', 'wen', 'bao']
    var result = this.data.result || {}
    var i
    var j

    for (i = 0; i < buckets.length; i += 1) {
      var list = result[buckets[i]] || []
      for (j = 0; j < list.length; j += 1) {
        if (list[j].itemKey === itemKey) {
          return list[j]
        }
      }
    }
    return null
  },

  onToggleSection(e) {
    var key = e.currentTarget.dataset.key
    var sections = this.data.sections || []
    var next = []

    for (var i = 0; i < sections.length; i += 1) {
      var section = sections[i]
      if (section.key === key) {
        next.push({
          key: section.key,
          title: section.title,
          subtitle: section.subtitle,
          expanded: !section.expanded,
          arrowText: section.expanded ? '展开' : '收起',
          itemCount: section.itemCount,
          items: section.items
        })
      } else {
        next.push(section)
      }
    }
    this.setData({ sections: next })
  },

  onToggleFavorite(e) {
    var item = this.getItemByDataset(e)
    if (!item) {
      return
    }
    var favorited = toggleFavoriteProgramGroup(this.data.student, item)
    this.refreshCollections()
    wx.showToast({ title: favorited ? '已收藏专业组' : '已取消收藏', icon: 'none' })
  },

  onAddToPlan(e) {
    var item = this.getItemByDataset(e)
    if (!item) {
      return
    }
    addApplicationPlanItem(this.data.student, item, 'direct')
    this.refreshCollections()
    wx.showToast({ title: '已加入正式志愿表', icon: 'none' })
  },

  onBuildPlanFromFavorites() {
    var count = buildApplicationPlanFromFavorites(this.data.student)
    if (!count) {
      wx.showToast({ title: '请先收藏专业组', icon: 'none' })
      return
    }
    this.refreshCollections()
    wx.navigateTo({ url: '/pages/plan-list/plan-list' })
  },

  clearAnalyzeTimers() {
    if (!this.analyzeTimerIds || !this.analyzeTimerIds.length) {
      return
    }
    for (var i = 0; i < this.analyzeTimerIds.length; i += 1) {
      clearTimeout(this.analyzeTimerIds[i])
    }
    this.analyzeTimerIds = []
  },

  stopAnalyzePolling() {
    if (this.analyzePollTimer) {
      clearInterval(this.analyzePollTimer)
      this.analyzePollTimer = null
    }
    this.analyzePollingRequest = false
  },

  startAnalyzeProgress(taskId, status) {
    this.clearAnalyzeTimers()
    this.stopAnalyzePolling()
    this.analyzeStartedAt = Date.now()
    this.setData({
      loading: true,
      analyzeTaskId: taskId || '',
      analyzeTaskStatus: status || 'pending',
      analyzingText: ANALYZE_PROGRESS_TEXTS[0],
      analyzeButtonText: 'AI 深度分析生成中',
      analyzeProgressPercent: 15,
      analyzeHelperText: '已切换到异步生成模式。你可以继续查看推荐结果，系统会自动轮询并在完成后跳转。'
    })
  },

  finishAnalyzeProgress(statusText) {
    this.clearAnalyzeTimers()
    this.stopAnalyzePolling()
    this.setData({
      loading: false,
      analyzeTaskId: '',
      analyzeTaskStatus: '',
      analyzingText: statusText || ANALYZE_IDLE_TEXT,
      analyzeButtonText: '生成黑龙江 AI 报考报告',
      analyzeProgressPercent: 0,
      analyzeHelperText: ANALYZE_HELPER_IDLE
    })
  },

  startAnalyzePolling(taskId) {
    var that = this
    if (!taskId) {
      return
    }
    this.stopAnalyzePolling()
    this.pollAnalyzeTask(taskId)
    this.analyzePollTimer = setInterval(function() {
      that.pollAnalyzeTask(taskId)
    }, 1500)
  },

  async pollAnalyzeTask(taskId) {
    if (!taskId || this.analyzePollingRequest) {
      return
    }
    this.analyzePollingRequest = true
    try {
      var data = await request({
        url: '/api/analyze/task',
        method: 'POST',
        data: { taskId: taskId },
        timeout: 20000
      })
      var taskStatus = data.status || 'pending'
      var elapsedSeconds = Math.max(0, Math.round((Date.now() - (this.analyzeStartedAt || Date.now())) / 1000))
      this.setData({
        analyzeTaskStatus: taskStatus,
        analyzeProgressPercent: buildAnalyzeProgressPercent(taskStatus, elapsedSeconds),
        analyzingText: buildAnalyzeStatusText(taskStatus, elapsedSeconds)
      })

      if (data.ready) {
        this.finishAnalyzeProgress('AI 报告已生成，正在打开。')
        wx.navigateTo({
          url: '/pages/report/report?title=' + encodeURIComponent(data.title || 'AI 志愿分析报告') + '&report=' + encodeURIComponent(data.report || '') + '&student=' + encodeURIComponent(JSON.stringify(data.student || this.data.student || {}))
        })
        return
      }

      if (data.failed) {
        this.finishAnalyzeProgress((data.errorMessage || 'AI 报告生成失败'))
        wx.showToast({ title: data.errorMessage || '生成失败', icon: 'none' })
      }
    } catch (err) {
      this.finishAnalyzeProgress('AI 报告轮询失败，请重新生成。')
      if (!err || !err.handledByModal) {
        wx.showToast({ title: (err && err.error) || '轮询失败', icon: 'none' })
      }
    } finally {
      this.analyzePollingRequest = false
    }
  },

  async onAnalyze() {
    if (this.data.loading) {
      return
    }

    this.startAnalyzeProgress()
    try {
      var payload = {
        student: this.data.student,
        recommend: this.data.result
      }
      var data = await request({
        url: '/api/analyze-task',
        method: 'POST',
        data: payload,
        timeout: 20000
      })
      this.startAnalyzeProgress(data.taskId || '', data.status || 'pending')
      this.startAnalyzePolling(data.taskId)
    } catch (err) {
      this.finishAnalyzeProgress('AI 报告生成未开始，请稍后重试。')
      if (!err || !err.handledByModal) {
        wx.showToast({ title: (err && err.error) || '生成失败', icon: 'none' })
      }
    }
  },

  onShareAppMessage() {
    var student = this.data.student || {}
    return {
      title: buildRecommendShareTitle(student),
      path: '/pages/index/index?' + buildRecommendShareQuery(student),
      imageUrl: ''
    }
  },

  onShareTimeline() {
    var student = this.data.student || {}
    return {
      title: buildRecommendShareTitle(student),
      query: buildRecommendShareQuery(student),
      imageUrl: ''
    }
  }
})

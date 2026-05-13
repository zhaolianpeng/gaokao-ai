const { request } = require('../../utils/request')
const { getVIPEntryVisibility } = require('../../utils/vip-entry')
const { shouldRequireShareGate, markShareGateUnlocked, createShareGateToken } = require('../../utils/share-gate')
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

const BUCKET_CONFIGS = [
  { key: 'chong', label: '冲刺', schoolTitle: '冲刺区', schoolSubtitle: '这部分是少量前置冲高位，主要承担向上试探更高层次学校的职责。', expanded: true },
  { key: 'jiaoChong', label: '较冲', schoolTitle: '较冲区', schoolSubtitle: '比冲刺更接近真实机会区，适合放在前段作为第二梯队。', expanded: true },
  { key: 'wen', label: '稳妥', schoolTitle: '稳妥区', schoolSubtitle: '这部分最适合承担主力志愿，兼顾录取把握和学校层次。', expanded: true },
  { key: 'jiaoBao', label: '较保', schoolTitle: '较保区', schoolSubtitle: '这部分录取把握更高，适合补强表格中后段的安全系数。', expanded: false },
  { key: 'bao', label: '保底', schoolTitle: '保底区', schoolSubtitle: '这部分主要负责兜住录取结果，避免整张表整体过冲。', expanded: false }
]

function getBucketKeys() {
  return ['chong', 'jiaoChong', 'wen', 'jiaoBao', 'bao']
}

function flattenBuckets(result, order) {
  var keys = Array.isArray(order) && order.length ? order : getBucketKeys()
  var merged = []
  for (var i = 0; i < keys.length; i += 1) {
    merged = merged.concat(result[keys[i]] || [])
  }
  return merged
}

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

function buildRecentRankText(item) {
  var parts = []
  if (item.rank_last_year) {
    parts.push(`去年 ${item.rank_last_year}`)
  }
  if (item.rank_two_years_ago) {
    parts.push(`前年 ${item.rank_two_years_ago}`)
  }
  if (item.rank_three_years_ago) {
    parts.push(`三年前 ${item.rank_three_years_ago}`)
  }
  return parts.join(' / ')
}

function cloneItem(item) {
  var probability = item.probability || 0
  var hasLineData = item.current_line_score && item.last_year_line_score
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
    score_last_year: item.score_last_year || 0,
    min_rank: item.min_rank || 0,
    avg_score: item.avg_score || 0,
    current_line_score: item.current_line_score || 0,
    last_year_line_score: item.last_year_line_score || 0,
    current_line_diff: item.current_line_diff || 0,
    last_year_line_diff: item.last_year_line_diff || 0,
    line_diff_gap: item.line_diff_gap || 0,
    rank_last_year: item.rank_last_year || 0,
    rank_two_years_ago: item.rank_two_years_ago || 0,
    rank_three_years_ago: item.rank_three_years_ago || 0,
    weighted_rank: item.weighted_rank || 0,
    probability: probability,
    probabilityLabel: item.probability_label || '',
    tag: item.tag || '',
    target_hit: item.target_hit || 0,
    itemKey: [item.college_id || 0, item.group_code || '', item.batch || '', item.subject_requirement || '', item.min_rank || 0].join('::'),
    probabilityText: Math.round(probability * 100) + '%',
    groupLabel: ((item.group_code || '') + ' ' + (item.group_name || '')).trim(),
    majorPreview: item.matched_major || item.major || '未提供组内专业',
    recentRankText: buildRecentRankText(item),
    weightedRankText: item.weighted_rank ? `加权参考位次 ${item.weighted_rank}` : '',
    lastYearScoreText: item.score_last_year ? `去年最低分 ${item.score_last_year}` : '',
    lineDiffText: hasLineData ? `线差 ${item.current_line_diff || 0} / 去年线差 ${item.last_year_line_diff || 0}` : '',
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
      jiaoChong: normalizeBucket(safePayload.result && safePayload.result.jiaoChong, 'jiaochong'),
      wen: normalizeBucket(safePayload.result && safePayload.result.wen, 'wen'),
      jiaoBao: normalizeBucket(safePayload.result && safePayload.result.jiaoBao, 'jiaobao'),
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
  var jiaoChong = result.jiaoChong || []
  var wen = result.wen || []
  var jiaoBao = result.jiaoBao || []
  var bao = result.bao || []
  var i

  if (strategyKey === 'balanced') {
    for (i = 0; i < 1 && i < chong.length; i += 1) {
      pushUniqueScenarioItem(items, chong[i], seen)
    }
    for (i = 0; i < 1 && i < jiaoChong.length; i += 1) {
      pushUniqueScenarioItem(items, jiaoChong[i], seen)
    }
    for (i = 0; i < 2 && i < wen.length; i += 1) {
      pushUniqueScenarioItem(items, wen[i], seen)
    }
    for (i = 0; i < 1 && i < jiaoBao.length; i += 1) {
      pushUniqueScenarioItem(items, jiaoBao[i], seen)
    }
    for (i = 0; i < 2 && i < bao.length; i += 1) {
      pushUniqueScenarioItem(items, bao[i], seen)
    }
  } else if (strategyKey === 'major') {
    var source = flattenBuckets(result, ['wen', 'jiaoChong', 'chong', 'jiaoBao', 'bao'])
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
    for (i = 0; i < 1 && i < jiaoBao.length; i += 1) {
      pushUniqueScenarioItem(items, jiaoBao[i], seen)
    }
    for (i = 0; i < 2 && i < bao.length; i += 1) {
      pushUniqueScenarioItem(items, bao[i], seen)
    }
  } else {
    for (i = 0; i < 2 && i < chong.length; i += 1) {
      pushUniqueScenarioItem(items, chong[i], seen)
    }
    for (i = 0; i < 2 && i < jiaoChong.length; i += 1) {
      pushUniqueScenarioItem(items, jiaoChong[i], seen)
    }
    for (i = 0; i < 1 && i < wen.length; i += 1) {
      pushUniqueScenarioItem(items, wen[i], seen)
    }
    for (i = 0; i < 1 && i < jiaoBao.length; i += 1) {
      pushUniqueScenarioItem(items, jiaoBao[i], seen)
    }
  }

  var fallback = flattenBuckets(result, ['wen', 'jiaoBao', 'jiaoChong', 'bao', 'chong'])
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

function buildTargetMajorKeywords(text) {
  var value = String(text || '').trim()
  if (!value) {
    return []
  }
  var normalized = value
    .replace(/[，、；;|/（）()]/g, ' ')
    .replace(/\s+/g, ' ')
    .trim()
  var parts = normalized ? normalized.split(' ') : []
  var seen = {}
  var result = []

  function pushKeyword(keyword) {
    var next = String(keyword || '').trim()
    if (!next || seen[next]) {
      return
    }
    seen[next] = true
    result.push(next)
  }

  for (var i = 0; i < parts.length; i += 1) {
    var part = parts[i]
    pushKeyword(part)
    var suffixes = ['专业', '类', '方向']
    for (var j = 0; j < suffixes.length; j += 1) {
      if (part.length > suffixes[j].length && part.indexOf(suffixes[j], part.length - suffixes[j].length) >= 0) {
        pushKeyword(part.slice(0, part.length - suffixes[j].length))
      }
    }
  }

  if (!result.length) {
    pushKeyword(value)
  }
  return result
}

function getMajorRelationScore(item, targetMajor) {
  if (!item) {
    return 0
  }
  var score = (item.matched_major ? 100 : 0) + (item.target_hit ? 40 : 0)
  var keywords = buildTargetMajorKeywords(targetMajor)
  if (!keywords.length) {
    return score
  }
  var text = `${item.majorPreview || ''}\n${item.major || ''}`
  for (var i = 0; i < keywords.length; i += 1) {
    if (text.indexOf(keywords[i]) >= 0) {
      score += Math.max(12, keywords[i].length * 4)
    }
  }
  return score
}

function getMajorCandidates(student, result) {
  var source = mergeUniqueItems(result)
  var targetMajor = student && student.targetMajor ? student.targetMajor : ''
  var candidates = []
  for (var i = 0; i < source.length; i += 1) {
    var item = source[i]
    var relationScore = getMajorRelationScore(item, targetMajor)
    if (relationScore > 0) {
      item.majorRelationScore = relationScore
      candidates.push(item)
    }
  }
  return candidates.sort(function(a, b) {
    if ((b.majorRelationScore || 0) !== (a.majorRelationScore || 0)) {
      return (b.majorRelationScore || 0) - (a.majorRelationScore || 0)
    }
    if ((b.probability || 0) !== (a.probability || 0)) {
      return (b.probability || 0) - (a.probability || 0)
    }
    return (a.min_rank || 0) - (b.min_rank || 0)
  })
}

function buildStrategyCards(student, result) {
  var chongTop = getTopItem(result.chong)
  var jiaoChongTop = getTopItem(result.jiaoChong)
  var wenTop = getTopItem(result.wen)
  var jiaoBaoTop = getTopItem(result.jiaoBao)
  var baoTop = getTopItem(result.bao)
  var majorMatches = []
  var source = flattenBuckets(result, ['chong', 'jiaoChong', 'wen', 'jiaoBao', 'bao'])
  for (var i = 0; i < source.length; i += 1) {
    if (source[i].matched_major || source[i].target_hit) {
      majorMatches.push(source[i])
    }
  }
  var majorCandidates = getMajorCandidates(student, result)
  var majorFocusItem = majorMatches[0] || majorCandidates[0] || wenTop || chongTop || baoTop || null
  var majorFocusText = '当前没有可优先排查的专业组，建议先补充意向专业。'
  var majorNoteText = `当前命中意向专业/相近方向 ${majorMatches.length} 组。`
  if (majorMatches.length) {
    majorFocusText = `${majorMatches[0].college_name} ${majorMatches[0].majorPreview}`
  } else if (majorCandidates.length) {
    majorFocusText = `先排查 ${majorCandidates[0].college_name} ${majorCandidates[0].majorPreview}`
    majorNoteText = `当前严格命中 0 组，按专业名称接近度补出 ${majorCandidates.length} 组。`
  } else if (majorFocusItem) {
    majorFocusText = `先核查 ${majorFocusItem.college_name} ${majorFocusItem.majorPreview}`
    majorNoteText = '当前没有直接命中专业，先从当前结果里逐组核查组内专业。'
  }
  return [
    {
      key: 'balanced',
      title: '稳妥优先',
      desc: '先把主体志愿放在稳妥和较保区，再留少量位置给冲刺与较冲。',
      focus: wenTop ? `${wenTop.college_name} ${wenTop.groupLabel}` : '先从稳妥组前 3 所里定主力志愿',
      note: jiaoBaoTop ? `适合先稳住录取结果，再用 ${jiaoBaoTop.college_name} 这一类学校补安全垫。` : '适合家长希望先稳住录取结果、再少量冲高的填法。'
    },
    {
      key: 'major',
      title: '专业优先',
      desc: '先确认学校组里有没有目标专业或相近方向，再决定是否接受调剂。',
      focus: majorFocusText,
      note: majorNoteText
    },
    {
      key: 'tier',
      title: '冲层次优先',
      desc: '把冲刺和较冲当成提升学校层次的机会，但不要让整张表都偏冒险。',
      focus: chongTop ? `${chongTop.college_name} ${chongTop.groupLabel}` : jiaoChongTop ? `${jiaoChongTop.college_name} ${jiaoChongTop.groupLabel}` : '当前前段冲高位较少，建议先把主力和保底补齐',
      note: baoTop ? `家长沟通时，建议至少保留 ${baoTop.college_name} 这一类能兜住结果的学校。` : '较保和保底仍需补足。'
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
  var buckets = ['wen', 'jiaoBao', 'jiaoChong', 'bao', 'chong']
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
  var sections = []
  for (var i = 0; i < BUCKET_CONFIGS.length; i += 1) {
    sections.push(createLensSection({
      key: 'school-' + BUCKET_CONFIGS[i].key,
      title: BUCKET_CONFIGS[i].schoolTitle,
      subtitle: BUCKET_CONFIGS[i].schoolSubtitle,
      expanded: BUCKET_CONFIGS[i].expanded
    }, result[BUCKET_CONFIGS[i].key] || [], favoriteMap))
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
    } else if (item.tag === 'jiaobao' || item.tag === 'bao') {
      backup.push(item)
    } else {
      related.push(item)
    }
  }
  return [
    createLensSection({ key: 'major-match', title: '优先保专业', subtitle: '优先保住意向专业或相近方向。', expanded: true }, sortByMajorPriority(matched), favoriteMap),
    createLensSection({ key: 'major-related', title: '相近专业备选', subtitle: '专业方向相近，但需要你进一步核查组内专业结构。', expanded: true }, sortByMajorPriority(related), favoriteMap),
    createLensSection({ key: 'major-backup', title: '保录取兜底', subtitle: '当专业命中不足时，用较保和保底区先兜住录取。', expanded: false }, sortByMajorPriority(backup), favoriteMap)
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
      probabilityText: item.probabilityLabel || `录取概率 ${item.probabilityText}`,
      rankText: item.weighted_rank ? `近3年加权位次 ${item.weighted_rank}` : item.min_rank ? `最低位次 ${item.min_rank}` : '最低位次待补充'
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
      { title: '最后补较保和保底', desc: '较保和保底主要负责兜录取，不要用它们来承担主要专业诉求。' },
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
    { title: '先定主力稳妥区', desc: sections.length > 2 && sections[2].itemCount ? `先从“${sections[2].title}”里挑 4-6 个主力学校，稳住录取结果。` : '先从主力稳妥区里定好核心学校。' },
    { title: '再看专业会不会跑偏', desc: `围绕 ${majorText} 逐一核查组内专业，避免学校合适但专业方向偏掉。` },
    { title: '最后留足较保和保底', desc: '较保和保底区至少保留 2-3 个家庭也能接受的学校，不要把全部希望都压在冲刺上。' },
    { title: '统一家庭排序', desc: '家长和考生先说清楚，到底是先保学校、先保专业，还是先保城市，再排正式志愿。' }
  ]
}

function buildLensFamilySummary(student, activeLens, sections) {
  var focusText = activeLens === 'major' ? '当前这版方案优先保专业。' : activeLens === 'city' ? '当前这版方案优先保城市。' : '当前这版方案优先稳住学校层次和录取结果。'
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
    lines.push(`当前优先讨论的专业组：${top.college_name}${top.city ? '（' + top.city + '）' : ''} ${top.groupLabel}，${top.probabilityLabel || ('录取概率约 ' + top.probabilityText)}。`)
    if (top.weightedRankText) {
      lines.push(`${top.weightedRankText}，近 3 年参考：${top.recentRankText || '暂无完整历史位次'}。`)
    }
  }
  return lines.join('\n')
}

Page({
  data: {
    loading: false,
    loadError: '',
    shareGateReady: false,
    shareUnlocked: false,
    shareUnlockPending: false,
    analyzingText: ANALYZE_IDLE_TEXT,
    analyzeButtonText: '生成黑龙江 AI 报考报告',
    student: {},
    result: { chong: [], jiaoChong: [], wen: [], jiaoBao: [], bao: [] },
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
    wx.setNavigationBarTitle({ title: '黑龙江专业组方案' })
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
        result: { chong: [], jiaoChong: [], wen: [], jiaoBao: [], bao: [] },
        summary: [],
        sections: [],
        topSummary: []
      })
    }
    shouldRequireShareGate('recommendResult', true, safeQuery.shareToken || '').then((required) => {
      if (required) {
        this.setData({ shareGateReady: true })
        wx.setNavigationBarTitle({ title: '分享后查看智能推荐结果' })
        return
      }
      this.unlockRecommendResult()
    })
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

  requestShareUnlock() {
    this.setData({ shareUnlockPending: true })
  },

  unlockRecommendResult() {
    if (this.data.shareUnlocked) {
      this.setData({ shareGateReady: true })
      return
    }
    this.setData({ shareUnlocked: true, shareUnlockPending: false, shareGateReady: true })
    wx.setNavigationBarTitle({ title: '黑龙江专业组方案' })
  },

  syncVIPEntryVisibility(forceRefresh) {
  return getVIPEntryVisibility(forceRefresh).then((showVipEntry) => {
		if (this.data.showVipEntry !== showVipEntry) {
			this.setData({ showVipEntry })
		}
	})
  },

  buildSummary(result) {
    var summary = []
    for (var i = 0; i < BUCKET_CONFIGS.length; i += 1) {
      summary.push({ label: BUCKET_CONFIGS[i].label, value: (result[BUCKET_CONFIGS[i].key] || []).length, type: BUCKET_CONFIGS[i].key })
    }
    return summary
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

    var summary = []
    for (var i = 0; i < BUCKET_CONFIGS.length; i += 1) {
      summary.push({ label: BUCKET_CONFIGS[i].label, text: makeText(result[BUCKET_CONFIGS[i].key]) })
    }
    return summary
  },

  applyViewModel() {
    var result = this.data.result || { chong: [], jiaoChong: [], wen: [], jiaoBao: [], bao: [] }
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
    var buckets = getBucketKeys()
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
    var self = this
    var shareToken = createShareGateToken('recommendResult')
    return {
      title: buildRecommendShareTitle(student),
      path: '/pages/recommend/recommend?shareToken=' + encodeURIComponent(shareToken),
      imageUrl: '',
      success() {
        if (self.data.shareUnlockPending && !self.data.shareUnlocked) {
          markShareGateUnlocked('recommendResult')
          self.unlockRecommendResult()
        }
      },
      fail() {
        if (self.data.shareUnlockPending && !self.data.shareUnlocked) {
          self.setData({ shareUnlockPending: false })
        }
      }
    }
  },

  onShareTimeline() {
    var shareToken = createShareGateToken('recommendResult')
    return {
      title: '黑龙江高报助手：查看智能推荐结果',
      query: 'shareToken=' + encodeURIComponent(shareToken),
      imageUrl: ''
    }
  }
})

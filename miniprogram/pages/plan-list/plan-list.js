const {
  getFavoriteProgramGroups,
  getApplicationPlan,
  replaceApplicationPlan,
  getPlanScenarios,
  buildApplicationPlanFromFavorites,
  applyPlanScenario,
  removePlanScenario,
  removeApplicationPlanItem,
  clearApplicationPlan
} = require('../../utils/storage')
const { getVIPEntryVisibility } = require('../../utils/vip-entry')
const { shouldRequireShareGate, markShareGateUnlocked, createShareGateToken } = require('../../utils/share-gate')

const TAG_ORDER = {
  chong: 1,
  jiaochong: 2,
  wen: 3,
  jiaobao: 4,
  bao: 5,
  other: 6
}

const DRAG_ROW_RPX = 160

function enableShareMenus() {
  if (wx.showShareMenu) {
    wx.showShareMenu({ menus: ['shareAppMessage', 'shareTimeline'] })
  }
}

function formatTime(timestamp) {
  const date = new Date(timestamp)
  const month = `${date.getMonth() + 1}`.padStart(2, '0')
  const day = `${date.getDate()}`.padStart(2, '0')
  const hour = `${date.getHours()}`.padStart(2, '0')
  const minute = `${date.getMinutes()}`.padStart(2, '0')
  return `${month}-${day} ${hour}:${minute}`
}

function comparePlanItems(sortMode) {
  switch (sortMode) {
    case 'manual':
      return () => 0
    case 'rankAsc':
      return (left, right) => (left.item.min_rank || Number.MAX_SAFE_INTEGER) - (right.item.min_rank || Number.MAX_SAFE_INTEGER)
    case 'scoreDesc':
      return (left, right) => (right.item.min_score || 0) - (left.item.min_score || 0)
    case 'collegeAsc':
      return (left, right) => (left.item.college_name || '').localeCompare(right.item.college_name || '', 'zh-Hans-CN')
    default:
      return (left, right) => (right.createdAt || 0) - (left.createdAt || 0)
  }
}

function buildOrderedPlanList(list, sortMode) {
  const source = Array.isArray(list) ? list.slice() : []
  if (sortMode === 'manual') {
    return source
  }
  return source.sort(comparePlanItems(sortMode))
}

function buildGroupedPlanList(list, sortMode) {
  const orderedList = buildOrderedPlanList(list, sortMode)
  const groups = {
    chong: { key: 'chong', title: '冲刺组', items: [] },
    jiaochong: { key: 'jiaochong', title: '较冲组', items: [] },
    wen: { key: 'wen', title: '稳妥组', items: [] },
    jiaobao: { key: 'jiaobao', title: '较保组', items: [] },
    bao: { key: 'bao', title: '保底组', items: [] },
    other: { key: 'other', title: '待定组', items: [] }
  }
  orderedList.forEach((entry) => {
    const key = entry.item.tag || 'other'
    if (!groups[key]) {
      groups.other.items.push(entry)
      return
    }
    groups[key].items.push(entry)
  })
  return ['chong', 'jiaochong', 'wen', 'jiaobao', 'bao', 'other']
    .map((key) => ({
      ...groups[key],
      items: groups[key].items.sort(comparePlanItems(sortMode))
    }))
    .filter((group) => group.items.length)
}

function buildArchiveText(student) {
  const safeStudent = student || {}
  const parts = [safeStudent.schoolName, safeStudent.schoolYear, safeStudent.className].filter(Boolean)
  return parts.length ? parts.join(' / ') : ''
}

function buildStudentSummary(student) {
  const safeStudent = student || {}
  const subjectText = safeStudent.subject || '未填写科类'
  const scoreText = safeStudent.score ? `${safeStudent.score}分` : '未填分数'
  const rankText = safeStudent.rank ? `${safeStudent.rank}名` : '未填位次'
  const archiveText = buildArchiveText(safeStudent)
  const sourceText = safeStudent.fromRecommend ? '推荐来源' : ''
  return {
    baseText: `黑龙江 ${subjectText} · ${scoreText} / ${rankText}`,
    archiveText,
    sourceText
  }
}

function buildExportText(list, summary, sequenceRisk) {
  const lines = ['黑龙江正式志愿表']
  if (summary && summary.baseText) {
    lines.push(summary.baseText)
  }
  if (summary && summary.archiveText) {
    lines.push(`档案上下文：${summary.archiveText}`)
  }
  if (summary && summary.sourceText) {
    lines.push(summary.sourceText)
  }
  if (sequenceRisk && sequenceRisk.summaryText) {
    lines.push(`顺序评估：${sequenceRisk.summaryText}`)
  }
  ;(Array.isArray(list) ? list : []).forEach((entry, index) => {
    const item = entry.item || {}
    const tagText = item.tag === 'chong' ? '冲刺' : item.tag === 'jiaochong' ? '较冲' : item.tag === 'wen' ? '稳妥' : item.tag === 'jiaobao' ? '较保' : item.tag === 'bao' ? '保底' : '待定'
    lines.push(
      `${index + 1}. [${tagText}] ${item.college_name || ''} ${item.group_code || ''} ${item.group_name || ''} | ${item.batch || ''} | 选科 ${item.subject_requirement || '不限'} | 最低位次 ${item.min_rank || '无'} | 最低分 ${item.min_score || '无'} | 专业 ${item.majorPreview || item.matched_major || item.major || '未提供'}`
    )
  })
  return lines.join('\n')
}

function getTagOrderValue(tag) {
  return TAG_ORDER[tag] || TAG_ORDER.other
}

function buildSequenceRisk(list) {
  const ordered = Array.isArray(list) ? list : []
  if (!ordered.length) {
    return {
      score: 0,
      level: '未开始',
      levelClass: 'pending',
      summaryText: '先加入志愿后再看顺序风险。',
      warnings: ['当前志愿表还没有内容。'],
      suggestions: ['先从推荐结果页加入学校，再开始拖动微调顺序。'],
      conflictDetails: []
    }
  }

  let reverseTagCount = 0
  let reverseRankCount = 0
  let denseCount = 0
  const warnings = []
  const conflictDetails = []
  for (let i = 1; i < ordered.length; i += 1) {
    const prev = ordered[i - 1].item || {}
    const next = ordered[i].item || {}
    if (getTagOrderValue(next.tag) < getTagOrderValue(prev.tag)) {
      reverseTagCount += 1
      conflictDetails.push({
        key: `tag-${i}`,
        type: '层级回跳',
        firstId: ordered[i - 1].id,
        secondId: ordered[i].id,
        firstIndex: i,
        secondIndex: i + 1,
        text: `第 ${i} 志愿 ${prev.college_name || '未命名院校'} 之后，紧跟的第 ${i + 1} 志愿 ${next.college_name || '未命名院校'} 更冲。`
      })
    }
    const prevRank = Number(prev.min_rank || 0)
    const nextRank = Number(next.min_rank || 0)
    if (prevRank && nextRank) {
      if (nextRank < prevRank) {
        reverseRankCount += 1
        conflictDetails.push({
          key: `rank-${i}`,
          type: '位次倒挂',
          firstId: ordered[i - 1].id,
          secondId: ordered[i].id,
          firstIndex: i,
          secondIndex: i + 1,
          text: `第 ${i} 志愿 ${prev.college_name || '未命名院校'} 最低位次 ${prevRank}，但第 ${i + 1} 志愿 ${next.college_name || '未命名院校'} 更高到 ${nextRank}。`
        })
      }
      const threshold = Math.max(120, Math.round(prevRank * 0.03))
      if (Math.abs(nextRank - prevRank) < threshold) {
        denseCount += 1
        conflictDetails.push({
          key: `dense-${i}`,
          type: '梯度过密',
          firstId: ordered[i - 1].id,
          secondId: ordered[i].id,
          firstIndex: i,
          secondIndex: i + 1,
          text: `第 ${i} 志愿 ${prev.college_name || '未命名院校'} 和第 ${i + 1} 志愿 ${next.college_name || '未命名院校'} 位次差仅 ${Math.abs(nextRank - prevRank)}。`
        })
      }
    }
  }

  let deduction = 0
  if (reverseTagCount) {
    warnings.push('前后志愿的冲稳保层级有回跳，后面的学校比前面更冲。')
    deduction += Math.min(20, reverseTagCount * 6)
  }
  if (reverseRankCount) {
    warnings.push('部分后置志愿的最低位次比前项更高，顺序存在倒挂。')
    deduction += Math.min(18, reverseRankCount * 5)
  }
  if (denseCount >= 2) {
    warnings.push('相邻志愿位次过密，容易出现整段扎堆。')
    deduction += Math.min(16, denseCount * 4)
  }

  const score = Math.max(0, 100 - deduction)
  let level = '顺序合理'
  let levelClass = 'excellent'
  if (score < 50) {
    level = '顺序风险高'
    levelClass = 'danger'
  } else if (score < 75) {
    level = '顺序需调整'
    levelClass = 'warning'
  } else if (score < 90) {
    level = '基本合理'
    levelClass = 'good'
  }

  const suggestions = []
  if (reverseTagCount || reverseRankCount) {
    suggestions.push('把更冲的学校往前拖，把更稳和保底的学校放到后段。')
  }
  if (denseCount >= 2) {
    suggestions.push('拉开相邻学校最低位次差，避免志愿密集在同一区间。')
  }
  if (!suggestions.length) {
    suggestions.push('当前顺序已经比较顺，可以继续细看专业和城市偏好。')
  }

  return {
    score,
    level,
    levelClass,
    summaryText: `倒挂 ${reverseRankCount} 处，层级回跳 ${reverseTagCount} 处，密集 ${denseCount} 处。`,
    warnings: warnings.length ? warnings : ['当前顺序没有明显倒挂，整体梯度比较顺。'],
    suggestions,
    conflictDetails: conflictDetails.slice(0, 6)
  }
}

function buildDragPlanList(list, rowHeightPx, draggingId, dragYMap, fromIndex, targetIndex) {
  const ordered = Array.isArray(list) ? list : []
  return ordered.map((entry, index) => ({
    ...entry,
    orderIndex: index + 1,
    dragY: draggingId === entry.id && dragYMap && typeof dragYMap[entry.id] === 'number'
      ? dragYMap[entry.id]
      : index * rowHeightPx + (draggingId && typeof fromIndex === 'number' && typeof targetIndex === 'number'
        ? (fromIndex < targetIndex && index > fromIndex && index <= targetIndex
          ? -rowHeightPx
          : (fromIndex > targetIndex && index >= targetIndex && index < fromIndex ? rowHeightPx : 0))
        : 0)
  }))
}

function buildScenarioCompareBoard(scenarios) {
  const selected = (Array.isArray(scenarios) ? scenarios : []).filter((item) => item.selected).slice(0, 3)
  if (selected.length < 2) {
    return null
  }

  function getAggressiveCount(item) {
    const metrics = (item && item.metrics) || {}
    return (Number(metrics.chong) || 0) + (Number(metrics.jiaoChong) || 0)
  }

  function getWinnerIds(metricKey) {
    const values = selected.map((item) => Number(item.metrics && item.metrics[metricKey]) || 0)
    const maxValue = Math.max.apply(null, values)
    if (maxValue <= 0) {
      return []
    }
    return selected.filter((item) => (Number(item.metrics && item.metrics[metricKey]) || 0) === maxValue).map((item) => item.id)
  }

  const stableWinnerIds = getWinnerIds('wen')
  const aggressiveValues = selected.map((item) => getAggressiveCount(item))
  const aggressiveMax = Math.max.apply(null, aggressiveValues)
  const aggressiveWinnerIds = aggressiveMax > 0 ? selected.filter((item) => getAggressiveCount(item) === aggressiveMax).map((item) => item.id) : []
  const targetWinnerIds = getWinnerIds('targetHits')

  function getTitles(ids) {
    return selected.filter((item) => ids.indexOf(item.id) >= 0).map((item) => item.title)
  }

  function buildRecommendationLine(prefix, ids, fallback) {
    const titles = getTitles(ids)
    if (!titles.length) {
      return fallback
    }
    if (titles.length === 1) {
      return `${prefix}，优先看${titles[0]}。`
    }
    return `${prefix}，优先在${titles.join('、')}之间再结合城市和院校偏好细比。`
  }

  function buildCells(label, getText, highlightIds) {
    return selected.map((item) => ({
      id: `${label}-${item.id}`,
      text: getText(item),
      highlighted: highlightIds.indexOf(item.id) >= 0
    }))
  }

  return {
    columns: selected.map((item) => ({
      id: item.id,
      title: item.title,
      total: (item.metrics && item.metrics.total) || 0,
      badges: [
        stableWinnerIds.indexOf(item.id) >= 0 ? '更稳' : '',
        aggressiveWinnerIds.indexOf(item.id) >= 0 ? '更冲' : '',
        targetWinnerIds.indexOf(item.id) >= 0 ? '更贴近目标专业' : ''
      ].filter(Boolean)
    })),
    rows: [
      {
        label: '方案定位',
        cells: buildCells('position', (item) => item.desc || item.note || '暂无说明', [])
      },
      {
        label: '更稳',
        cells: buildCells('stable', (item) => `稳妥组 ${(item.metrics && item.metrics.wen) || 0} 个`, stableWinnerIds)
      },
      {
        label: '更冲',
        cells: buildCells('aggressive', (item) => `冲刺+较冲 ${getAggressiveCount(item)} 个`, aggressiveWinnerIds)
      },
      {
        label: '更贴近目标专业',
        cells: buildCells('target', (item) => `命中目标方向 ${(item.metrics && item.metrics.targetHits) || 0} 组`, targetWinnerIds)
      },
      {
        label: '五层结构',
        cells: buildCells('mix', (item) => `冲 ${(item.metrics && item.metrics.chong) || 0} / 较冲 ${(item.metrics && item.metrics.jiaoChong) || 0} / 稳 ${(item.metrics && item.metrics.wen) || 0} / 较保 ${(item.metrics && item.metrics.jiaoBao) || 0} / 保 ${(item.metrics && item.metrics.bao) || 0}`, [])
      },
      {
        label: '代表院校',
        cells: buildCells('college', (item) => item.collegeText || '暂无代表院校', [])
      },
      {
        label: '适合谁',
        cells: buildCells('fit', (item) => item.focus || item.note || '建议结合家长偏好再判断', [])
      }
    ],
    summaryChips: [
      stableWinnerIds.length ? `更稳：${selected.filter((item) => stableWinnerIds.indexOf(item.id) >= 0).map((item) => item.title).join('、')}` : '',
      aggressiveWinnerIds.length ? `更冲：${selected.filter((item) => aggressiveWinnerIds.indexOf(item.id) >= 0).map((item) => item.title).join('、')}` : '',
      targetWinnerIds.length ? `更贴近目标专业：${selected.filter((item) => targetWinnerIds.indexOf(item.id) >= 0).map((item) => item.title).join('、')}` : ''
    ].filter(Boolean),
    conclusions: [
      buildRecommendationLine('如果家长更看重录取把握', stableWinnerIds, '如果家长更看重录取把握，建议先看稳妥和较保更多的方案。'),
      buildRecommendationLine('如果家长更看重学校层次提升', aggressiveWinnerIds, '如果家长更看重学校层次提升，建议先看冲刺和较冲更多的方案。'),
      buildRecommendationLine('如果家长更看重专业贴合', targetWinnerIds, '如果家长更看重专业贴合，建议优先看命中目标方向更多的方案。')
    ]
  }
}

function buildFamilySharePayload(compareBoard, scenarios) {
  const selected = (Array.isArray(scenarios) ? scenarios : []).filter((item) => item.selected).slice(0, 3)
  if (!compareBoard || selected.length < 2) {
    return null
  }
  const student = selected[0] && selected[0].student ? selected[0].student : {}
  return {
    mode: 'compare',
    title: '给家长看的方案对比卡',
    student,
    summary: compareBoard.summaryChips || [],
    conclusions: compareBoard.conclusions || [],
    columns: compareBoard.columns.map((item) => ({ title: item.title, badges: item.badges || [] })),
    rows: compareBoard.rows.slice(0, 5).map((row) => ({
      label: row.label,
      cells: (row.cells || []).map((cell) => ({ text: cell.text, highlighted: !!cell.highlighted }))
    })),
    footer: '先统一是保结果、保专业还是冲层次，再决定正式志愿表。'
  }
}

Page({
  data: {
    favorites: [],
    applicationList: [],
    groupedApplicationList: [],
    studentSummary: null,
    scenarios: [],
    compareBoard: null,
    orderedApplicationList: [],
    dragPlanList: [],
    dragRowHeightPx: 0,
    dragViewportHeightPx: 0,
    dragAreaHeightPx: 0,
    draggingPlanId: '',
    dragReadyPlanId: '',
    draggingFromIndex: -1,
    currentDropIndex: -1,
    dragYMap: {},
    dragScrollTop: 0,
    highlightedPlanIds: [],
    activeConflictKey: '',
    sequenceRisk: buildSequenceRisk([]),
    sortMode: 'manual',
    sortOptions: [
      { value: 'manual', label: '自定义顺序' },
      { value: 'createdDesc', label: '按加入时间' },
      { value: 'rankAsc', label: '按最低位次升序' },
      { value: 'scoreDesc', label: '按最低分降序' },
      { value: 'collegeAsc', label: '按院校名称' }
    ],
    currentSortLabel: '按加入时间',
    showVipEntry: false,
    shareGateReady: false,
    shareUnlocked: false,
    shareUnlockPending: false
  },

  onLoad(query) {
    this.shareToken = (query && query.shareToken) || ''
    const info = wx.getSystemInfoSync ? wx.getSystemInfoSync() : { windowWidth: 375, windowHeight: 812 }
    this.dragRowHeightPx = Math.round((info.windowWidth || 375) * DRAG_ROW_RPX / 750)
    this.dragViewportHeightPx = Math.min(Math.round((info.windowHeight || 812) * 0.52), this.dragRowHeightPx * 6)
  },

  onShow() {
    enableShareMenus()
    this.syncVIPEntryVisibility(true)
    this.refreshData()
    if (this.data.shareUnlocked) {
      this.setData({ shareGateReady: true })
      wx.setNavigationBarTitle({ title: '正式志愿表与方案对比' })
      return
    }
    shouldRequireShareGate('planCompare', true, this.shareToken || '').then((required) => {
      if (required) {
        this.setData({ shareGateReady: true })
        wx.setNavigationBarTitle({ title: '分享后查看志愿表与方案对比' })
        return
      }
      this.unlockPlanCompare()
    })
  },

  syncVIPEntryVisibility(forceRefresh) {
	return getVIPEntryVisibility(forceRefresh).then((showVipEntry) => {
		if (this.data.showVipEntry !== showVipEntry) {
			this.setData({ showVipEntry })
		}
	})
  },

  refreshData() {
    const favorites = getFavoriteProgramGroups().map((entry) => ({
      ...entry,
      timeText: formatTime(entry.createdAt)
    }))
    const previousSelectedMap = {}
    ;(this.data.scenarios || []).forEach((item) => {
      if (item && item.selected) {
        previousSelectedMap[item.id] = true
      }
    })
    const rawScenarios = getPlanScenarios().map((entry) => ({
      ...entry,
      timeText: formatTime(entry.updatedAt || entry.createdAt),
      collegeText: ((entry.metrics && entry.metrics.topColleges) || []).join('、') || '暂无代表院校'
    }))
    const scenarios = rawScenarios.map((entry, index) => ({
      ...entry,
      selected: previousSelectedMap[entry.id] || (!Object.keys(previousSelectedMap).length && index < 2)
    }))
    const applicationList = getApplicationPlan().map((entry) => ({
      ...entry,
      timeText: formatTime(entry.createdAt)
    }))
    const orderedApplicationList = buildOrderedPlanList(applicationList, this.data.sortMode)
    const primaryStudent = (applicationList[0] && applicationList[0].student) || (rawScenarios[0] && rawScenarios[0].student) || null
    this.setData({
      favorites,
      scenarios,
      compareBoard: buildScenarioCompareBoard(scenarios),
      applicationList,
      orderedApplicationList,
      dragPlanList: buildDragPlanList(orderedApplicationList, this.dragRowHeightPx, this.data.draggingPlanId, this.data.dragYMap, this.data.draggingFromIndex, this.data.currentDropIndex),
      dragAreaHeightPx: orderedApplicationList.length * this.dragRowHeightPx,
      sequenceRisk: buildSequenceRisk(orderedApplicationList),
      groupedApplicationList: buildGroupedPlanList(applicationList, this.data.sortMode),
      studentSummary: buildStudentSummary(primaryStudent),
      currentSortLabel: (this.data.sortOptions.find((option) => option.value === this.data.sortMode) || this.data.sortOptions[0]).label
    })
  },

  requestShareUnlock() {
    this.setData({ shareUnlockPending: true })
  },

  unlockPlanCompare() {
    this.setData({ shareUnlocked: true, shareUnlockPending: false, shareGateReady: true })
    wx.setNavigationBarTitle({ title: '正式志愿表与方案对比' })
    this.refreshData()
  },

  onSortChange(e) {
    const sortMode = this.data.sortOptions[e.detail.value].value
    const orderedApplicationList = buildOrderedPlanList(this.data.applicationList, sortMode)
    this.setData({
      sortMode,
      orderedApplicationList,
      currentSortLabel: this.data.sortOptions[e.detail.value].label,
      groupedApplicationList: buildGroupedPlanList(this.data.applicationList, sortMode),
      dragPlanList: buildDragPlanList(orderedApplicationList, this.dragRowHeightPx, '', {}, -1, -1),
      dragAreaHeightPx: orderedApplicationList.length * this.dragRowHeightPx,
      draggingPlanId: '',
      dragReadyPlanId: '',
      draggingFromIndex: -1,
      currentDropIndex: -1,
      dragYMap: {},
      dragScrollTop: 0,
      highlightedPlanIds: [],
      activeConflictKey: '',
      sequenceRisk: buildSequenceRisk(orderedApplicationList)
    })
  },

  onExportPlan() {
    const text = buildExportText(this.data.orderedApplicationList, this.data.studentSummary, this.data.sequenceRisk)
    wx.setClipboardData({
      data: text,
      success: () => wx.showToast({ title: '已复制填报清单', icon: 'none' })
    })
  },

  onBuildFromFavorites() {
    const count = buildApplicationPlanFromFavorites()
    if (!count) {
      wx.showToast({ title: '暂无收藏专业组', icon: 'none' })
      return
    }
    this.refreshData()
    wx.showToast({ title: `已导入 ${count} 个专业组`, icon: 'none' })
  },

  onApplyScenario(e) {
    const id = e.currentTarget.dataset.id
    if (!id) {
      return
    }
    const count = applyPlanScenario(id)
    if (!count) {
      wx.showToast({ title: '方案内容为空', icon: 'none' })
      return
    }
    this.refreshData()
    wx.showToast({ title: `已应用 ${count} 个专业组`, icon: 'none' })
  },

  onToggleScenarioCompare(e) {
    const id = e.currentTarget.dataset.id
    if (!id) {
      return
    }
    const current = this.data.scenarios || []
    const selectedCount = current.filter((item) => item.selected).length
    const scenarios = current.map((item) => {
      if (item.id !== id) {
        return item
      }
      if (!item.selected && selectedCount >= 3) {
        wx.showToast({ title: '最多同时比较 3 套', icon: 'none' })
        return item
      }
      return {
        ...item,
        selected: !item.selected
      }
    })
    this.setData({
      scenarios,
      compareBoard: buildScenarioCompareBoard(scenarios)
    })
  },

  onRemoveScenario(e) {
    const id = e.currentTarget.dataset.id
    if (!id) {
      return
    }
    removePlanScenario(id)
    this.refreshData()
  },

  onRemovePlanItem(e) {
    const id = e.currentTarget.dataset.id
    if (!id) {
      return
    }
    removeApplicationPlanItem(id)
    this.refreshData()
  },

  onConflictDetailTap(e) {
    const detailKey = e.currentTarget.dataset.key
    const firstId = e.currentTarget.dataset.firstId
    const secondId = e.currentTarget.dataset.secondId
    const firstIndex = Number(e.currentTarget.dataset.firstIndex || 1)
    const secondIndex = Number(e.currentTarget.dataset.secondIndex || firstIndex + 1)
    if (!detailKey || !firstId || !secondId) {
      return
    }
    if (this.data.activeConflictKey === detailKey) {
      this.setData({ activeConflictKey: '', highlightedPlanIds: [] })
      return
    }
    const maxScrollTop = Math.max(0, this.data.dragAreaHeightPx - this.dragViewportHeightPx)
    const centerRow = ((firstIndex + secondIndex) / 2) - 0.5
    const nextScrollTop = this.data.sortMode === 'manual'
      ? Math.max(0, Math.min(maxScrollTop, Math.round(centerRow * this.dragRowHeightPx - this.dragViewportHeightPx / 2)))
      : this.data.dragScrollTop
    this.setData({
      activeConflictKey: detailKey,
      highlightedPlanIds: [firstId, secondId],
      dragScrollTop: nextScrollTop
    })
  },

  onEnablePlanDrag(e) {
    const id = e.currentTarget.dataset.id
    if (!id || this.data.sortMode !== 'manual') {
      return
    }
    if (this.dragReadyTimer) {
      clearTimeout(this.dragReadyTimer)
      this.dragReadyTimer = null
    }
    this.setData({ dragReadyPlanId: id })
    wx.showToast({ title: '已激活拖动，请拖动当前志愿', icon: 'none' })
    this.dragReadyTimer = setTimeout(() => {
      if (this.data.dragReadyPlanId === id && !this.data.draggingPlanId) {
        this.setData({ dragReadyPlanId: '' })
        wx.showToast({ title: '拖动激活已失效，请重新长按', icon: 'none' })
      }
      this.dragReadyTimer = null
    }, 5000)
  },

  onPlanDragStart(e) {
    const id = e.currentTarget.dataset.id
    if (!id || this.data.sortMode !== 'manual' || this.data.dragReadyPlanId !== id) {
      return
    }
    if (this.dragReadyTimer) {
      clearTimeout(this.dragReadyTimer)
      this.dragReadyTimer = null
    }
    const fromIndex = (this.data.orderedApplicationList || []).findIndex((entry) => entry.id === id)
    this.setData({ draggingPlanId: id, draggingFromIndex: fromIndex, currentDropIndex: fromIndex })
  },

  onPlanDragChange(e) {
    const id = e.currentTarget.dataset.id
    if (!id || this.data.sortMode !== 'manual') {
      return
    }
    const ordered = this.data.orderedApplicationList || []
    const fromIndex = this.data.draggingFromIndex >= 0 ? this.data.draggingFromIndex : ordered.findIndex((entry) => entry.id === id)
    const y = e.detail.y || 0
    const maxIndex = Math.max(0, ordered.length - 1)
    const nextDropIndex = Math.max(0, Math.min(Math.round(y / Math.max(1, this.dragRowHeightPx)), maxIndex))
    const maxScrollTop = Math.max(0, this.data.dragAreaHeightPx - this.dragViewportHeightPx)
    const threshold = Math.round(this.dragRowHeightPx * 1.2)
    let dragScrollTop = this.data.dragScrollTop || 0
    const visibleTop = y - dragScrollTop
    const visibleBottom = dragScrollTop + this.dragViewportHeightPx - y
    if (visibleTop < threshold) {
      dragScrollTop = Math.max(0, dragScrollTop - Math.round(this.dragRowHeightPx * 0.7))
    } else if (visibleBottom < threshold) {
      dragScrollTop = Math.min(maxScrollTop, dragScrollTop + Math.round(this.dragRowHeightPx * 0.7))
    }
    const dragYMap = {
      ...this.data.dragYMap,
      [id]: y
    }
    this.setData({
      dragYMap,
      dragScrollTop,
      currentDropIndex: nextDropIndex,
      dragPlanList: buildDragPlanList(ordered, this.dragRowHeightPx, id, dragYMap, fromIndex, nextDropIndex)
    })
  },

  onDragListScroll(e) {
    if (!e || !e.detail) {
      return
    }
    this.setData({ dragScrollTop: e.detail.scrollTop || 0 })
  },

  onPlanDragEnd(e) {
    const id = e.currentTarget.dataset.id
    if (!id || this.data.sortMode !== 'manual') {
      return
    }
    const ordered = (this.data.orderedApplicationList || []).slice()
    const fromIndex = ordered.findIndex((entry) => entry.id === id)
    if (fromIndex < 0) {
      this.setData({ draggingPlanId: '', draggingFromIndex: -1, currentDropIndex: -1, dragYMap: {} })
      return
    }
    const targetIndex = this.data.currentDropIndex >= 0
      ? this.data.currentDropIndex
      : Math.max(0, Math.min(fromIndex, ordered.length - 1))
    if (targetIndex !== fromIndex) {
      const moved = ordered.splice(fromIndex, 1)[0]
      ordered.splice(targetIndex, 0, moved)
      replaceApplicationPlan(ordered)
      wx.showToast({ title: '已更新志愿顺序', icon: 'none' })
    }
    if (this.dragReadyTimer) {
      clearTimeout(this.dragReadyTimer)
      this.dragReadyTimer = null
    }
    this.setData({ draggingPlanId: '', dragReadyPlanId: '', draggingFromIndex: -1, currentDropIndex: -1, dragYMap: {} })
    this.refreshData()
  },

  onClearPlan() {
    clearApplicationPlan()
    this.refreshData()
  },

  openVip() {
    wx.navigateTo({ url: '/pages/vip/vip' })
  },

  openFamilyShareCard() {
    const payload = buildFamilySharePayload(this.data.compareBoard, this.data.scenarios)
    if (!payload) {
      wx.showToast({ title: '请先选中 2 套以上方案', icon: 'none' })
      return
    }
    wx.navigateTo({
      url: '/pages/family-share/family-share?payload=' + encodeURIComponent(JSON.stringify(payload))
    })
  },

  onShareAppMessage() {
    const self = this
    const shareToken = createShareGateToken('planCompare')
    return {
      title: '黑龙江高报助手：查看正式志愿表与方案对比',
      path: `/pages/plan-list/plan-list?shareToken=${encodeURIComponent(shareToken)}`,
      success() {
        if (self.data.shareUnlockPending && !self.data.shareUnlocked) {
          markShareGateUnlocked('planCompare')
          self.unlockPlanCompare()
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
    const shareToken = createShareGateToken('planCompare')
    return {
      title: '黑龙江高报助手：查看正式志愿表与方案对比',
      query: `shareToken=${encodeURIComponent(shareToken)}`,
      imageUrl: ''
    }
  }
})

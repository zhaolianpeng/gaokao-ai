const { saveReportHistory, savePendingExploreFilters, savePendingReportPayload, getPendingReportPayload } = require('../../utils/storage')
const { getVIPEntryVisibility } = require('../../utils/vip-entry')
const { shouldRequireShareGate, markShareGateUnlocked, createShareGateToken } = require('../../utils/share-gate')

function enableShareMenus() {
  if (wx.showShareMenu) {
    wx.showShareMenu({ menus: ['shareAppMessage', 'shareTimeline'] })
  }
}

function buildReportBlocks(report) {
  return (report || '')
    .split(/\n\s*\n/)
    .map((block) => block.trim())
    .filter(Boolean)
    .map((block, index) => {
      const heading = block.match(/^#{1,3}\s*(.+)$/)
      if (heading) {
        return {
          id: `block-${index}`,
          type: 'heading',
          text: heading[1].trim()
        }
      }

      if (/^[-*]\s+/.test(block) || /^\d+\.\s+/.test(block)) {
        return {
          id: `block-${index}`,
          type: 'bullet',
          text: block.replace(/^[-*]\s+/, '').trim()
        }
      }

      return {
        id: `block-${index}`,
        type: 'paragraph',
        text: block
      }
    })
}

function buildOutline(blocks) {
  return blocks.filter((block) => block.type === 'heading').slice(0, 4)
}

function buildChecklist(student, suggestions) {
  const targetMajor = student && student.targetMajor ? student.targetMajor : '目标专业'
  const checklist = [
    `先确认 ${targetMajor} 是否真的出现在目标专业组里，而不是只看学校名称。`,
    '把冲稳保每组至少各保留 2-3 个能接受的选择。',
    '和家长先统一优先级：保学校、保专业、保城市到底谁排第一。',
    '正式提交前再核查招生计划、选科要求、学费和是否接受调剂。'
  ]
  if (Array.isArray(suggestions) && suggestions.length) {
    checklist.push(`下一步优先去院校库搜：${suggestions.map((item) => item.keyword).filter(Boolean).slice(0, 3).join('、')}`)
  }
  return checklist
}

function buildFamilyBrief(student, suggestions) {
  const archiveText = buildArchiveText(student)
  const lines = [
    `这是一份黑龙江${student && student.subject ? student.subject : ''}考生的 AI 报告。`,
    `当前分数/位次：${student && student.score ? student.score : '未填'} 分，${student && student.rank ? student.rank : '未填'} 名。`,
    `建议重点讨论：是否优先保 ${student && student.targetMajor ? student.targetMajor : '目标专业'}，以及是否接受组内调剂。`
  ]
  if (archiveText) {
    lines.push(`档案上下文：${archiveText}。`)
  }
  if (student && student.fromRecommend) {
    lines.push('当前考生已标记为通过推荐链路进入。')
  }
  if (Array.isArray(suggestions) && suggestions.length) {
    lines.push(`可直接继续看的关键词：${suggestions.map((item) => item.title).slice(0, 3).join('、')}。`)
  }
  lines.push('先统一志愿策略，再排正式志愿表，效率会高很多。')
  return lines.join('\n')
}

function buildArchiveText(student) {
  const safeStudent = student || {}
  const parts = [safeStudent.schoolName, safeStudent.schoolYear, safeStudent.className].filter(Boolean)
  return parts.length ? parts.join(' / ') : ''
}

function buildFamilySharePayload(title, student, familyBrief, checklist, suggestions) {
  return {
    mode: 'report',
    title: '给家长看的沟通卡',
    sourceTitle: title || 'AI 报考报告',
    student: student || null,
    summary: familyBrief ? familyBrief.split('\n').slice(0, 4) : [],
    checklist: (checklist || []).slice(0, 4),
    suggestions: (suggestions || []).slice(0, 3).map((item) => item.title),
    footer: '建议家长和考生先统一优先级，再决定正式志愿表排序。'
  }
}

function safeDecodeURIComponent(value, fallback) {
  if (value === undefined || value === null || value === '') {
    return fallback
  }
  try {
    return decodeURIComponent(value)
  } catch (err) {
    return value
  }
}

function safeParseJSON(value, fallback) {
  if (value === undefined || value === null || value === '') {
    return fallback
  }
  try {
    return JSON.parse(value)
  } catch (err) {
    return fallback
  }
}

Page({
  data: {
    shareGateReady: false,
    shareUnlocked: false,
    shareUnlockPending: false,
    gateChecking: true,
    title: '黑龙江 AI 报考报告',
    student: null,
    suggestions: [],
    reportBlocks: [],
    outline: [],
    stats: null,
    checklist: [],
    familyBrief: '',
    vipPrompt: null,
    showVipEntry: false
  },

  onLoad(query) {
    enableShareMenus()
    const fallbackPayload = getPendingReportPayload() || {}
    const studentValue = safeDecodeURIComponent(query.student, '')
    const suggestionsValue = safeDecodeURIComponent(query.suggestions, '')
    this.pendingQuery = {
      report: safeDecodeURIComponent(query.report, fallbackPayload.report || '暂无报告内容'),
      title: safeDecodeURIComponent(query.title, fallbackPayload.title || '黑龙江 AI 报考报告'),
      student: studentValue ? safeParseJSON(studentValue, fallbackPayload.student || null) : (fallbackPayload.student || null),
      suggestions: suggestionsValue ? safeParseJSON(suggestionsValue, fallbackPayload.suggestions || []) : (fallbackPayload.suggestions || [])
    }
    this.hydrateReport()
    this.setData({ shareGateReady: true, gateChecking: true })
    wx.setNavigationBarTitle({ title: this.data.title || 'AI 报考报告' })
    shouldRequireShareGate('aiReport', false, query.shareToken || '').then((required) => {
      if (required) {
        this.setData({ gateChecking: false })
        wx.setNavigationBarTitle({ title: '分享后查看 AI 报告' })
        return
      }
      this.unlockReport()
    }).catch(() => {
      this.unlockReport()
    })
  },

  onShow() {
    enableShareMenus()
  },

  hydrateReport() {
    if (!this.pendingQuery) {
      return
    }
    const report = this.pendingQuery.report
    const title = this.pendingQuery.title
    const student = this.pendingQuery.student
    const suggestions = this.pendingQuery.suggestions
    const reportBlocks = buildReportBlocks(report)
    this.fullReport = report
    this.setData({
      title,
      student,
      suggestions,
      reportBlocks,
      outline: buildOutline(reportBlocks),
      checklist: buildChecklist(student, suggestions),
      familyBrief: buildFamilyBrief(student, suggestions),
      vipPrompt: {
        title: '如果你还在反复摇摆，可以继续往深处做',
        desc: '现在这份报告解决的是“先看懂方向”。如果你还要和家长老师反复比较不同志愿策略，建议继续去方案对比库或 VIP 中心。',
        points: [
          '把稳妥版、保专业版、冲层次版并排对比',
          '保留多套方案，避免一次讨论后又要重做',
          '深度服务更适合临近填报时反复复盘'
        ]
      },
      stats: {
        blockCount: reportBlocks.length,
        charCount: report.length
      },
      archiveText: buildArchiveText(student),
      fromRecommendText: student && student.fromRecommend ? '推荐来源：是' : ''
    })
  },

  unlockReport() {
    if (!this.pendingQuery || this.data.shareUnlocked) {
      return
    }
    this.hydrateReport()
    this.setData({
      shareUnlocked: true,
      shareUnlockPending: false,
      gateChecking: false,
      shareGateReady: true
    })
    wx.setNavigationBarTitle({ title: this.data.title || '黑龙江 AI 报考报告' })
    saveReportHistory({ report: this.fullReport || '', student: this.data.student })
    this.syncVIPEntryVisibility(true)
  },

  requestShareUnlock() {
    this.setData({ shareUnlockPending: true })
  },

  syncVIPEntryVisibility(forceRefresh) {
	return getVIPEntryVisibility(forceRefresh).then((showVipEntry) => {
		if (this.data.showVipEntry !== showVipEntry) {
			this.setData({ showVipEntry })
		}
  }).catch(() => false)
  },

  copyReport() {
    wx.setClipboardData({
      data: this.fullReport || '',
      success: () => wx.showToast({ title: '报告已复制', icon: 'success' })
    })
  },

  copyFamilyBrief() {
    wx.setClipboardData({
      data: this.data.familyBrief,
      success: () => wx.showToast({ title: '已复制家长沟通版', icon: 'success' })
    })
  },

  backHome() {
    wx.switchTab ? wx.switchTab({ url: '/pages/index/index' }) : wx.reLaunch({ url: '/pages/index/index' })
  },

  openPlanList() {
    wx.navigateTo({ url: '/pages/plan-list/plan-list' })
  },

  openVip() {
    wx.navigateTo({ url: '/pages/vip/vip' })
  },

  openFamilyShareCard() {
    const payload = buildFamilySharePayload(
      this.data.title,
      this.data.student,
      this.data.familyBrief,
      this.data.checklist,
      this.data.suggestions
    )
    wx.navigateTo({
      url: '/pages/family-share/family-share?payload=' + encodeURIComponent(JSON.stringify(payload))
    })
  },

  openExploreSuggestion(e) {
    const keyword = e.currentTarget.dataset.keyword || ''
    const subject = e.currentTarget.dataset.subject || (this.data.student && this.data.student.subject) || '历史'
    savePendingExploreFilters({ keyword, subject })
    wx.switchTab({ url: '/pages/explore/explore' })
  },

  onShareAppMessage() {
    const self = this
    const shareToken = createShareGateToken('aiReport')
    savePendingReportPayload({
      report: this.fullReport || (this.pendingQuery && this.pendingQuery.report) || '',
      title: this.data.title || (this.pendingQuery && this.pendingQuery.title) || '黑龙江 AI 报考报告',
      student: this.data.student || (this.pendingQuery && this.pendingQuery.student) || null,
      suggestions: this.data.suggestions || (this.pendingQuery && this.pendingQuery.suggestions) || []
    })
    return {
      title: '黑龙江高报助手：查院校、看推荐、做 AI 报考分析',
      path: `/pages/report/report?shareToken=${encodeURIComponent(shareToken)}`,
      success() {
        if (self.data.shareUnlockPending && !self.data.shareUnlocked) {
          markShareGateUnlocked('aiReport')
          self.unlockReport()
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
    const shareToken = createShareGateToken('aiReport')
    savePendingReportPayload({
      report: this.fullReport || (this.pendingQuery && this.pendingQuery.report) || '',
      title: this.data.title || (this.pendingQuery && this.pendingQuery.title) || '黑龙江 AI 报考报告',
      student: this.data.student || (this.pendingQuery && this.pendingQuery.student) || null,
      suggestions: this.data.suggestions || (this.pendingQuery && this.pendingQuery.suggestions) || []
    })
    return {
      title: '黑龙江高报助手：查院校、看推荐、做 AI 报考分析',
      query: `shareToken=${encodeURIComponent(shareToken)}`,
      imageUrl: ''
    }
  }
})

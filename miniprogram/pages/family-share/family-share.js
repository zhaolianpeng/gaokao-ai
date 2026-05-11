const { getVIPEntryVisibility } = require('../../utils/vip-entry')

function decodePayload(value) {
  if (!value) {
    return null
  }
  try {
    return JSON.parse(decodeURIComponent(value))
  } catch (error) {
    return null
  }
}

function buildShareText(payload) {
  if (!payload) {
    return ''
  }
  const lines = [payload.title || '给家长看的沟通卡']
  const student = payload.student || {}
  if (student.subject || student.score || student.rank) {
    lines.push(`黑龙江 ${student.subject || ''}，${student.score || ''}分，${student.rank || ''}名`)
  }
  const archiveText = buildArchiveText(student)
  if (archiveText) {
    lines.push(`档案上下文：${archiveText}`)
  }
  if (student.fromRecommend) {
    lines.push('推荐来源：是')
  }
  ;(payload.summary || []).forEach((item) => lines.push(item))
  ;(payload.conclusions || []).forEach((item) => lines.push(item))
  ;(payload.footer ? [payload.footer] : []).forEach((item) => lines.push(item))
  return lines.join('\n')
}

function buildArchiveText(student) {
  const safeStudent = student || {}
  const parts = [safeStudent.schoolName, safeStudent.schoolYear, safeStudent.className].filter(Boolean)
  return parts.length ? parts.join(' / ') : ''
}

function buildPosterMetrics(payload) {
  if (!payload) {
    return []
  }
  const student = payload.student || {}
  return [
    { label: '科类', value: student.subject || '待补充' },
    { label: '分数', value: student.score ? `${student.score}` : '待补充' },
    { label: '位次', value: student.rank ? `${student.rank}` : '待补充' }
  ]
}

function buildPosterLead(payload) {
  if (!payload) {
    return ''
  }
  if (payload.conclusions && payload.conclusions.length) {
    return payload.conclusions[0]
  }
  if (payload.summary && payload.summary.length) {
    return payload.summary[0]
  }
  return payload.footer || ''
}

function buildSignature(payload) {
  if (!payload) {
    return ''
  }
  if (payload.mode === 'compare') {
    return '黑龙江高报助手 · 方案对比沟通版'
  }
  return '黑龙江高报助手 · AI 报告沟通版'
}

Page({
  data: {
    payload: null,
    shareText: '',
    posterMetrics: [],
    posterLead: '',
    signatureText: '',
    archiveText: '',
    fromRecommendText: '',
    showVipEntry: false
  },

  onLoad(query) {
    const payload = decodePayload(query.payload)
    const student = (payload && payload.student) || {}
    this.setData({
      payload,
      shareText: buildShareText(payload),
      posterMetrics: buildPosterMetrics(payload),
      posterLead: buildPosterLead(payload),
      signatureText: buildSignature(payload),
      archiveText: buildArchiveText(student),
      fromRecommendText: student.fromRecommend ? '推荐来源：是' : ''
    })
    wx.setNavigationBarTitle({ title: (payload && payload.title) || '家长沟通卡片' })
    this.syncVIPEntryVisibility()
  },

  syncVIPEntryVisibility() {
	return getVIPEntryVisibility().then((showVipEntry) => {
		if (this.data.showVipEntry !== showVipEntry) {
			this.setData({ showVipEntry })
		}
	})
  },

  copyCardText() {
    if (!this.data.shareText) {
      return
    }
    wx.setClipboardData({
      data: this.data.shareText,
      success: () => wx.showToast({ title: '已复制卡片文案', icon: 'success' })
    })
  },

  openPlanList() {
    wx.navigateTo({ url: '/pages/plan-list/plan-list' })
  },

  openVip() {
    wx.navigateTo({ url: '/pages/vip/vip' })
  },

  onShareAppMessage() {
    return {
      title: (this.data.payload && this.data.payload.title) || '给家长看的沟通卡片',
      path: '/pages/index/index'
    }
  }
})
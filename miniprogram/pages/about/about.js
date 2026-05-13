const { getNetworkDiagnostics, clearNetworkDiagnostics, getAuthUser, hasPrivacyConsent, savePrivacyConsent } = require('../../utils/storage')
const { request } = require('../../utils/request')

const BACKEND_PRESETS = [
  { label: '默认推荐线路', value: 'https://api.succ.online' },
  { label: '备用线路 1', value: 'https://82.156.54.232:443' },
  { label: '备用线路 2', value: 'http://82.156.54.232:8080' }
]

Page({
  data: {
    positioning: [
      { title: '我们在做什么', desc: '这是一个面向黑龙江高考志愿填报的辅助工具，帮助考生和家长把查数据、看梯度、做方案放到同一个入口里完成。' },
      { title: '我们怎么帮你判断', desc: '先把批次线、一分一段、院校和专业组看清楚，再给出 5 层推荐，方便判断哪些适合冲、哪些适合作为主力、哪些负责兜底。' },
      { title: '我们想解决什么问题', desc: '尽量把“信息太散、位次难理解、家长和考生意见难统一”这些填报前最常见的问题，讲得更直白、更容易讨论。' }
    ],
    audiences: [
      { title: '第一次正式填志愿的考生', desc: '适合先建立整体梯度概念，知道自己应该重点看哪些学校和专业组。' },
      { title: '需要一起决策的家长', desc: '推荐结果和沟通摘要适合直接一起看，减少来回解释和重复讨论。' },
      { title: '想反复比较多套方案的人', desc: '适合想从保学校、保专业、保城市几个角度反复比较，再慢慢定表的人。' }
    ],
    differentiators: [
      { title: '推荐理由尽量讲清楚', desc: '不是只给一个结论，而是把 5 层划分、位次依据、线差依据和推荐原因都摆出来，方便家长和考生一起判断。', badge: '讲清楚' },
      { title: '先快给结果，再补深分析', desc: '先给你能直接拿来讨论和排表的推荐结果，再在需要时补充更完整的 AI 分析。', badge: '先结果' },
      { title: '更适合家庭共同决策', desc: '页面会尽量把专业、学校、城市和录取把握拆开说，方便发给家长或老师一起讨论。', badge: '可共看' }
    ],
    aiExperienceModes: [
      { title: '先查清基础数据', desc: '先看批次线、一分一段、院校和专业组，把基础情况看明白。', tone: 'light' },
      { title: '再生成 5 层方案', desc: '推荐结果会分成冲刺、较冲、稳妥、较保、保底五层，更适合直接讨论填报顺序。', tone: 'mid' },
      { title: '最后补充深度分析', desc: '如果你还想继续比较学校层次、专业取舍和风险点，再看 AI 深度分析。', tone: 'strong' }
    ],
    usageGuide: [
      { title: '第一步：先把分数和位次查准', desc: '先在首页看批次线和一分一段，确认自己所在的大致梯度，不要一上来就盲目选学校。' },
      { title: '第二步：再看 5 层推荐结果', desc: '把冲刺、较冲、稳妥、较保、保底五层一起看，先知道哪些适合冲高，哪些适合作为主力，哪些负责兜底。' },
      { title: '第三步：和家长统一优先顺序', desc: '先说清楚这次到底是更想保学校、保专业还是保城市，再去看正式志愿表，会省掉很多来回争论。' },
      { title: '第四步：需要时再看深度分析', desc: '如果还拿不准，就再看 AI 深度分析和沟通摘要，把关键风险点和选择理由说明白。' }
    ],
    techBackground: [
      { title: '研发目标不是只做展示页', desc: '我们把批次线、一分一段、院校、专业组、历年录取数据和推荐逻辑打通，重点是让结果真正能支持填报决策。', badge: '数据底座' },
      { title: '推荐结果强调可解释', desc: '不是只给一个“推荐学校”，而是把 5 层梯度、位次参考、线差依据和推荐原因一起展示，方便家庭共同判断。', badge: '可解释' },
      { title: '技术链路兼顾速度和稳定性', desc: '基础查询尽量秒级返回，复杂分析再异步展开，目标是在高峰期也尽量保证页面可用、结果清晰。', badge: '稳定性' }
    ],
    feedbackContent: '',
    feedbackSubmitting: false,
    feedbackSuccessMessage: '',
    backendBaseUrl: '',
    backendBaseUrlInput: '',
    backendCheckMessage: '',
    latestDiagnostic: null,
    backendPresets: BACKEND_PRESETS,
    privacyConsentAgreed: false
  },

  onShow() {
    const app = getApp()
    const backendBaseUrl = (app.globalData && app.globalData.httpBaseUrl) || app.defaultHttpBaseUrl
    const latestDiagnostic = getNetworkDiagnostics()[0] || null
    this.setData({
      backendBaseUrl,
      backendBaseUrlInput: backendBaseUrl,
      latestDiagnostic,
      privacyConsentAgreed: hasPrivacyConsent()
    })
  },

  openServiceAgreement() {
    wx.navigateTo({ url: '/pages/service-agreement/service-agreement' })
  },

  openPrivacyPolicy() {
    wx.navigateTo({ url: '/pages/privacy-policy/privacy-policy' })
  },

  agreePolicies() {
    savePrivacyConsent({ agreed: true })
    this.setData({ privacyConsentAgreed: true })
    wx.showToast({ title: '已确认服务说明', icon: 'success' })
  },

  normalizeBackendBaseUrl(value) {
    return String(value || '').trim().replace(/\/+$/, '')
  },

  onBackendBaseUrlInput(e) {
    this.setData({ backendBaseUrlInput: e.detail.value })
  },

  useBackendPreset(e) {
    this.setData({
      backendBaseUrlInput: e.currentTarget.dataset.value,
      backendCheckMessage: ''
    })
  },

  applyBackendBaseUrl() {
    const app = getApp()
    const normalized = this.normalizeBackendBaseUrl(this.data.backendBaseUrlInput)
    if (!/^https?:\/\//.test(normalized)) {
      wx.showToast({ title: '地址需以 http:// 或 https:// 开头', icon: 'none' })
      return
    }
    const nextUrl = app.setHttpBaseUrl(normalized)
    this.setData({
      backendBaseUrl: nextUrl,
      backendBaseUrlInput: nextUrl,
      backendCheckMessage: `当前生效地址：${nextUrl}`
    })
    wx.showToast({ title: '已切换连接线路', icon: 'success' })
  },

  resetBackendBaseUrl() {
    const app = getApp()
    const nextUrl = app.resetHttpBaseUrl()
    this.setData({
      backendBaseUrl: nextUrl,
      backendBaseUrlInput: nextUrl,
      backendCheckMessage: `已恢复推荐地址：${nextUrl}`
    })
    wx.showToast({ title: '已恢复默认线路', icon: 'success' })
  },

  verifyBackendBaseUrl() {
    const target = this.normalizeBackendBaseUrl(this.data.backendBaseUrlInput || this.data.backendBaseUrl)
    if (!/^https?:\/\//.test(target)) {
      wx.showToast({ title: '请先输入完整地址', icon: 'none' })
      return
    }
    wx.showLoading({ title: '检测中', mask: true })
    wx.request({
      url: `${target}/healthz`,
      method: 'GET',
      timeout: 8000,
      success: (res) => {
        if (res.statusCode >= 200 && res.statusCode < 300) {
          const body = typeof res.data === 'string' ? res.data : JSON.stringify(res.data || {})
          this.setData({ backendCheckMessage: `健康检查通过：${target}/healthz -> ${body}` })
          wx.showToast({ title: '连接正常', icon: 'success' })
          return
        }
        this.setData({ backendCheckMessage: `健康检查失败：HTTP ${res.statusCode}` })
        wx.showToast({ title: `HTTP ${res.statusCode}`, icon: 'none' })
      },
      fail: (err) => {
        const message = (err && err.errMsg) || '请求失败'
        this.setData({ backendCheckMessage: `健康检查失败：${message}` })
        wx.showToast({ title: '连接异常', icon: 'none' })
      },
      complete: () => wx.hideLoading()
    })
  },

  openDiagnostics() {
    wx.navigateTo({ url: '/pages/diagnostics/diagnostics' })
  },

  copyLatestDiagnostic() {
    const item = this.data.latestDiagnostic
    if (!item) {
      wx.showToast({ title: '当前没有可复制的记录', icon: 'none' })
      return
    }
    wx.setClipboardData({
      data: item.message,
      success: () => wx.showToast({ title: '已复制排查信息', icon: 'success' })
    })
  },

  clearDiagnostics() {
    clearNetworkDiagnostics()
    this.setData({ latestDiagnostic: null })
    wx.showToast({ title: '已清空排查记录', icon: 'success' })

  },

  onFeedbackInput(e) {
    this.setData({
      feedbackContent: e.detail.value,
      feedbackSuccessMessage: ''
    })

  },

  async submitFeedback() {
    const content = String(this.data.feedbackContent || '').trim()
    if (!content) {
      wx.showToast({ title: '请先输入反馈内容', icon: 'none' })
      return
    }

    this.setData({ feedbackSubmitting: true })
    try {
      const user = getAuthUser() || {}
      const result = await request({
        url: '/api/about/feedback',
        method: 'POST',
        data: {
          content,
          page: 'pages/about/about',
          backendBaseUrl: this.data.backendBaseUrl,
          phone: user.phone || '',
          nickname: user.nickname || ''
        }
      })
      this.setData({
        feedbackContent: '',
        feedbackSuccessMessage: `已收到你的反馈，编号 ${result.id || '-'}。`
      })
      wx.showToast({ title: '已收到反馈', icon: 'success' })
    } catch (err) {
      wx.showToast({ title: (err && err.error) || '提交失败', icon: 'none' })
    } finally {
      this.setData({ feedbackSubmitting: false })
    }
  }
})
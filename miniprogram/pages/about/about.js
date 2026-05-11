const { getNetworkDiagnostics, clearNetworkDiagnostics, getAuthUser, hasPrivacyConsent, savePrivacyConsent } = require('../../utils/storage')
const { request } = require('../../utils/request')

const BACKEND_PRESETS = [
  { label: '推荐 HTTPS 域名', value: 'https://api.succ.online' },
  { label: 'IP:443 仅排障', value: 'https://82.156.54.232:443' },
  { label: '直连 8080 仅排障', value: 'http://82.156.54.232:8080' }
]

Page({
  data: {
    positioning: [
      { title: '产品定位', desc: '面向黑龙江高考志愿场景，把结构化数据查询、位次判断和方案生成放在一个入口里。 ' },
      { title: '核心方式', desc: '首页先给看板式数据，再进入推荐、AI 分析和家长沟通页，减少前置理解成本。' },
      { title: '当前版本', desc: '默认通过 HTTPS 域名访问自建后端，避免真机被微信合法域名机制拦截。' }
    ],
    audiences: [
      { title: '考生本人', desc: '先快速看批次线、位次和院校，再决定要不要生成方案。' },
      { title: '家长共看', desc: '推荐结果、报告和家长沟通卡片适合截图或一起讨论。' },
      { title: '反复比较人群', desc: '适合需要多方案对比、持续追踪数据和多轮讨论的用户。' }
    ],
    differentiators: [
      { title: '透明推荐逻辑', desc: '推荐页直接展示冲稳保划分、位次依据和推荐原因，不做黑盒结论。', badge: '可解释' },
      { title: '双层 AI 体验', desc: '快速填报方案走规则引擎，深度分析再走 AI 智能体，兼顾速度和深度。', badge: '快 + 深' },
      { title: '家长协同决策', desc: '结果页和报告页可直接整理成可分享摘要，适合发给家长或老师讨论。', badge: '易沟通' }
    ],
    aiExperienceModes: [
      { title: '即时查数据', desc: '院校、专业组、批次线、一分一段优先走结构化查询，尽量把等待压到 1-2 秒。', tone: 'light' },
      { title: '快速出方案', desc: '核心志愿推荐以位次匹配和规则引擎为主，先给能落地的冲稳保结果。', tone: 'mid' },
      { title: '深度 AI 分析', desc: '复杂需求和个性化建议交给 AI 智能体，页面用分阶段提示降低长等待焦虑。', tone: 'strong' }
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
    wx.showToast({ title: '已记录授权同意', icon: 'success' })
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
    wx.showToast({ title: '后端地址已切换', icon: 'success' })
  },

  resetBackendBaseUrl() {
    const app = getApp()
    const nextUrl = app.resetHttpBaseUrl()
    this.setData({
      backendBaseUrl: nextUrl,
      backendBaseUrlInput: nextUrl,
      backendCheckMessage: `已恢复推荐地址：${nextUrl}`
    })
    wx.showToast({ title: '已恢复默认地址', icon: 'success' })
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
          wx.showToast({ title: '检测通过', icon: 'success' })
          return
        }
        this.setData({ backendCheckMessage: `健康检查失败：HTTP ${res.statusCode}` })
        wx.showToast({ title: `HTTP ${res.statusCode}`, icon: 'none' })
      },
      fail: (err) => {
        const message = (err && err.errMsg) || '请求失败'
        this.setData({ backendCheckMessage: `健康检查失败：${message}` })
        wx.showToast({ title: '检测失败', icon: 'none' })
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
      wx.showToast({ title: '当前没有可复制的诊断记录', icon: 'none' })
      return
    }
    wx.setClipboardData({
      data: item.message,
      success: () => wx.showToast({ title: '已复制诊断信息', icon: 'success' })
    })
  },

  clearDiagnostics() {
    clearNetworkDiagnostics()
    this.setData({ latestDiagnostic: null })
    wx.showToast({ title: '已清空诊断记录', icon: 'success' })

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
        feedbackSuccessMessage: `已提交，反馈编号 ${result.id || '-'}。`
      })
      wx.showToast({ title: '反馈已提交', icon: 'success' })
    } catch (err) {
      wx.showToast({ title: (err && err.error) || '提交失败', icon: 'none' })
    } finally {
      this.setData({ feedbackSubmitting: false })
    }
  }
})
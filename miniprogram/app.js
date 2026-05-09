const LEGACY_BACKEND_URLS = {
  'http://82.156.54.232:80': 'https://api.succ.online',
  'https://82.156.54.232:443': 'https://api.succ.online',
  'http://82.156.54.232:8080': 'https://api.succ.online'
}

function normalizeBackendBaseUrl(value, fallback) {
  const normalized = String(value || '').trim().replace(/\/+$/, '') || fallback
  return LEGACY_BACKEND_URLS[normalized] || normalized
}

App({
  defaultHttpBaseUrl: 'https://api.succ.online',

  globalData: {
    cloudAvailable: false,
    cloudReady: false,
    httpBaseUrl: '',
    user: null,
    profile: null,
    pendingWechatProfileCompletion: false
  },

  onLaunch() {
    if (wx.cloud) {
      wx.cloud.init({
        env: wx.cloud.DYNAMIC_CURRENT_ENV,
        traceUser: true
      })
      this.globalData.cloudAvailable = true
      this.globalData.cloudReady = true
    }
    const storedBaseUrl = wx.getStorageSync('backendBaseUrl') || this.defaultHttpBaseUrl
    const nextBaseUrl = normalizeBackendBaseUrl(storedBaseUrl, this.defaultHttpBaseUrl)
    this.globalData.httpBaseUrl = nextBaseUrl
    if (nextBaseUrl !== storedBaseUrl) {
      wx.setStorageSync('backendBaseUrl', nextBaseUrl)
    }
    this.globalData.user = wx.getStorageSync('authUser') || null
    this.globalData.profile = wx.getStorageSync('userProfile') || null
  },

  setHttpBaseUrl(baseUrl) {
    const normalized = normalizeBackendBaseUrl(baseUrl, this.defaultHttpBaseUrl)
    this.globalData.httpBaseUrl = normalized
    wx.setStorageSync('backendBaseUrl', normalized)
    return normalized
  },

  resetHttpBaseUrl() {
    this.globalData.httpBaseUrl = this.defaultHttpBaseUrl
    wx.removeStorageSync('backendBaseUrl')
    return this.globalData.httpBaseUrl
  },

  setUser(user) {
    this.globalData.user = user || null
    wx.setStorageSync('authUser', this.globalData.user)
  },

  setProfile(profile) {
    this.globalData.profile = profile || null
    wx.setStorageSync('userProfile', this.globalData.profile)
  }
})

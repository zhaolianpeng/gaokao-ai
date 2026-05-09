App({
  defaultHttpBaseUrl: 'http://82.156.54.232:80',

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
    this.globalData.httpBaseUrl = wx.getStorageSync('backendBaseUrl') || this.defaultHttpBaseUrl
    this.globalData.user = wx.getStorageSync('authUser') || null
    this.globalData.profile = wx.getStorageSync('userProfile') || null
  },

  setHttpBaseUrl(baseUrl) {
    const normalized = String(baseUrl || '').trim().replace(/\/+$/, '') || this.defaultHttpBaseUrl
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

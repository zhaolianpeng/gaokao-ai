App({
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

    this.globalData.httpBaseUrl = wx.getStorageSync('backendBaseUrl') || 'http://82.156.54.232'
    this.globalData.user = wx.getStorageSync('authUser') || null
    this.globalData.profile = wx.getStorageSync('userProfile') || null
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
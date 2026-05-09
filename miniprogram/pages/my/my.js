const { getAuthUser, saveAuthUser, clearAuthUser, getUserProfile, saveUserProfile, clearUserProfile } = require('../../utils/storage')
const { request } = require('../../utils/request')

function wxGetUserProfile() {
  return new Promise((resolve, reject) => {
    wx.getUserProfile({
      desc: '用于补全头像和昵称',
      success: (res) => resolve((res && res.userInfo) || {}),
      fail: reject
    })
  })
}

function cacheRemoteAvatar(url) {
  return new Promise((resolve) => {
    if (!url) {
      resolve('')
      return
    }
    wx.getImageInfo({
      src: url,
      success: (res) => resolve((res && res.path) || url),
      fail: () => resolve(url)
    })
  })
}

function isAnonymousWechatNickname(nickname) {
  const text = String(nickname || '').trim()
  return text === '微信用户' || /^微信用户\d*$/.test(text)
}

Page({
  data: {
    user: null,
    userInitial: '考',
    avatarDisplayUrl: '',
    profile: {
      province: '黑龙江',
      subject: '历史',
      score: '',
      rank: '',
      targetMajor: '',
      notes: ''
    },
    subjectOptions: ['历史', '物理']
  },

  onShow() {
    const app = getApp()
    const user = getAuthUser()
    const profile = getUserProfile() || this.data.profile
    app.setUser(user)
    app.setProfile(profile)
    this.setData({
      user,
      userInitial: this.getUserInitial(user),
      avatarDisplayUrl: this.getDisplayAvatarUrl(user),
      profile
    })
  },

  getUserInitial(user) {
    const nickname = user && user.nickname ? String(user.nickname).trim() : ''
    return nickname ? nickname.slice(0, 1) : '考'
  },

  getDisplayAvatarUrl(user) {
    return (user && (user.avatarLocalPath || user.avatarUrl)) || ''
  },

  openVip() {
    wx.navigateTo({ url: '/pages/vip/vip' })
  },

  openAboutPage() {
    wx.navigateTo({ url: '/pages/about/about' })
  },

  async syncWechatProfile() {
    const user = getAuthUser()
    if (!user || !user.id || user.storageMode !== 'server') {
      wx.showToast({ title: '请先完成微信手机号登录', icon: 'none' })
      return
    }

    try {
      const profile = await wxGetUserProfile()
      if (!profile.avatarUrl) {
        wx.showToast({ title: '未获取到微信头像', icon: 'none' })
        return
      }

      const payload = await request({
        url: '/api/auth/wx-profile',
        method: 'POST',
        data: {
          userId: user.id,
          phone: user.phone || '',
          nickname: profile.nickName || user.nickname || '',
          avatarUrl: profile.avatarUrl
        }
      })

      const avatarLocalPath = await cacheRemoteAvatar((payload && payload.avatarUrl) || (payload && payload.user && payload.user.avatarUrl) || profile.avatarUrl)

      const nextUser = saveAuthUser({
        ...user,
        ...(payload && payload.user ? payload.user : payload),
        nickname: !isAnonymousWechatNickname((payload && payload.nickname) || (payload && payload.user && payload.user.nickname) || profile.nickName)
          ? ((payload && payload.nickname) || (payload && payload.user && payload.user.nickname) || profile.nickName)
          : (user.nickname || ''),
        avatarUrl: (payload && payload.avatarUrl) || (payload && payload.user && payload.user.avatarUrl) || profile.avatarUrl,
        avatarLocalPath,
        storageMode: 'server'
      })

      getApp().setUser(nextUser)
      this.setData({
        user: nextUser,
        userInitial: this.getUserInitial(nextUser),
        avatarDisplayUrl: this.getDisplayAvatarUrl(nextUser)
      })
      wx.showToast({ title: '头像昵称已同步', icon: 'success' })
    } catch (err) {
      const message = (err && err.error) || (err && err.message) || '同步失败'
      wx.showToast({ title: message, icon: 'none' })
    }
  },

  goLogin() {
    wx.navigateTo({ url: '/pages/login/login' })
  },

  onSubjectChange(e) {
    const subject = this.data.subjectOptions[e.detail.value]
    this.setData({ 'profile.subject': subject })
  },

  onInput(e) {
    const field = e.currentTarget.dataset.field
    this.setData({ [`profile.${field}`]: e.detail.value })
  },

  saveProfile() {
    if (!this.data.user) {
      wx.showToast({ title: '请先登录', icon: 'none' })
      return
    }
    const profile = saveUserProfile({
      ...this.data.profile,
      score: Number(this.data.profile.score || 0),
      rank: Number(this.data.profile.rank || 0)
    })
    getApp().setProfile(profile)
    this.setData({ profile })
    wx.showToast({ title: '已保存资料', icon: 'success' })
  },

  useDemoAccount() {
    const user = saveAuthUser({
      nickname: '黑龙江考生',
      phone: '13800000000',
      loginType: 'demo',
      storageMode: 'local'
    })
    getApp().setUser(user)
    this.setData({
      user,
      userInitial: this.getUserInitial(user),
      avatarDisplayUrl: this.getDisplayAvatarUrl(user)
    })
    wx.showToast({ title: '已快速登录', icon: 'success' })
  },

  logout() {
    clearAuthUser()
    clearUserProfile()
    getApp().setUser(null)
    getApp().setProfile(null)
    this.setData({
      user: null,
      userInitial: '考',
      avatarDisplayUrl: '',
      profile: {
        province: '黑龙江',
        subject: '历史',
        score: '',
        rank: '',
        targetMajor: '',
        notes: ''
      }
    })
    wx.showToast({ title: '已退出登录', icon: 'success' })
  }
})
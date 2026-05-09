const { getAuthUser, saveAuthUser, clearAuthUser, getUserProfile, saveUserProfile, clearUserProfile } = require('../../utils/storage')
const { request } = require('../../utils/request')

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
    syncDraftNickname: '',
    syncDraftAvatarPath: '',
    syncSubmitting: false,
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
      syncDraftNickname: (user && user.nickname) || '',
      syncDraftAvatarPath: (user && user.avatarLocalPath) || '',
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

  onSyncNicknameInput(e) {
    this.setData({ syncDraftNickname: e.detail.value })
  },

  onChooseAvatar(e) {
    const avatarUrl = e && e.detail ? e.detail.avatarUrl : ''
    if (!avatarUrl) {
      wx.showToast({ title: '未获取到头像', icon: 'none' })
      return
    }
    this.setData({
      syncDraftAvatarPath: avatarUrl,
      avatarDisplayUrl: avatarUrl
    })
  },

  async syncWechatProfile() {
    const user = getAuthUser()
    if (!user || !user.id || user.storageMode !== 'server') {
      wx.showToast({ title: '请先完成微信手机号登录', icon: 'none' })
      return
    }

    const nextNickname = String(this.data.syncDraftNickname || '').trim()
    const nextAvatarPath = String(this.data.syncDraftAvatarPath || '').trim()
    if (!nextNickname && !nextAvatarPath) {
      wx.showToast({ title: '请先填写昵称或选择头像', icon: 'none' })
      return
    }

    try {
      this.setData({ syncSubmitting: true })

      let avatarLocalPath = nextAvatarPath
      if (nextAvatarPath && /^https?:\/\//.test(nextAvatarPath)) {
        avatarLocalPath = await cacheRemoteAvatar(nextAvatarPath)
      }

      const payload = await request({
        url: '/api/auth/wx-profile',
        method: 'POST',
        data: {
          userId: user.id,
          phone: user.phone || '',
          nickname: nextNickname || user.nickname || '考生用户',
          avatarUrl: user.avatarUrl || ''
        }
      })

      const serverAvatarUrl = (payload && payload.avatarUrl) || (payload && payload.user && payload.user.avatarUrl) || user.avatarUrl || ''
      const finalAvatarLocalPath = avatarLocalPath || user.avatarLocalPath || ''
      const finalNickname = !isAnonymousWechatNickname((payload && payload.nickname) || (payload && payload.user && payload.user.nickname) || nextNickname)
        ? ((payload && payload.nickname) || (payload && payload.user && payload.user.nickname) || nextNickname)
        : (user.nickname || nextNickname || '考生用户')

      const nextUser = saveAuthUser({
        ...user,
        ...(payload && payload.user ? payload.user : payload),
        nickname: finalNickname,
        avatarUrl: serverAvatarUrl,
        avatarLocalPath: finalAvatarLocalPath,
        storageMode: 'server'
      })

      getApp().setUser(nextUser)
      this.setData({
        user: nextUser,
        userInitial: this.getUserInitial(nextUser),
        avatarDisplayUrl: this.getDisplayAvatarUrl(nextUser),
        syncDraftNickname: nextUser.nickname || '',
        syncDraftAvatarPath: nextUser.avatarLocalPath || ''
      })
      wx.showToast({ title: '头像昵称已同步', icon: 'success' })
    } catch (err) {
      const message = (err && err.error) || (err && err.message) || '同步失败'
      wx.showToast({ title: message, icon: 'none' })
      this.setData({ avatarDisplayUrl: this.getDisplayAvatarUrl(user) })
    } finally {
      this.setData({ syncSubmitting: false })
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
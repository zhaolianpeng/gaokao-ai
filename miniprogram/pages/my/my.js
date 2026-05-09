const { getAuthUser, saveAuthUser, clearAuthUser, getUserProfile, saveUserProfile, clearUserProfile } = require('../../utils/storage')
const { request, uploadFile } = require('../../utils/request')

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

function normalizeDisplayUser(user) {
  if (!user) {
    return null
  }

  return {
    ...user,
    nickname: isAnonymousWechatNickname(user.nickname) ? '考生用户' : user.nickname
  }
}

Page({
  data: {
    user: null,
    userInitial: '考',
    avatarDisplayUrl: '',
    syncDraftNickname: '',
    syncSubmitting: false,
    avatarSubmitting: false,
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
    const user = normalizeDisplayUser(getAuthUser())
    const profile = getUserProfile() || this.data.profile
    app.setUser(user)
    app.setProfile(profile)
    this.setData({
      user,
      userInitial: this.getUserInitial(user),
      avatarDisplayUrl: this.getDisplayAvatarUrl(user),
      syncDraftNickname: (user && user.nickname) || '',
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

  openProfileEditor() {
    wx.navigateTo({ url: '/pages/profile-edit/profile-edit' })
  },

  openAboutPage() {
    wx.navigateTo({ url: '/pages/about/about' })
  },

  async onChooseAvatar(e) {
    const user = getAuthUser()
    if (!user || !user.id || user.storageMode !== 'server') {
      wx.showToast({ title: '请先完成手机号快捷登录', icon: 'none' })
      return
    }

    const avatarPath = e && e.detail ? String(e.detail.avatarUrl || '').trim() : ''
    if (!avatarPath) {
      wx.showToast({ title: '未获取到头像文件', icon: 'none' })
      return
    }

    try {
      this.setData({ avatarSubmitting: true })
      const payload = await uploadFile({
        url: '/api/auth/wx-avatar',
        filePath: avatarPath,
        name: 'avatar',
        formData: {
          userId: user.id
        }
      })
      const nextUser = saveAuthUser({
        ...user,
        ...(payload && payload.user ? payload.user : payload),
        avatarUrl: (payload && payload.avatarUrl) || (payload && payload.user && payload.user.avatarUrl) || user.avatarUrl || '',
        avatarLocalPath: avatarPath,
        storageMode: 'server'
      })
      getApp().setUser(nextUser)
      this.setData({
        user: nextUser,
        userInitial: this.getUserInitial(nextUser),
        avatarDisplayUrl: this.getDisplayAvatarUrl(nextUser)
      })
      wx.showToast({ title: '头像已更新', icon: 'success' })
    } catch (err) {
      const message = (err && err.error) || (err && err.message) || '头像上传失败'
      wx.showToast({ title: message, icon: 'none' })
    } finally {
      this.setData({ avatarSubmitting: false })
    }
  },

  goLogin() {
    wx.navigateTo({ url: '/pages/login/login' })
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
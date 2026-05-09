const { getAuthUser, saveAuthUser, getUserProfile, saveUserProfile } = require('../../utils/storage')
const { request } = require('../../utils/request')

function isAnonymousWechatNickname(nickname) {
  const text = String(nickname || '').trim()
  return text === '微信用户' || /^微信用户\d*$/.test(text)
}

Page({
  data: {
    user: null,
    nickname: '',
    syncSubmitting: false,
    saveSubmitting: false,
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
    const user = getAuthUser()
    const profile = getUserProfile() || this.data.profile
    this.setData({
      user,
      nickname: (user && user.nickname) || '',
      profile: {
        ...this.data.profile,
        ...profile
      }
    })
  },

  onNicknameInput(e) {
    this.setData({ nickname: e.detail.value })
  },

  onSubjectChange(e) {
    const subject = this.data.subjectOptions[e.detail.value]
    this.setData({ 'profile.subject': subject })
  },

  onInput(e) {
    const field = e.currentTarget.dataset.field
    this.setData({ [`profile.${field}`]: e.detail.value })
  },

  async syncNickname() {
    const user = getAuthUser()
    if (!user || !user.id || user.storageMode !== 'server') {
      wx.showToast({ title: '请先完成手机号快捷登录', icon: 'none' })
      return
    }

    const nickname = String(this.data.nickname || '').trim()
    if (!nickname) {
      wx.showToast({ title: '请先填写昵称', icon: 'none' })
      return
    }

    try {
      this.setData({ syncSubmitting: true })
      const payload = await request({
        url: '/api/auth/wx-profile',
        method: 'POST',
        data: {
          userId: user.id,
          phone: user.phone || '',
          nickname,
          avatarUrl: user.avatarUrl || ''
        }
      })
      const rawUser = (payload && payload.user) || payload || {}
      const nextUser = saveAuthUser({
        ...user,
        ...rawUser,
        nickname: !isAnonymousWechatNickname(rawUser.nickname) ? rawUser.nickname : nickname,
        avatarUrl: rawUser.avatarUrl || user.avatarUrl || '',
        avatarLocalPath: user.avatarLocalPath || '',
        storageMode: 'server'
      })
      getApp().setUser(nextUser)
      this.setData({
        user: nextUser,
        nickname: nextUser.nickname || nickname
      })
      wx.showToast({ title: '昵称已同步', icon: 'success' })
    } catch (err) {
      const message = (err && err.error) || (err && err.message) || '同步失败'
      wx.showToast({ title: message, icon: 'none' })
    } finally {
      this.setData({ syncSubmitting: false })
    }
  },

  saveProfile() {
    if (this.data.saveSubmitting) {
      return
    }
    this.setData({ saveSubmitting: true })
    try {
      const profile = saveUserProfile({
        ...this.data.profile,
        score: Number(this.data.profile.score || 0),
        rank: Number(this.data.profile.rank || 0)
      })
      getApp().setProfile(profile)
      this.setData({ profile })
      wx.showToast({ title: '已保存资料', icon: 'success' })
    } finally {
      this.setData({ saveSubmitting: false })
    }
  }
})
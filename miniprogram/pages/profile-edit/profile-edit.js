const { getAuthUser, saveAuthUser, getUserProfile, saveUserProfile, hasPrivacyConsent } = require('../../utils/storage')
const { request } = require('../../utils/request')

function ensurePrivacyConsent(actionText) {
  if (hasPrivacyConsent()) {
    return true
  }
  wx.showModal({
    title: '请先完成授权同意',
    content: `在${actionText}前，请先阅读并同意《用户服务协议》和《隐私政策》。`,
    confirmText: '去查看',
    success(res) {
      if (res.confirm) {
        wx.navigateTo({ url: '/pages/about/about' })
      }
    }
  })
  return false
}

function isAnonymousWechatNickname(nickname) {
  const text = String(nickname || '').trim()
  return text === '微信用户' || /^微信用户\d*$/.test(text)
}

function normalizePersistedNickname(nickname) {
  const text = String(nickname || '').trim()
  if (!text || text === '考生用户' || isAnonymousWechatNickname(text)) {
    return ''
  }
  return text
}

function defaultProfile() {
  return {
    province: '黑龙江',
    subject: '历史',
    score: '',
    rank: '',
    targetMajor: '',
    notes: '',
    schoolName: '',
    schoolYear: '',
    className: '',
    studentNo: '',
    fromRecommend: false
  }
}

function mergeProfile(profile, user) {
  const hasUserRecommend = !!(user && Object.prototype.hasOwnProperty.call(user, 'fromRecommend'))
  return {
    ...defaultProfile(),
    ...(profile || {}),
    schoolName: (user && user.schoolName) || (profile && profile.schoolName) || '',
    schoolYear: (user && user.schoolYear) || (profile && profile.schoolYear) || '',
    className: (user && user.className) || (profile && profile.className) || '',
    studentNo: (user && user.studentNo) || (profile && profile.studentNo) || '',
    fromRecommend: hasUserRecommend ? !!user.fromRecommend : !!(profile && profile.fromRecommend)
  }
}

function findOptionIndex(options, value) {
  const list = Array.isArray(options) ? options : []
  for (let index = 0; index < list.length; index += 1) {
    if ((list[index] && list[index].value) === value) {
      return index
    }
  }
  return -1
}

Page({
  data: {
    user: null,
    nickname: '',
    syncSubmitting: false,
    saveSubmitting: false,
    optionsLoading: false,
    profile: defaultProfile(),
    subjectOptions: ['历史', '物理'],
    schoolOptions: [],
    schoolYearOptions: [],
    classOptions: []
  },

  onLoad() {
    this.loadProfileOptions()
  },

  onShow() {
    const user = getAuthUser()
    const profile = mergeProfile(getUserProfile(), user)
    this.setData({
      user,
      nickname: (user && user.nickname) || '',
      profile
    })
  },

  onNicknameInput(e) {
    this.setData({ nickname: e.detail.value })
  },

  onSubjectChange(e) {
    const subject = this.data.subjectOptions[e.detail.value]
    this.setData({ 'profile.subject': subject })
  },

  onSchoolChange(e) {
    const option = this.data.schoolOptions[e.detail.value]
    this.setData({ 'profile.schoolName': (option && option.value) || '' })
  },

  onSchoolYearChange(e) {
    const option = this.data.schoolYearOptions[e.detail.value]
    this.setData({ 'profile.schoolYear': (option && option.value) || '' })
  },

  onClassChange(e) {
    const option = this.data.classOptions[e.detail.value]
    this.setData({ 'profile.className': (option && option.value) || '' })
  },

  onFromRecommendChange(e) {
    this.setData({ 'profile.fromRecommend': !!(e && e.detail ? e.detail.value : false) })
  },

  onInput(e) {
    const field = e.currentTarget.dataset.field
    this.setData({ [`profile.${field}`]: e.detail.value })
  },

  async loadProfileOptions() {
    try {
      this.setData({ optionsLoading: true })
      const data = await request({
        url: '/api/profile-options',
        method: 'POST',
        data: {}
      })
      this.setData({
        schoolOptions: data.schools || [],
        schoolYearOptions: data.schoolYears || [],
        classOptions: data.classNames || []
      })
    } catch (err) {
      this.setData({ schoolOptions: [], schoolYearOptions: [], classOptions: [] })
    } finally {
      this.setData({ optionsLoading: false })
    }
  },

  async syncNickname() {
    if (!ensurePrivacyConsent('同步昵称')) {
      return
    }
    const user = getAuthUser()
    if (!user || !user.id || user.storageMode !== 'server') {
      wx.showToast({ title: '请先完成手机号快捷登录', icon: 'none' })
      return
    }

    const nickname = normalizePersistedNickname(this.data.nickname)
    if (!nickname) {
      wx.showToast({ title: '请先填写真实昵称', icon: 'none' })
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
        nickname: normalizePersistedNickname(rawUser.nickname) || nickname,
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
    if (!ensurePrivacyConsent('保存档案')) {
      return
    }
    const user = getAuthUser()
    this.setData({ saveSubmitting: true })
    const nextProfile = {
      ...this.data.profile,
      score: Number(this.data.profile.score || 0),
      rank: Number(this.data.profile.rank || 0),
      fromRecommend: !!this.data.profile.fromRecommend
    }
    const persistedNickname = normalizePersistedNickname(this.data.nickname || user.nickname)

    if (user && user.id && user.storageMode === 'server' && !persistedNickname) {
      this.setData({ saveSubmitting: false })
      wx.showToast({ title: '请先填写真实昵称', icon: 'none' })
      return
    }

    const finalize = (profile) => {
      const savedProfile = saveUserProfile(profile)
      getApp().setProfile(savedProfile)
      this.setData({ profile: savedProfile })
      wx.showToast({ title: '已保存资料', icon: 'success' })
      setTimeout(() => {
        wx.switchTab({ url: '/pages/my/my' })
      }, 300)
    }

    if (!user || !user.id || user.storageMode !== 'server') {
      try {
        finalize(nextProfile)
      } finally {
        this.setData({ saveSubmitting: false })
      }
      return
    }

    request({
      url: '/api/auth/wx-profile',
      method: 'POST',
      data: {
        userId: user.id,
        phone: user.phone || '',
        nickname: persistedNickname,
        avatarUrl: user.avatarUrl || '',
        schoolName: nextProfile.schoolName || '',
        schoolYear: nextProfile.schoolYear || '',
        className: nextProfile.className || '',
        studentNo: nextProfile.studentNo || '',
        fromRecommend: !!nextProfile.fromRecommend
      }
    }).then((payload) => {
      const rawUser = (payload && payload.user) || payload || {}
      const nextUser = saveAuthUser({
        ...user,
        ...rawUser,
        nickname: normalizePersistedNickname(rawUser.nickname) || persistedNickname,
        avatarUrl: rawUser.avatarUrl || user.avatarUrl || '',
        avatarLocalPath: user.avatarLocalPath || '',
        storageMode: 'server'
      })
      getApp().setUser(nextUser)
      this.setData({ user: nextUser, nickname: nextUser.nickname || this.data.nickname })
      finalize(mergeProfile(nextProfile, nextUser))
    }).catch((err) => {
      const message = (err && err.error) || (err && err.message) || '保存失败'
      wx.showToast({ title: message, icon: 'none' })
    }).finally(() => {
      this.setData({ saveSubmitting: false })
    })
  }
})
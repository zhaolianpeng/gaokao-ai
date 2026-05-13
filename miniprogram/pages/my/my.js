const { getAuthUser, saveAuthUser, clearAuthUser, getUserProfile, saveUserProfile, clearUserProfile, hasPrivacyConsent } = require('../../utils/storage')
const { request, uploadFile } = require('../../utils/request')
const { getVIPEntryVisibility } = require('../../utils/vip-entry')

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

Page({
  data: {
    user: null,
    userInitial: '考',
    avatarDisplayUrl: '',
    syncDraftNickname: '',
    syncSubmitting: false,
    avatarSubmitting: false,
    profile: defaultProfile(),
    subjectOptions: ['历史', '物理'],
	showVipEntry: false,
	vipMembership: null,
	vipMembershipLoaded: false
  },

  onShow() {
    const app = getApp()
    const rawUser = getAuthUser()
    const user = normalizeDisplayUser(rawUser)
    const profile = mergeProfile(getUserProfile(), rawUser)
    app.setProfile(profile)
    this.setData({
      user,
      userInitial: this.getUserInitial(user),
      avatarDisplayUrl: this.getDisplayAvatarUrl(user),
      syncDraftNickname: (user && user.nickname) || '',
      profile
    })
    this.syncVIPEntryVisibility(true)
  this.loadVIPMembership(rawUser)
  },

  loadVIPMembership(user) {
    if (!user || user.storageMode !== 'server' || !user.id) {
    this.setData({ vipMembership: null, vipMembershipLoaded: true })
    return Promise.resolve()
    }
    return request({
    url: '/api/vip/membership',
    method: 'POST',
    data: {
      userId: String(user.id)
    }
    }).then((vipMembership) => {
    const normalized = vipMembership && (vipMembership.productId || vipMembership.productName || vipMembership.statusText)
      ? vipMembership
      : null
    this.setData({ vipMembership: normalized, vipMembershipLoaded: true })
    }).catch(() => {
    this.setData({ vipMembership: null, vipMembershipLoaded: true })
    })
  },

   syncVIPEntryVisibility(forceRefresh) {
  return getVIPEntryVisibility(forceRefresh).then((showVipEntry) => {
		if (this.data.showVipEntry !== showVipEntry) {
			this.setData({ showVipEntry })
		}
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
    if (!ensurePrivacyConsent('上传头像')) {
      return
    }
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
      const displayUser = normalizeDisplayUser(nextUser)
      this.setData({
        user: displayUser,
        userInitial: this.getUserInitial(displayUser),
        avatarDisplayUrl: this.getDisplayAvatarUrl(displayUser)
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
    const displayUser = normalizeDisplayUser(user)
    this.setData({
      user: displayUser,
      userInitial: this.getUserInitial(displayUser),
      avatarDisplayUrl: this.getDisplayAvatarUrl(displayUser)
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
      vipMembership: null,
      vipMembershipLoaded: false,
      profile: {
        ...defaultProfile()
      }
    })
    wx.showToast({ title: '已退出登录', icon: 'success' })
  }
})
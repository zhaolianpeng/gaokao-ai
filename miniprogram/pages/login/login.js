const { getAuthUser, saveAuthUser } = require('../../utils/storage')
const { request } = require('../../utils/request')

function wxLogin() {
  return new Promise((resolve, reject) => {
    wx.login({
      success: resolve,
      fail: reject
    })
  })
}

function wxGetUserProfile() {
  return new Promise((resolve, reject) => {
    wx.getUserProfile({
      desc: '用于补全头像和昵称',
      success: (res) => resolve((res && res.userInfo) || {}),
      fail: reject
    })
  })
}

function normalizeAuthUser(payload) {
  const candidates = [
    payload,
    payload && payload.user,
    payload && payload.data,
    payload && payload.data && payload.data.user,
    payload && payload.result,
    payload && payload.result && payload.result.user,
    payload && payload.authUser,
    payload && payload.userInfo
  ].filter(Boolean)

  return candidates.find((item) => item && (item.id || item.openid || item.phone)) || null
}

function isAnonymousWechatNickname(nickname) {
  const text = String(nickname || '').trim()
  return text === '微信用户' || /^微信用户\d*$/.test(text)
}

Page({
  data: {
    nickname: '',
    phone: '',
    loading: false
  },

  onShow() {
    const user = getAuthUser()
    if (user) {
      this.setData({ nickname: user.nickname || '', phone: user.phone || '' })
    }
  },

  onInput(e) {
    const field = e.currentTarget.dataset.field
    this.setData({ [field]: e.detail.value })
  },

  async onGetPhoneNumber(e) {
    const code = e.detail && e.detail.code
    if (!code) {
      wx.showToast({ title: '未获取到手机号授权', icon: 'none' })
      return
    }

    this.setData({ loading: true })
    try {
      const loginRes = await wxLogin()
      if (!loginRes.code) {
        throw new Error('wx.login 失败')
      }

      const payload = await request({
        url: '/api/auth/wx-login',
        method: 'POST',
        data: {
          code,
          loginCode: loginRes.code
        }
      })

      const loginUser = normalizeAuthUser(payload)
      if (!loginUser) {
        throw new Error('登录返回数据不完整，请重试')
      }

      const inputNickname = (this.data.nickname || '').trim()
      let profileFromWechat = {}
      try {
        profileFromWechat = await wxGetUserProfile()
      } catch (profileErr) {
      }

      const mergedUser = {
        ...loginUser,
        nickname: !isAnonymousWechatNickname(profileFromWechat.nickName) ? profileFromWechat.nickName : (inputNickname || loginUser.nickname || '考生用户'),
        avatarUrl: profileFromWechat.avatarUrl || loginUser.avatarUrl || '',
        storageMode: 'server'
      }

      let authUser = saveAuthUser(mergedUser)
      if (!authUser.id) {
        throw new Error('未拿到服务端用户ID，请重新登录')
      }

      getApp().setUser(authUser)
      this.setData({
        nickname: authUser.nickname || '',
        phone: authUser.phone || ''
      })

      if (mergedUser.avatarUrl) {
        try {
          const profileUser = await request({
            url: '/api/auth/wx-profile',
            method: 'POST',
            data: {
              userId: authUser.id,
              phone: authUser.phone || '',
              nickname: mergedUser.nickname || authUser.nickname || '',
              avatarUrl: mergedUser.avatarUrl
            }
          })
          const normalizedProfileUser = normalizeAuthUser(profileUser) || mergedUser
          authUser = saveAuthUser({
            ...authUser,
            ...normalizedProfileUser,
            nickname: !isAnonymousWechatNickname(normalizedProfileUser.nickname) ? normalizedProfileUser.nickname : (authUser.nickname || mergedUser.nickname || '考生用户'),
            avatarUrl: normalizedProfileUser.avatarUrl || mergedUser.avatarUrl || authUser.avatarUrl || '',
            storageMode: 'server'
          })
          getApp().setUser(authUser)
          this.setData({
            nickname: authUser.nickname || '',
            phone: authUser.phone || ''
          })
        } catch (profileErr) {
        }
      }

      wx.showToast({ title: '登录成功', icon: 'success' })
      setTimeout(() => {
        wx.switchTab({ url: '/pages/my/my' })
      }, 300)
    } catch (err) {
      const message = (err && err.error) || (err && err.message) || '登录失败'
      wx.showToast({ title: message, icon: 'none' })
    } finally {
      this.setData({ loading: false })
    }
  },

  submitDemo() {
    const nickname = (this.data.nickname || '').trim() || '体验考生'
    const phone = (this.data.phone || '').trim()
    const user = saveAuthUser({ nickname, phone, loginType: 'demo', storageMode: 'local' })
    getApp().setUser(user)
    wx.showToast({ title: '已进入体验模式', icon: 'success' })
    setTimeout(() => {
      wx.switchTab({ url: '/pages/my/my' })
    }, 300)
  }
})
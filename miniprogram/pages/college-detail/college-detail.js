const { request } = require('../../utils/request')
const { shouldRequireShareGate, markShareGateUnlocked, createShareGateToken } = require('../../utils/share-gate')

function enableShareMenus() {
  if (wx.showShareMenu) {
    wx.showShareMenu({ menus: ['shareAppMessage', 'shareTimeline'] })
  }
}

Page({
  data: {
    loading: false,
    detail: null,
    shareGateReady: false,
    shareUnlocked: false,
    shareUnlockPending: false,
    province: '黑龙江',
    subject: '历史',
    year: 2025
  },

  onLoad(query) {
    enableShareMenus()
    this.setData({
      province: '黑龙江',
      subject: decodeURIComponent(query.subject || '历史'),
      year: 2025,
      collegeID: Number(query.id || 0)
    })
    wx.setNavigationBarTitle({ title: '院校详情' })
    shouldRequireShareGate('collegeMajor', true, query.shareToken || '').then((required) => {
      if (required) {
        this.loadDetail()
        this.setData({ shareGateReady: true })
        wx.setNavigationBarTitle({ title: '分享后查看专业详情' })
        return
      }
      this.setData({ shareGateReady: true })
      this.unlockCollegeDetail()
    })
  },

  onShow() {
    enableShareMenus()
  },

  requestShareUnlock() {
    this.setData({ shareUnlockPending: true })
  },

  unlockCollegeDetail() {
    if (this.data.shareUnlocked) {
      return
    }
    this.setData({ shareUnlocked: true, shareUnlockPending: false, shareGateReady: true })
    if (!this.data.detail) {
      this.loadDetail()
    }
  },

  async loadDetail() {
    const { collegeID, province, subject, year } = this.data
    if (!collegeID) return
    this.setData({ loading: true })
    try {
      const detail = await request({
        url: `/api/colleges/${collegeID}`,
        method: 'POST',
        data: {
          province,
          subject,
          year
        }
      })
      this.setData({ detail })
      wx.setNavigationBarTitle({ title: detail.name || '院校详情' })
    } catch (err) {
      if (!err || !err.handledByModal) {
        wx.showToast({ title: (err && err.error) || '加载详情失败', icon: 'none' })
      }
    } finally {
      this.setData({ loading: false })
    }
  },

  openCharter() {
    const url = this.data.detail && this.data.detail.admissions_charter_url
    if (!url) {
      wx.showToast({ title: '暂无招生章程链接', icon: 'none' })
      return
    }
    wx.setClipboardData({ data: url })
  },

  onShareAppMessage() {
    const self = this
    const shareToken = createShareGateToken('collegeMajor')
    const subject = encodeURIComponent(this.data.subject || '历史')
    const year = encodeURIComponent(String(this.data.year || 2025))
    const collegeID = encodeURIComponent(String(this.data.collegeID || 0))
    return {
      title: '黑龙江高报助手：查看院校专业组、专业计划和录取信息',
      path: `/pages/college-detail/college-detail?id=${collegeID}&subject=${subject}&year=${year}&shareToken=${encodeURIComponent(shareToken)}`,
      success() {
        if (self.data.shareUnlockPending && !self.data.shareUnlocked) {
          markShareGateUnlocked('collegeMajor')
          self.unlockCollegeDetail()
        }
      },
      fail() {
        if (self.data.shareUnlockPending && !self.data.shareUnlocked) {
          self.setData({ shareUnlockPending: false })
        }
      }
    }
  },

  onShareTimeline() {
    const shareToken = createShareGateToken('collegeMajor')
    const subject = encodeURIComponent(this.data.subject || '历史')
    const year = encodeURIComponent(String(this.data.year || 2025))
    const collegeID = encodeURIComponent(String(this.data.collegeID || 0))
    return {
      title: '黑龙江高报助手：查看院校专业组、专业计划和录取信息',
      query: `id=${collegeID}&subject=${subject}&year=${year}&shareToken=${encodeURIComponent(shareToken)}`,
      imageUrl: ''
    }
  }
})

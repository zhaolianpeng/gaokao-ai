const { request } = require('../../utils/request')

Page({
  data: {
    loading: false,
    detail: null,
    province: '黑龙江',
    subject: '历史',
    year: 2025
  },

  onLoad(query) {
    this.setData({
      province: '黑龙江',
      subject: decodeURIComponent(query.subject || '历史'),
      year: 2025,
      collegeID: Number(query.id || 0)
    })
    this.loadDetail()
  },

  async loadDetail() {
    const { collegeID, province, subject, year } = this.data
    if (!collegeID) return
    this.setData({ loading: true })
    try {
      const detail = await request({
        url: `/api/colleges/${collegeID}?province=${encodeURIComponent(province)}&subject=${encodeURIComponent(subject)}&year=${year}`,
        method: 'GET'
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
  }
})

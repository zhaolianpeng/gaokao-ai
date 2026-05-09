const { getNetworkDiagnostics, clearNetworkDiagnostics } = require('../../utils/storage')

Page({
  data: {
    items: []
  },

  onShow() {
    this.setData({ items: getNetworkDiagnostics() })
  },

  copyDiagnostic(e) {
    const index = e.currentTarget.dataset.index
    const item = this.data.items[index]
    if (!item) {
      return
    }
    wx.setClipboardData({
      data: item.message,
      success: () => wx.showToast({ title: '已复制诊断信息', icon: 'success' })
    })
  },

  clearDiagnostics() {
    clearNetworkDiagnostics()
    this.setData({ items: [] })
    wx.showToast({ title: '已清空诊断记录', icon: 'success' })
  }
})
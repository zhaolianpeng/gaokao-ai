const { getMaterialById } = require('../../utils/materials')

Page({
  data: {
    material: null
  },

  onLoad(query) {
    const material = getMaterialById(query.id)
    wx.setNavigationBarTitle({
      title: material ? material.title : '资料详情'
    })
    this.setData({ material: material || null })
  },

  openMaterialsPage() {
    wx.navigateBack({
      delta: 1,
      fail: () => {
        wx.redirectTo({ url: '/pages/materials/materials' })
      }
    })
  }
})
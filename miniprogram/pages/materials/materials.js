const { getMaterials } = require('../../utils/materials')

Page({
  data: {
    materials: [],
    featured: [],
    loadError: ''
  },

  onLoad() {
    try {
      const materials = getMaterials()
      this.setData({
        loadError: '',
        materials,
        featured: materials.slice(0, 3)
      })
    } catch (error) {
      this.setData({
        loadError: '资料库加载失败，请重新进入页面',
        materials: [],
        featured: []
      })
    }
  },

  openMaterial(e) {
    const { id } = e.currentTarget.dataset
    if (!id) {
      return
    }
    wx.navigateTo({
      url: `/pages/material-detail/material-detail?id=${encodeURIComponent(String(id))}`
    })
  }
})
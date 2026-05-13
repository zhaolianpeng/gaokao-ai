const { getMaterials } = require('../../utils/materials')

const FEATURED_MATERIAL_IDS = [
  'simulate-2025-application-guide',
  'simulate-2025-plan-order-guide',
  'official-admission-charter-guide'
]

Page({
  data: {
    materials: [],
    featured: [],
    loadError: ''
  },

  onLoad() {
    try {
      const materials = getMaterials()
      const featured = FEATURED_MATERIAL_IDS.map((id) => materials.find((item) => item.id === id)).filter(Boolean)
      this.setData({
        loadError: '',
        materials,
        featured
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
  },

  openSimulationFlow() {
    wx.navigateTo({ url: '/pages/simulate-application/simulate-application' })
  }
})
const { request } = require('../../utils/request')

function normalizeLookupSubject(subject) {
  return subject === '物理' ? '物理类' : '历史类'
}

function buildYearHint(year, subject) {
  if (year === '2022' || year === '2023') {
    return `${year} 年黑龙江普通类仍沿用文科/理科发布口径，页面已自动映射到 ${subject} 查询。`
  }
  return `${year} 年黑龙江普通类按 ${subject} 实际批次线展示。`
}

Page({
  data: {
    loading: false,
    province: '黑龙江',
    subject: '历史',
    year: '2025',
    subjectOptions: ['历史', '物理'],
    yearOptions: ['2025', '2024', '2023', '2022'],
    yearHint: buildYearHint('2025', '历史'),
    items: []
  },

  onLoad(query) {
    this.setData({
      subject: decodeURIComponent(query.subject || '历史'),
      year: decodeURIComponent(query.year || '2025'),
      yearHint: buildYearHint(decodeURIComponent(query.year || '2025'), decodeURIComponent(query.subject || '历史'))
    })
    this.loadItems()
  },

  onSubjectChange(e) {
    const subject = this.data.subjectOptions[e.detail.value]
    this.setData({
      subject,
      yearHint: buildYearHint(this.data.year, subject)
    })
    this.loadItems()
  },

  onYearChange(e) {
    const year = this.data.yearOptions[e.detail.value]
    this.setData({
      year,
      yearHint: buildYearHint(year, this.data.subject)
    })
    this.loadItems()
  },

  async loadItems() {
    const { province, subject, year } = this.data
    this.setData({ loading: true })
    try {
      const data = await request({
        url: `/api/province-lines?province=${encodeURIComponent(province)}&subject=${encodeURIComponent(normalizeLookupSubject(subject))}&year=${year}`,
        method: 'GET'
      })
      this.setData({ items: data.items || [] })
    } catch (err) {
      if (!err || !err.handledByModal) {
        wx.showToast({ title: (err && err.error) || '加载批次线失败', icon: 'none' })
      }
      this.setData({ items: [] })
    } finally {
      this.setData({ loading: false })
    }
  }
})

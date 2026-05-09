const { request } = require('../../utils/request')
const { consumePendingExploreSubject, consumePendingExploreFilters } = require('../../utils/storage')

function resolveSubject(querySubject, currentSubject) {
  return querySubject || consumePendingExploreSubject() || currentSubject || '历史'
}

Page({
  data: {
    loading: false,
    loadingMore: false,
    province: '黑龙江',
    subject: '历史',
    year: 2025,
    subjectOptions: ['历史', '物理'],
    keyword: '',
    sortMode: 'tier',
    sortOptions: [
      { value: 'tier', label: '按学校层次排序' },
      { value: 'admission', label: '按录取位次排序' }
    ],
    items: [],
    hasItems: false,
    showEmptyState: false,
    page: 1,
    pageSize: 20,
    hasMore: false,
    paginationReady: true,
    filterSummary: '',
    pageSummary: '',
    suggestedKeywords: ['师范', '医学', '计算机', '财经', '哈尔滨'],
    resultTip: ''
  },

  onLoad(query) {
    const pendingFilters = consumePendingExploreFilters()
    this.setData({
      subject: decodeURIComponent(resolveSubject((pendingFilters && pendingFilters.subject) || query.subject, this.data.subject)),
      province: '黑龙江',
      year: 2025,
      keyword: (pendingFilters && pendingFilters.keyword) || decodeURIComponent(query.keyword || '')
    })
    this.loadColleges({ reset: true })
  },

  onShow() {
    const pendingFilters = consumePendingExploreFilters()
    const nextSubject = consumePendingExploreSubject()
    if ((pendingFilters && (pendingFilters.subject !== this.data.subject || pendingFilters.keyword !== this.data.keyword)) || (nextSubject && nextSubject !== this.data.subject)) {
      this.setData({
        subject: (pendingFilters && pendingFilters.subject) || nextSubject || this.data.subject,
        keyword: pendingFilters && typeof pendingFilters.keyword === 'string' ? pendingFilters.keyword : this.data.keyword
      })
      this.loadColleges({ reset: true })
    }
  },

  onSubjectChange(e) {
    const value = this.data.subjectOptions[e.detail.value]
    this.setData({ subject: value })
    this.loadColleges({ reset: true })
  },

  onKeywordInput(e) {
    this.setData({ keyword: e.detail.value })
  },

  onKeywordConfirm() {
    this.loadColleges({ reset: true })
  },

  onQuickKeyword(e) {
    const keyword = e.currentTarget.dataset.keyword || ''
    this.setData({ keyword })
    this.loadColleges({ reset: true })
  },

  buildFilterSummary() {
    const { province, subject, year, keyword, pageSize, sortMode, sortOptions } = this.data
    const currentSort = (sortOptions || []).find((item) => item.value === sortMode)
    const sortLabel = currentSort ? currentSort.label : '按学校层次排序'
    return keyword
      ? `${province} · ${year} · ${subject} · 关键词“${keyword}” · ${sortLabel} · 每页 ${pageSize} 所`
      : `${province} · ${year} · ${subject} · 全部院校 · ${sortLabel} · 每页 ${pageSize} 所`
  },

  async loadColleges({ reset = false } = {}) {
    const nextPage = reset ? 1 : this.data.page + 1
    const loadingKey = reset ? 'loading' : 'loadingMore'
    if (!reset && (!this.data.hasMore || this.data.loadingMore || this.data.loading || !this.data.paginationReady)) {
      return
    }
    this.setData({ [loadingKey]: true, ...(reset ? { resultTip: '' } : {}) })
    try {
      const { province, subject, year, keyword, pageSize, items, sortMode } = this.data
      const data = await request({
        url: `/api/colleges?province=${encodeURIComponent(province)}&subject=${encodeURIComponent(subject)}&year=${year}&keyword=${encodeURIComponent(keyword)}&sortMode=${encodeURIComponent(sortMode)}&page=${nextPage}&limit=${pageSize}`,
        method: 'GET'
      })
      const nextItems = data.items || []
      const paginationReady = typeof data.hasMore === 'boolean' && typeof data.page === 'number' && typeof data.limit === 'number'
      const mergedItems = reset ? nextItems : items.concat(nextItems)
      const hasItems = mergedItems.length > 0
      const pageSummary = !mergedItems.length
        ? ''
        : paginationReady
          ? `已加载第 ${nextPage} 页${data.hasMore ? '，继续下滑加载更多' : '，当前结果已全部展示'}`
          : '当前后端还未上线分页接口，暂时仅展示这 20 所院校'
      this.setData({
        items: mergedItems,
        hasItems,
        showEmptyState: !hasItems,
        page: nextPage,
        hasMore: paginationReady ? !!data.hasMore : false,
        paginationReady,
        filterSummary: this.buildFilterSummary(),
        pageSummary,
        resultTip: paginationReady
          ? (keyword ? `关键词“${keyword}”当前已加载 ${mergedItems.length} 所院校` : `当前已加载 ${mergedItems.length} 所院校`)
          : (keyword ? `关键词“${keyword}”当前仅返回 20 所院校，等待后端分页上线` : '当前后端仅返回 20 所院校，等待后端分页上线')
      })
    } catch (err) {
      if (reset) {
        this.setData({ hasItems: this.data.items.length > 0, showEmptyState: this.data.items.length === 0 })
      }
      if (!err || !err.handledByModal) {
        wx.showToast({ title: (err && err.error) || '加载院校失败', icon: 'none' })
      }
    } finally {
      this.setData({ [loadingKey]: false })
    }
  },

  onSearch() {
    this.loadColleges({ reset: true })
  },

  onSortModeChange(e) {
  const value = e.currentTarget.dataset.value || 'tier'
  if (value === this.data.sortMode) {
    return
  }
  this.setData({ sortMode: value })
  this.loadColleges({ reset: true })
  },

  onReachBottom() {
    this.loadColleges()
  },

  openCollegeDetail(e) {
    const id = e.currentTarget.dataset.id
    if (!id) return
    const { subject } = this.data
    wx.navigateTo({
      url: `/pages/college-detail/college-detail?id=${id}&subject=${encodeURIComponent(subject)}`
    })
  }
})

const { getRecommendHistory, savePendingRecommendPayload } = require('../../utils/storage')

function formatTime(timestamp) {
  const date = new Date(timestamp)
  const year = date.getFullYear()
  const month = `${date.getMonth() + 1}`.padStart(2, '0')
  const day = `${date.getDate()}`.padStart(2, '0')
  const hour = `${date.getHours()}`.padStart(2, '0')
  const minute = `${date.getMinutes()}`.padStart(2, '0')
  return `${year}-${month}-${day} ${hour}:${minute}`
}

function buildArchiveText(student) {
  const safeStudent = student || {}
  const parts = [safeStudent.schoolName, safeStudent.schoolYear, safeStudent.className].filter(Boolean)
  return parts.length ? parts.join(' / ') : ''
}

Page({
  data: {
    history: []
  },

  onShow() {
    const history = getRecommendHistory().map((item) => ({
      ...item,
      timeText: formatTime(item.createdAt),
      archiveText: buildArchiveText(item.student),
      fromRecommendText: item && item.student && item.student.fromRecommend ? '推荐来源' : ''
    }))
    this.setData({ history })
  },

  openRecord(e) {
    const record = this.data.history[e.currentTarget.dataset.index]
    if (!record) {
      return
    }
    const pendingPayload = savePendingRecommendPayload(record.student, record.result)
    wx.navigateTo({
      url: '/pages/recommend/recommend',
      success(res) {
        if (res && res.eventChannel) {
          res.eventChannel.emit('acceptRecommendPayload', pendingPayload)
        }
      }
    })
  }
})
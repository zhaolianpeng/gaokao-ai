const { request } = require('../../utils/request')

function normalizeLookupSubject(subject) {
  return subject === '物理' ? '物理类' : '历史类'
}

function isLegacyMappedYear(year) {
  return year === '2022' || year === '2023'
}

function buildYearHint(year, subject) {
  const legacyMapped = isLegacyMappedYear(year)
  if (year === '2023') {
    return {
      mappingTone: 'info',
      mappingText: `${year} 年普通类按老高考文科/理科口径发布，页面会自动映射到 ${subject} 查询。`,
      reliabilityTone: 'positive',
      reliabilityText: `${year} 年普通类一分一段已补齐，命中同分值时返回精确位次。`
    }
  }
  if (year === '2022') {
    return {
      mappingTone: 'info',
      mappingText: `${year} 年按老高考文科/理科口径发布，页面会自动映射到 ${subject} 查询。`,
      reliabilityTone: 'positive',
      reliabilityText: `${year} 年当前已支持精确分值命中，未命中时才回退最近分段。`
    }
  }
  return {
    mappingTone: 'info',
    mappingText: `${year} 年按黑龙江 ${subject} 新高考口径直接查询。`,
    reliabilityTone: legacyMapped ? 'positive' : 'positive',
    reliabilityText: `${year} 年命中同分值时返回精确位次，未命中时回退最近分段。`
  }
}

function buildResultView(result, year) {
  if (!result || !result.available) {
    return null
  }
  const lookupType = result.lookup_type || 'score'
  const diff = Number(result.diff || 0)
  const legacyMapped = isLegacyMappedYear(year)
  if (result.exact) {
    return {
      ...result,
      lookupType,
      status: 'exact',
      label: '精确命中',
      message: lookupType === 'rank'
        ? (legacyMapped
          ? `命中精确位次，当前分数人数 ${result.count}。`
          : `命中精确位次，当前分数人数 ${result.count}`)
        : (legacyMapped
          ? `命中精确分值，当前分数人数 ${result.count}。`
          : `命中精确分值，当前分数人数 ${result.count}`)
    }
  }
  if (diff <= 5) {
    return {
      ...result,
      lookupType,
      status: 'near',
      label: '近似命中',
      message: lookupType === 'rank'
        ? (legacyMapped
          ? `未命中精确位次，已回退到最近的 ${result.rank} 名分段。当前年份按老高考文科/理科口径自动映射。`
          : `未命中精确位次，已回退到最近的 ${result.rank} 名分段。`)
        : (legacyMapped
          ? `未命中精确分值，已回退到最近的 ${result.matched_score} 分分段。当前年份按老高考文科/理科口径自动映射。`
          : `未命中精确分值，已回退到最近的 ${result.matched_score} 分分段。`)
    }
  }
  return {
    ...result,
    lookupType,
    status: 'approx',
    label: '参考结果',
    message: lookupType === 'rank'
      ? (legacyMapped
        ? `当前仅命中 ${result.rank} 名分段，和输入排名相差 ${diff} 名，结果仅供参考。当前年份按老高考文科/理科口径自动映射。`
        : `当前仅命中 ${result.rank} 名分段，和输入排名相差 ${diff} 名，结果仅供参考。`)
      : (legacyMapped
        ? `当前仅命中 ${result.matched_score} 分分段，和输入分数相差 ${diff} 分，结果仅供参考。当前年份按老高考文科/理科口径自动映射。`
        : `当前仅命中 ${result.matched_score} 分分段，和输入分数相差 ${diff} 分，结果仅供参考。`)
  }
}

function exceedsTolerance(inputValue, suggestedValue) {
  const current = Number(inputValue || 0)
  const suggested = Number(suggestedValue || 0)
  if (current <= 0 || suggested <= 0) {
    return false
  }
  const diff = Math.abs(current - suggested)
  return diff / suggested > 0.1
}

function buildToleranceToast(fieldLabel, suggestedValue) {
  return `${fieldLabel}调整已超过10%，当前建议值是 ${suggestedValue}`
}

Page({
  data: {
    loading: false,
    province: '黑龙江',
    subject: '历史',
    year: '2025',
    score: '',
    rank: '',
    result: null,
    batchLines: [],
    suggestedScore: 0,
    suggestedRank: 0,
    lastEditedField: 'score',
    yearHint: buildYearHint('2025', '历史'),
    subjectOptions: ['历史', '物理'],
    yearOptions: ['2025', '2024', '2023', '2022']
  },

  onLoad(query) {
    this.setData({
      subject: decodeURIComponent(query.subject || '历史'),
      year: decodeURIComponent(query.year || '2025'),
      score: decodeURIComponent(query.score || ''),
      rank: decodeURIComponent(query.rank || ''),
      yearHint: buildYearHint(decodeURIComponent(query.year || '2025'), decodeURIComponent(query.subject || '历史'))
    })
    this.loadContext()
  },

  onSubjectChange(e) {
    const subject = this.data.subjectOptions[e.detail.value]
    this.setData({
      subject,
      yearHint: buildYearHint(this.data.year, subject)
    })
    this.loadContext()
  },

  onYearChange(e) {
    const year = this.data.yearOptions[e.detail.value]
    this.setData({
      year,
      yearHint: buildYearHint(year, this.data.subject)
    })
    this.loadContext()
  },

  onScoreInput(e) {
    this.setData({ score: e.detail.value, lastEditedField: 'score' })
    this.scheduleLookup('score')
  },

  onScoreBlur(e) {
    const nextScore = e.detail && e.detail.value ? e.detail.value : this.data.score
    if (exceedsTolerance(nextScore, this.data.suggestedScore)) {
      wx.showToast({ title: buildToleranceToast('分数', this.data.suggestedScore), icon: 'none' })
    }
  },

  onSearch() {
    this.lookupByField(this.data.lastEditedField || 'score')
  },

  onRankInput(e) {
    this.setData({ rank: e.detail.value, lastEditedField: 'rank' })
    this.scheduleLookup('rank')
  },

  onRankBlur(e) {
    const nextRank = e.detail && e.detail.value ? e.detail.value : this.data.rank
    if (exceedsTolerance(nextRank, this.data.suggestedRank)) {
      wx.showToast({ title: buildToleranceToast('排名', this.data.suggestedRank), icon: 'none' })
    }
  },

  scheduleLookup(field) {
    if (this.lookupTimer) {
      clearTimeout(this.lookupTimer)
    }
    this.lookupTimer = setTimeout(() => this.lookupByField(field), 250)
  },

  async lookupByField(field) {
    if (field === 'rank') {
      await this.lookupRankScore()
      return
    }
    await this.lookupScoreRank()
  },

  async loadContext() {
    await this.loadBatchLines()
    const lookupField = this.data.score ? 'score' : (this.data.rank ? 'rank' : 'score')
    await this.lookupByField(lookupField)
  },

  async loadBatchLines() {
    const { province, subject, year } = this.data
    try {
      const data = await request({
        url: '/api/province-lines',
        method: 'POST',
        data: {
          province,
          subject: normalizeLookupSubject(subject),
          year: Number(year)
        }
      })
      this.setData({ batchLines: data.items || [] })
    } catch (err) {
      this.setData({ batchLines: [] })
    }
  },

  async lookupScoreRank() {
    const { province, subject, year, score } = this.data
    if (!score || Number(score) <= 0) {
      this.setData({ result: null, suggestedRank: 0 })
      return
    }
    this.setData({ loading: true })
    try {
      const result = await request({
        url: '/api/score-rank',
        method: 'POST',
        data: {
          province,
          subject: normalizeLookupSubject(subject),
          year: Number(year),
          score: Number(score)
        }
      })
      const nextRank = result && result.available && Number(result.rank || 0) > 0 ? String(result.rank) : this.data.rank
      this.setData({
        rank: nextRank,
        suggestedRank: Number(result && result.rank) || 0,
        result: buildResultView(result, year)
      })
    } catch (err) {
      if (!err || !err.handledByModal) {
        wx.showToast({ title: (err && err.error) || '查询位次失败', icon: 'none' })
      }
      this.setData({ result: null, suggestedRank: 0 })
    } finally {
      this.setData({ loading: false })
    }
  },

  async lookupRankScore() {
    const { province, subject, year, rank } = this.data
    if (!rank || Number(rank) <= 0) {
      this.setData({ result: null, suggestedScore: 0 })
      return
    }
    this.setData({ loading: true })
    try {
      const result = await request({
        url: '/api/rank-score',
        method: 'POST',
        data: {
          province,
          subject: normalizeLookupSubject(subject),
          year: Number(year),
          rank: Number(rank)
        }
      })
      const nextScore = result && result.available && Number(result.matched_score || 0) > 0 ? String(result.matched_score) : this.data.score
      this.setData({
        score: nextScore,
        suggestedScore: Number(result && result.matched_score) || 0,
        result: buildResultView(result, year)
      })
    } catch (err) {
      if (!err || !err.handledByModal) {
        wx.showToast({ title: (err && err.error) || '反查分数失败', icon: 'none' })
      }
      this.setData({ result: null, suggestedScore: 0 })
    } finally {
      this.setData({ loading: false })
    }
  }
})

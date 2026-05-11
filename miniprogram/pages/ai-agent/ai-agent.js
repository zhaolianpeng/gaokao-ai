const { request } = require('../../utils/request')

const DEMAND_TEMPLATES = [
  { id: 'local', label: '留省内', prompt: '我想优先留在黑龙江省内读书，尽量不要去外省。', keyword: '黑龙江' },
  { id: 'harbin', label: '优先哈尔滨', prompt: '我想优先去哈尔滨读书，城市因素优先级较高。', keyword: '哈尔滨' },
  { id: 'public', label: '优先公办', prompt: '我更倾向公办院校，民办和高收费项目优先级低。', keyword: '公办' },
  { id: '211', label: '冲 211', prompt: '如果有机会，我希望冲一冲 211 或更高层次院校。', keyword: '211' },
  { id: 'computer', label: '偏计算机', prompt: '我的专业偏好集中在计算机、软件、电子信息方向。', keyword: '计算机' },
  { id: 'adjust', label: '接受调剂', prompt: '如果整体层次更合适，我可以接受组内调剂。', keyword: '' }
]

const AGENT_WORKFLOW = [
  {
    key: 'submit',
    title: '先提交需求',
    eta: '10-20 秒内入队',
    desc: '这里主要收集你的城市、学校层次、专业偏好和调剂接受度。'
  },
  {
    key: 'task',
    title: '异步生成分析',
    eta: '通常 60-120 秒',
    desc: 'AI 智能体会单独跑任务，不占住首页查询和推荐链路。'
  },
  {
    key: 'report',
    title: '输出可执行结论',
    eta: '完成后可直接查看',
    desc: '最终会给出报考策略、解释文本和可继续查院校的建议入口。'
  }
]

const AGENT_DELIVERABLES = [
  '先讲结论，再解释为什么这么排。',
  '重点说明学校层次、专业命中和调剂风险。',
  '输出给家长也能直接看懂的报考建议。'
]

function unique(values) {
  return Array.from(new Set((values || []).filter(Boolean)))
}

function buildExploreSuggestions(form, selectedTemplates) {
  const suggestions = []
  const pushSuggestion = (title, keyword) => {
    if (!keyword) {
      return
    }
    suggestions.push({
      id: `${title}-${keyword}`,
      title,
      keyword,
      subject: form.subject || '历史'
    })
  }

  if (form.targetMajor) {
    pushSuggestion('按意向专业筛选', form.targetMajor)
  }

  if (form.notes && /哈尔滨/.test(form.notes)) {
    pushSuggestion('查看哈尔滨院校', '哈尔滨')
  }

  if (form.demand && /哈尔滨/.test(form.demand)) {
    pushSuggestion('优先哈尔滨', '哈尔滨')
  }

  if (form.demand && /计算机|软件|电子信息/.test(form.demand)) {
    pushSuggestion('查看计算机方向', '计算机')
  }

  selectedTemplates.forEach((item) => {
    if (item.keyword) {
      pushSuggestion(item.label, item.keyword)
    }
  })

  return unique(suggestions.map((item) => JSON.stringify(item))).slice(0, 5).map((item) => JSON.parse(item))
}

function getTaskStatusText(status) {
  if (status === 'succeeded') {
    return '智能体推荐结果已生成，可以点击下方按钮查看。'
  }
  if (status === 'failed') {
    return '智能体生成失败，请重新提交。'
  }
  if (status === 'processing') {
    return '智能体正在分析需求，页面会自动刷新结果状态。'
  }
  if (status === 'pending') {
    return '请求已入库保存，等待智能体开始处理。'
  }
  return ''
}

Page({
  data: {
    loading: false,
    polling: false,
    taskId: '',
    taskStatus: '',
    taskStatusText: '',
    taskError: '',
    taskReady: false,
    taskResult: null,
    subjectOptions: ['历史', '物理'],
    templates: DEMAND_TEMPLATES,
    workflow: AGENT_WORKFLOW,
    deliverables: AGENT_DELIVERABLES,
    selectedTemplateIds: [],
    form: {
      province: '黑龙江',
      subject: '历史',
      analysisYear: '2025',
      score: '',
      rank: '',
      targetMajor: '',
      notes: '',
      demand: '我想优先去哈尔滨，尽量公办，专业偏计算机或电子信息，能接受组内调剂，请给我一个清晰的报考策略。'
    }
  },

  onLoad(query) {
    const subject = decodeURIComponent(query.subject || '历史')
    const analysisYear = decodeURIComponent(query.analysisYear || '2025')
    this.setData({
      form: {
        ...this.data.form,
        subject,
        analysisYear,
        score: decodeURIComponent(query.score || ''),
        rank: decodeURIComponent(query.rank || ''),
        targetMajor: decodeURIComponent(query.targetMajor || ''),
        notes: decodeURIComponent(query.notes || '')
      }
    })
  },

  onShow() {
    if (this.data.taskId && !this.data.taskReady && this.data.taskStatus !== 'failed') {
      this.startPolling(this.data.taskId)
    }
  },

  onHide() {
    this.stopPolling()
  },

  onUnload() {
    this.stopPolling()
  },

  onSubjectChange(e) {
    const subject = this.data.subjectOptions[e.detail.value]
    this.setData({ 'form.subject': subject })
  },

  onInput(e) {
    const field = e.currentTarget.dataset.field
    this.setData({ [`form.${field}`]: e.detail.value })
  },

  applyTemplate(e) {
    const id = e.currentTarget.dataset.id
    const template = this.data.templates.find((item) => item.id === id)
    if (!template) {
      return
    }
    const selectedTemplateIds = this.data.selectedTemplateIds.indexOf(id) >= 0
      ? this.data.selectedTemplateIds
      : this.data.selectedTemplateIds.concat(id)
    const currentDemand = String(this.data.form.demand || '').trim()
    const nextDemand = currentDemand.indexOf(template.prompt) >= 0
      ? currentDemand
      : (currentDemand ? `${currentDemand}\n${template.prompt}` : template.prompt)
    this.setData({
      selectedTemplateIds,
      'form.demand': nextDemand
    })
  },

  validate() {
    if (!this.data.form.demand.trim()) {
      return '请先输入你的报考需求'
    }
    return ''
  },

  getSelectedTemplates() {
    return this.data.templates.filter((item) => this.data.selectedTemplateIds.indexOf(item.id) >= 0)
  },

  buildPayload() {
    return {
      student: {
        province: this.data.form.province,
        subject: this.data.form.subject,
        analysisYear: this.data.form.analysisYear,
        score: Number(this.data.form.score || 0),
        rank: Number(this.data.form.rank || 0),
        targetMajor: this.data.form.targetMajor,
        notes: this.data.form.notes
      },
      demand: this.data.form.demand,
      templates: this.getSelectedTemplates().map((item) => item.label)
    }
  },

  stopPolling() {
    if (this.pollTimer) {
      clearInterval(this.pollTimer)
      this.pollTimer = null
    }
  },

  startPolling(taskId) {
    if (!taskId) {
      return
    }
    this.stopPolling()
    this.pollAgentTask(taskId)
    this.pollTimer = setInterval(() => {
      this.pollAgentTask(taskId)
    }, 1500)
  },

  async pollAgentTask(taskId) {
    if (!taskId || this.pollingRequest) {
      return
    }
    this.pollingRequest = true
    try {
      const data = await request({
        url: '/api/agent-recommend/task',
        method: 'POST',
        data: { taskId },
        timeout: 20000
      })

      const taskStatus = data.status || 'pending'
      const taskReady = !!data.ready
      const taskFailed = !!data.failed
      this.setData({
        polling: !taskReady && !taskFailed,
        taskStatus,
        taskStatusText: getTaskStatusText(taskStatus),
        taskError: data.errorMessage || '',
        taskReady,
        taskResult: data
      })

      if (taskReady || taskFailed) {
        this.stopPolling()
      }
    } catch (err) {
      this.stopPolling()
      this.setData({
        polling: false,
        taskStatus: 'failed',
        taskStatusText: getTaskStatusText('failed'),
        taskError: (err && err.error) || '轮询任务状态失败'
      })
      if (!err || !err.handledByModal) {
        wx.showToast({ title: (err && err.error) || '轮询失败', icon: 'none' })
      }
    } finally {
      this.pollingRequest = false
    }
  },

  viewResult() {
    const result = this.data.taskResult
    if (!result || !this.data.taskReady) {
      return
    }

    const payload = this.buildPayload()
    const suggestions = (result.suggestions && result.suggestions.length)
      ? result.suggestions
      : buildExploreSuggestions(this.data.form, this.getSelectedTemplates())

    wx.navigateTo({
      url: '/pages/report/report?title=' + encodeURIComponent(result.title || 'AI 智能体报考建议') + '&report=' + encodeURIComponent(result.report || '') + '&student=' + encodeURIComponent(JSON.stringify(result.student || payload.student)) + '&suggestions=' + encodeURIComponent(JSON.stringify(suggestions))
    })
  },

  async onSubmit() {
    const message = this.validate()
    if (message) {
      wx.showToast({ title: message, icon: 'none' })
      return
    }

    this.setData({ loading: true })
    try {
      const payload = this.buildPayload()
      this.stopPolling()
      const data = await request({
        url: '/api/agent-recommend',
        method: 'POST',
        data: payload,
        timeout: 20000
      })

      this.setData({
        polling: true,
        taskId: data.taskId || '',
        taskStatus: data.status || 'pending',
        taskStatusText: getTaskStatusText(data.status || 'pending'),
        taskError: '',
        taskReady: false,
        taskResult: null
      })
      wx.showToast({ title: '请求已提交', icon: 'success' })
      this.startPolling(data.taskId)
    } catch (err) {
      if (!err || !err.handledByModal) {
        wx.showToast({ title: (err && err.error) || '生成失败', icon: 'none' })
      }
    } finally {
      this.setData({ loading: false })
    }
  }
})
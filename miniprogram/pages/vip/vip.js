const { request } = require('../../utils/request')
const { getAuthUser } = require('../../utils/storage')

const PRICE_BANDS = [
  { label: '免费层', price: '￥0', desc: '先把查询、一分一段、冲稳保推荐和家长摘要用起来。' },
  { label: '轻量体验', price: '低于 ￥100', desc: '适合只在关键节点确认 1-2 次深度解释。' },
  { label: '主流会员', price: '￥98-198', desc: '竞品最常见的标准决策价带，适合集中填报。' },
  { label: '全周期', price: '￥199-399', desc: '适合完整志愿期反复比较、复盘和长期保存。' }
]

const PRODUCTS = [
  {
    id: 'vip_single',
    title: '单次深度报告',
    desc: '适合临门一脚确认 1 次推荐解释，不想一次买长期会员。',
    badge: '入门',
    anchorText: '对标轻量体验价带',
    suitedFor: '适合只想看 1 次深度解释的考生',
    highlights: ['1 次深度 AI 报告', '保学校/保专业/保城市三视角复盘', '适合考前关键节点确认'],
    amountFen: 1
  },
  {
    id: 'vip_day',
    title: '冲刺日卡',
    desc: '按天集中体验，适合和家长、老师在一天内快速比多套方案。',
    badge: '试用',
    anchorText: '对标短期冲刺价带',
    suitedFor: '适合 1 天内高强度比较多套志愿方案',
    highlights: ['短期集中看报告', '多轮沟通时更省时间', '适合出分后快速决策'],
    amountFen: 1
  },
  {
    id: 'vip_month',
    title: '填报月卡',
    desc: '对应竞品主流会员价带，适合出分到正式填报这一整个阶段使用。',
    badge: '主推',
    anchorText: '对标主流会员 ￥98-198',
    suitedFor: '适合需要集中填报、反复比学校和专业的家庭',
    highlights: ['更适合高频深度 AI 分析', '多套方案长期保存', '和家长老师反复讨论更顺手'],
    amountFen: 1
  },
  {
    id: 'vip_season',
    title: '全程季卡',
    desc: '覆盖完整志愿准备周期，适合从预估到正式填报持续复盘。',
    badge: '全周期',
    anchorText: '对标重度会员 ￥199-399',
    suitedFor: '适合完整志愿周期、多次改方案的人群',
    highlights: ['适合长期保存多套版本', '适合多轮深度 AI 解释', '覆盖预估、比对、定稿全流程'],
    amountFen: 1
  }
]

const ACCESS_MATRIX = [
  { feature: '院校/专业组/批次线/位次查询', free: '全部可用', lite: '可用', core: '可用', full: '可用' },
  { feature: '冲稳保推荐与家长摘要', free: '可用', lite: '可用', core: '可用', full: '可用' },
  { feature: '深度 AI 报告', free: '少量体验', lite: '单次确认', core: '集中使用', full: '全程高频' },
  { feature: '多套方案长期保存', free: '少量', lite: '1-2 套', core: '多套对比', full: '长期留存' },
  { feature: '反复和家长老师复盘', free: '基础', lite: '可做', core: '更省时间', full: '最适合' },
  { feature: '适合场景', free: '先做基础判断', lite: '临门一脚', core: '集中填报', full: '全周期决策' }
]

function buildOrderId(productId) {
  return `${productId}_${Date.now()}`
}

function isServerUser(user) {
  const userId = user && user.id ? String(user.id) : ''
  return !!user && user.storageMode === 'server' && /^\d+$/.test(userId)
}

function requestWechatPayment(payParams) {
  return new Promise((resolve, reject) => {
    wx.requestPayment({
      ...payParams,
      success: resolve,
      fail: reject
    })
  })
}

function normalizePaymentParams(payload) {
  const payment = payload && payload.payment ? payload.payment : payload
  if (!payment) {
    return null
  }

  const normalized = {
    timeStamp: payment.timeStamp ? String(payment.timeStamp) : '',
    nonceStr: payment.nonceStr || '',
    package: payment.package || '',
    signType: payment.signType || 'RSA',
    paySign: payment.paySign || ''
  }

  if (!normalized.timeStamp || !normalized.nonceStr || !normalized.package || !normalized.paySign) {
    return null
  }

  return normalized
}

function formatPrice(amountFen) {
  return `￥${(Number(amountFen || 0) / 100).toFixed(2)}`
}

Page({
  data: {
    user: null,
    priceBands: PRICE_BANDS,
    products: PRODUCTS.map((product) => ({
      ...product,
      priceText: `当前内测价 ${formatPrice(product.amountFen)}`
    })),
    accessMatrix: ACCESS_MATRIX,
    loadingProductId: '',
    canPay: false,
    loginHint: ''
  },

  onShow() {
    const user = getAuthUser()
    this.setData({
      user,
      canPay: isServerUser(user),
      loginHint: this.buildLoginHint(user)
    })
  },

  buildLoginHint(user) {
    if (!user) {
      return '未检测到登录信息，请先完成手机号快捷登录。'
    }
    if (user.storageMode !== 'server') {
      return '当前是体验模式账号，无法发起服务端支付。'
    }
    if (!user.id) {
      return '当前账号缺少服务端用户ID，请重新登录。'
    }
    return '已检测到服务端登录账号，可以直接购买。'
  },

  goLogin() {
    wx.navigateTo({ url: '/pages/login/login' })
  },

  async purchase(e) {
    const productId = e.currentTarget.dataset.productId
    const user = getAuthUser()
    if (!isServerUser(user)) {
      wx.showToast({ title: this.buildLoginHint(user), icon: 'none' })
      return
    }

    const orderId = buildOrderId(productId)
    this.setData({ loadingProductId: productId })
    try {
      const payResult = await request({
        url: '/api/vip/pay',
        method: 'POST',
        data: {
          productId,
          orderId,
          userId: String(user.id)
        }
      })

      const payParams = normalizePaymentParams(payResult)
      if (!payParams) {
        throw new Error('支付参数不完整')
      }

      await requestWechatPayment(payParams)

      await request({
        url: '/api/vip/pay/confirm',
        method: 'POST',
        data: {
          productId,
          orderId,
          userId: String(user.id)
        }
      })

      wx.showToast({ title: '支付成功', icon: 'success' })
    } catch (err) {
      const message = (err && err.errMsg) || (err && err.error) || (err && err.message) || '支付未完成'
      if (String(message).indexOf('cancel') !== -1) {
        wx.showToast({ title: '已取消支付', icon: 'none' })
      } else {
        wx.showToast({ title: message, icon: 'none' })
      }
    } finally {
      this.setData({ loadingProductId: '' })
    }
  }
})
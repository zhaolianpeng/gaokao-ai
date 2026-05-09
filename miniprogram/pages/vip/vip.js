const { request } = require('../../utils/request')
const { getAuthUser } = require('../../utils/storage')

const PRODUCTS = [
  { id: 'vip_single', title: '次卡', desc: '适合临门一脚看 1 次深度报告', badge: '低门槛', amountFen: 1 },
  { id: 'vip_day', title: '天卡', desc: '按天体验 VIP 服务，适合集中比方案', badge: '试用', amountFen: 1 },
  { id: 'vip_month', title: '月卡', desc: '适合集中填报阶段使用', badge: '推荐', amountFen: 1 },
  { id: 'vip_season', title: '季卡', desc: '覆盖完整志愿准备周期', badge: '长期规划', amountFen: 1 }
]

const ACCESS_MATRIX = [
  { feature: '冲稳保推荐结果', free: '可用', vip: '可用' },
  { feature: '家长沟通摘要', free: '可用', vip: '可用' },
  { feature: '正式志愿表基础整理', free: '可用', vip: '可用' },
  { feature: '深度 AI 报告', free: '基础体验', vip: '更适合高频使用' },
  { feature: '多套方案长期保存', free: '少量体验', vip: '更适合持续对比' },
  { feature: '反复复盘与讨论', free: '可体验', vip: '更省时间' }
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
    products: PRODUCTS.map((product) => ({
      ...product,
      priceText: formatPrice(product.amountFen)
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
      return '未检测到登录信息，请先完成微信手机号登录。'
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
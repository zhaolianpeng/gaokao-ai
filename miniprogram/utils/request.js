const { saveNetworkDiagnostic } = require('./storage')
const CLOUD_FUNCTION_NAME = 'gaokaoApi'
const RETRY_DELAYS = [400, 1200]

function getCloudEnvText() {
  try {
    const appBaseInfo = typeof wx.getAppBaseInfo === 'function' ? wx.getAppBaseInfo() : null
    const env = appBaseInfo && appBaseInfo.host && appBaseInfo.host.env ? appBaseInfo.host.env : 'unknown'
    if (wx.cloud && typeof wx.cloud.DYNAMIC_CURRENT_ENV !== 'undefined') {
      return `cloud=${env}`
    }
    return `cloud=${env}`
  } catch (err) {
    return 'cloud=unknown'
  }
}

function getRuntimeDebugInfo(route) {
  try {
    const accountInfo = typeof wx.getAccountInfoSync === 'function' ? wx.getAccountInfoSync() : null
    const appId = accountInfo && accountInfo.miniProgram ? accountInfo.miniProgram.appId : 'unknown'
    const envVersion = accountInfo && accountInfo.miniProgram ? accountInfo.miniProgram.envVersion : 'unknown'
    return `appid=${appId}, env=${envVersion}, ${getCloudEnvText()}, route=${route}`
  } catch (err) {
    return `route=${route}`
  }
}

function getDeviceDebugInfo() {
  try {
    if (typeof wx.getDeviceInfo !== 'function') {
      return 'device=unknown'
    }
    const info = wx.getDeviceInfo()
    return `platform=${info.platform || 'unknown'}, system=${info.system || 'unknown'}, model=${info.model || 'unknown'}`
  } catch (err) {
    return 'device=unknown'
  }
}

function toErrorString(payload) {
  if (!payload) {
    return ''
  }
  if (typeof payload === 'string') {
    return payload
  }
  try {
    return JSON.stringify(payload)
  } catch (err) {
    return String(payload)
  }
}

function buildDiagnosticMessage({ title, method, route, message, extra }) {
  return [
    title,
    `时间：${new Date().toLocaleString('zh-CN')}`,
    `请求：${method} ${route}`,
    `错误：${message}`,
    `运行：${getRuntimeDebugInfo(route)}`,
    `设备：${getDeviceDebugInfo()}`,
    extra ? `补充：${extra}` : '',
    '完整记录已保存到“我的-网络诊断”。'
  ].filter(Boolean).join('\n')
}

function wait(delay) {
  return new Promise((resolve) => setTimeout(resolve, delay))
}

function parseRoute(url) {
  const [path, queryString = ''] = String(url || '').split('?')
  const query = {}
  queryString.split('&').filter(Boolean).forEach((pair) => {
    const [rawKey, rawValue = ''] = pair.split('=')
    const key = decodeURIComponent(rawKey || '')
    if (!key) {
      return
    }
    query[key] = decodeURIComponent(rawValue)
  })
  return { path, query }
}

function shouldRetry(method, message, attempt) {
  return attempt < RETRY_DELAYS.length && (
    message.indexOf('timeout') !== -1 ||
    message.indexOf('超时') !== -1 ||
    message.indexOf('request:fail') !== -1 ||
    message.indexOf('云函数') !== -1
  )
}

function showNetworkError(title, message) {
  try {
    wx.showModal({
      title,
      content: message,
      confirmText: '查看诊断',
      cancelText: '关闭',
      success: (res) => {
        if (!res.confirm) {
          return
        }
        const pages = typeof getCurrentPages === 'function' ? getCurrentPages() : []
        const current = pages.length ? pages[pages.length - 1] : null
        if (current && current.route === 'pages/diagnostics/diagnostics') {
          return
        }
        wx.navigateTo({ url: '/pages/diagnostics/diagnostics' })
      }
    })
  } catch (err) {
  }
}

function shouldUseHttp(route) {
  return /^\/api\//.test(String(route || ''))
}

function getHttpBaseUrl() {
  try {
    const app = getApp()
    if (app && app.globalData && app.globalData.httpBaseUrl) {
      return app.globalData.httpBaseUrl
    }
  } catch (err) {
  }
  return wx.getStorageSync('backendBaseUrl') || 'http://82.156.54.232:80'
}

function httpRequest(options, method) {
  return new Promise((resolve, reject) => {
    wx.request({
      url: `${getHttpBaseUrl()}${options.url}`,
      method,
      data: options.data || {},
      timeout: options.timeout || 20000,
      header: {
        'Content-Type': 'application/json',
        ...(options.header || {})
      },
      success: (res) => {
        if (res.statusCode >= 200 && res.statusCode < 300) {
          resolve(res.data)
          return
        }
        const rawMessage = (res.data && res.data.error) || `HTTP ${res.statusCode}`
        const modalMessage = buildDiagnosticMessage({
          title: '后端接口异常',
          method,
          route: options.url,
          message: rawMessage,
          extra: toErrorString(res.data)
        })
        saveNetworkDiagnostic({
          type: 'http-error',
          method,
          route: options.url,
          statusCode: res.statusCode,
          message: modalMessage,
          rawMessage,
          response: toErrorString(res.data)
        })
        reject({ error: rawMessage, handledByModal: false })
      },
      fail: (err) => {
        const rawMessage = (err && err.errMsg) || 'HTTP 请求失败'
        const modalMessage = buildDiagnosticMessage({
          title: '后端请求失败',
          method,
          route: options.url,
          message: rawMessage
        })
        saveNetworkDiagnostic({
          type: 'http-fail',
          method,
          route: options.url,
          message: modalMessage,
          rawMessage
        })
        reject({ error: rawMessage, handledByModal: false })
      }
    })
  })
}

function request(options) {
  const method = (options.method || 'GET').toUpperCase()
  if (shouldUseHttp(options.url)) {
    return httpRequest(options, method)
  }

  const app = getApp()
  if (!app.globalData.cloudReady || !wx.cloud) {
    const message = '当前基础库不支持云开发，请升级微信开发者工具并开启云开发能力。'
    saveNetworkDiagnostic({
      type: 'cloud-unavailable',
      method,
      route: options.url,
      message
    })
    return Promise.reject({ error: message, handledByModal: false })
  }

  const route = parseRoute(options.url)

  const execute = (attempt) => new Promise((resolve, reject) => {
    wx.cloud.callFunction({
      name: CLOUD_FUNCTION_NAME,
      config: { env: wx.cloud.DYNAMIC_CURRENT_ENV },
      data: {
        route: route.path,
        method,
        query: route.query,
        body: options.data || {},
        timeout: options.timeout || 20000
      },
      success: async (res) => {
        const payload = res && res.result ? res.result : {}
        if (payload.ok) {
          resolve(payload.data)
          return
        }

        const rawMessage = payload.error || '云函数返回异常'
        const responseText = toErrorString(payload.details)
        if (shouldRetry(method, rawMessage, attempt)) {
          await wait(RETRY_DELAYS[attempt])
          try {
            const data = await execute(attempt + 1)
            resolve(data)
            return
          } catch (retryErr) {
            reject(retryErr)
            return
          }
        }

        const modalMessage = buildDiagnosticMessage({
          title: '云开发接口异常',
          method,
          route: options.url,
          message: rawMessage,
          extra: responseText ? `响应：${responseText.slice(0, 500)}` : ''
        })
        saveNetworkDiagnostic({
          type: 'cloud-error',
          method,
          route: options.url,
          attempt: attempt + 1,
          message: modalMessage,
          rawMessage,
          response: responseText
        })
        showNetworkError('云开发接口异常', modalMessage)
        reject({ error: rawMessage, handledByModal: true })
      },
      fail: async (err) => {
        const rawMessage = (err && err.errMsg) || '云函数调用失败'
        if (shouldRetry(method, rawMessage, attempt)) {
          await wait(RETRY_DELAYS[attempt])
          try {
            const data = await execute(attempt + 1)
            resolve(data)
            return
          } catch (retryErr) {
            reject(retryErr)
            return
          }
        }

        const modalMessage = buildDiagnosticMessage({
          title: '云函数调用失败',
          method,
          route: options.url,
          message: rawMessage,
          extra: `已自动重试 ${attempt} 次。`
        })
        saveNetworkDiagnostic({
          type: 'cloud-fail',
          method,
          route: options.url,
          attempt: attempt + 1,
          message: modalMessage,
          rawMessage
        })
        showNetworkError('云函数调用失败', modalMessage)
        reject({ error: modalMessage, handledByModal: true })
      }
    })
  })

  return execute(0)
}

module.exports = { request }

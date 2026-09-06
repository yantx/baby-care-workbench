// 统一网络请求封装：自动携带 Token、解包响应信封 {code, message, data}、统一错误处理
const app = getApp()

function request(options) {
  return new Promise((resolve, reject) => {
    wx.request({
      url: app.globalData.apiBase + options.url,
      method: options.method || 'GET',
      data: options.data || {},
      header: Object.assign(
        { 'Content-Type': 'application/json' },
        options.header || {},
        app.globalData.token ? { Authorization: 'Bearer ' + app.globalData.token } : {}
      ),
      success(res) {
        if (res.statusCode === 401) {
          // Token 失效，清理并跳转登录
          app.logout()
          wx.reLaunch({ url: '/pages/login/login' })
          reject(new Error('登录已过期，请重新登录'))
          return
        }
        if (res.statusCode >= 200 && res.statusCode < 300) {
          const body = res.data || {}
          if (body.code === 0) {
            resolve(body.data)
          } else {
            const err = new Error(body.message || '请求失败')
            err.code = body.code
            if (!options.silent) wx.showToast({ title: err.message, icon: 'none' })
            reject(err)
          }
        } else {
          const msg = (res.data && res.data.message) || '请求失败(' + res.statusCode + ')'
          if (!options.silent) wx.showToast({ title: msg, icon: 'none' })
          reject(new Error(msg))
        }
      },
      fail(err) {
        if (!options.silent) wx.showToast({ title: '网络连接失败', icon: 'none' })
        reject(err)
      }
    })
  })
}

const get = (url, data, opts) => request(Object.assign({ url, method: 'GET', data }, opts))
const post = (url, data, opts) => request(Object.assign({ url, method: 'POST', data }, opts))
const put = (url, data, opts) => request(Object.assign({ url, method: 'PUT', data }, opts))
const del = (url, data, opts) => request(Object.assign({ url, method: 'DELETE', data }, opts))

module.exports = { request, get, post, put, del }

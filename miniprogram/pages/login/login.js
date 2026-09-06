const { post, get } = require('../../utils/request')

Page({
  data: {
    loading: false
  },

  onLoad() {
    // 已登录直接进入
    if (getApp().globalData.token) {
      this.routeAfterLogin()
    }
  },

  async handleLogin() {
    if (this.data.loading) return
    this.setData({ loading: true })
    try {
      const app = getApp()
      let payload = { code: this.devCode() }
      // 正式环境：wx.login 获取一次性 code 换 openid
      // Mock 模式（后端未配置微信 appid）：code 是设备标识，用本地固定值保证重复登录同一账号
      if (app.globalData.useWxLogin) {
        try {
          const session = await new Promise((resolve, reject) => {
            wx.login({ success: resolve, fail: reject })
          })
          if (session && session.code) payload.code = session.code
        } catch (e) {
          // 拿不到 code 时用本地标识兜底
        }
      }
      const resp = await post('/api/v1/auth/login', payload)
      app.saveLogin(resp)
      require('../../utils/ws').connect()
      this.routeAfterLogin()
    } catch (e) {
      console.error('登录失败:', e)
    } finally {
      this.setData({ loading: false })
    }
  },

  // Mock 模式设备标识：首次生成后持久化，重复登录同一账号
  devCode() {
    let code = wx.getStorageSync('dev_code')
    if (!code) {
      code = 'dev' + Math.random().toString(36).slice(2, 10)
      wx.setStorageSync('dev_code', code)
    }
    return code
  },

  // 登录后路由：有家庭进首页，无家庭进家庭页创建/加入
  async routeAfterLogin() {
    const app = getApp()
    try {
      const family = await get('/api/v1/families/current', {}, { silent: true })
      if (family && family.family && family.family.id) {
        app.saveFamily(family)
        wx.reLaunch({ url: '/pages/home/home' })
        return
      }
    } catch (e) {
      // 40401 尚未创建或加入家庭 → 去家庭页
    }
    wx.reLaunch({ url: '/pages/family/family' })
  }
})

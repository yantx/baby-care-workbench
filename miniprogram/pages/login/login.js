const { post } = require('../../utils/request')

Page({
  data: {
    loading: false,
    canIUseWxProfile: false
  },

  onLoad() {
    // 已登录直接跳首页
    if (getApp().globalData.token) {
      wx.reLaunch({ url: '/pages/home/home' })
    }
  },

  // 微信登录（测试号场景：无真实 code，使用 dev 模式登录）
  async handleLogin() {
    if (this.data.loading) return
    this.setData({ loading: true })
    try {
      const app = getApp()
      let payload = {}
      try {
        const session = await new Promise((resolve, reject) => {
          wx.login({
            success: resolve,
            fail: reject
          })
        })
        payload.code = session.code
      } catch (e) {
        // 测试号环境拿不到 code 时走 dev 登录
      }
      payload.nickname = '新手妈妈'
      const res = await post('/api/v1/auth/login', payload)
      app.globalData.token = res.token
      app.globalData.user = res.user
      wx.setStorageSync('token', res.token)
      wx.setStorageSync('user', res.user)
      wx.reLaunch({ url: '/pages/home/home' })
    } catch (e) {
      console.error('登录失败:', e)
    } finally {
      this.setData({ loading: false })
    }
  }
})

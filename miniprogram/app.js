// app.js 全局入口
const ws = require('./utils/ws')

App({
  globalData: {
    // 本地开发地址；真机调试请改为局域网 IP（如 http://192.168.1.x:8080）；上线需配置已备案的 HTTPS 域名
    apiBase: 'http://127.0.0.1:8080',
    // true=正式环境用 wx.login code 换 openid；false=Mock 模式用本地设备标识登录
    useWxLogin: false,
    token: '',
    user: null,
    family: null // {family, members, babies, my_role}
  },

  onLaunch() {
    // 恢复登录态
    this.globalData.token = wx.getStorageSync('token') || ''
    this.globalData.user = wx.getStorageSync('user') || null
    this.globalData.family = wx.getStorageSync('family') || null
    // 已登录则建立 WebSocket 长连接
    if (this.globalData.token) {
      ws.connect()
    }
  },

  // 保存登录态
  saveLogin(loginResp) {
    this.globalData.token = loginResp.token
    this.globalData.user = loginResp.user
    wx.setStorageSync('token', loginResp.token)
    wx.setStorageSync('user', loginResp.user)
  },

  saveFamily(familyDetail) {
    this.globalData.family = familyDetail
    wx.setStorageSync('family', familyDetail)
    // 家庭ID可能变化（新建/加入家庭），重连 WebSocket 使房间归属正确
    ws.connect()
  },

  isLoggedIn() {
    return !!this.globalData.token
  },

  logout() {
    this.globalData.token = ''
    this.globalData.user = null
    this.globalData.family = null
    wx.removeStorageSync('token')
    wx.removeStorageSync('user')
    wx.removeStorageSync('family')
    ws.close()
  }
})

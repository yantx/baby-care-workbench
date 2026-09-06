// app.js 全局入口
const ws = require('./utils/ws')

App({
  globalData: {
    // 本地开发地址；真机调试请改为局域网 IP；上线需配置已备案的 HTTPS 域名
    apiBase: 'http://127.0.0.1:8080',
    wsBase: 'ws://127.0.0.1:8080/ws',
    token: '',
    user: null,
    family: null // {family, members, babies, my_role}
  },

  onLaunch() {
    // 恢复登录态
    this.globalData.token = wx.getStorageSync('token') || ''
    this.globalData.user = wx.getStorageSync('user') || null
    this.globalData.family = wx.getStorageSync('family') || null
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
    // token 中的 family_id 已更新，重连 WebSocket
    ws.reconnect(this.globalData.wsBase, this.globalData.token)
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

const app = getApp()

Page({
  data: {
    user: null,
    version: '0.1.0'
  },

  onShow() {
    this.setData({ user: app.globalData.user })
  },

  handleLogout() {
    wx.showModal({
      title: '退出登录',
      content: '确定要退出登录吗？',
      success(res) {
        if (res.confirm) {
          app.globalData.token = ''
          app.globalData.user = null
          wx.removeStorageSync('token')
          wx.removeStorageSync('user')
          wx.reLaunch({ url: '/pages/login/login' })
        }
      }
    })
  },

  handleAbout() {
    wx.showModal({
      title: '关于',
      content: '爸妈育儿工作台 v' + this.data.version + '\n喂奶·睡眠·尿布记录，全家实时同步协作。',
      showCancel: false
    })
  }
})

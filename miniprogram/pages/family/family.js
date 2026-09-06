const { get, post } = require('../../utils/request')

Page({
  data: {
    family: null,
    members: [],
    inviteCode: '',
    loading: true,
    showJoin: false,
    joinCode: '',
    roleMap: {
      mom: { label: '妈妈', icon: '👩', color: '#f48fb1' },
      dad: { label: '爸爸', icon: '👨', color: '#64b5f6' },
      grandma: { label: '奶奶/外婆', icon: '👵', color: '#ffb74d' },
      grandpa: { label: '爷爷/外公', icon: '👴', color: '#aed581' },
      other: { label: '其他家人', icon: '🧑', color: '#90a4ae' }
    },
    roleOptions: ['mom', 'dad', 'grandma', 'grandpa', 'other'],
    newMemberRole: 'dad'
  },

  onShow() {
    this.fetchData()
  },

  async fetchData() {
    try {
      const data = await get('/api/v1/family')
      this.setData({
        family: data.family,
        members: (data.members || []).map((m) => {
          const role = this.data.roleMap[m.role] || this.data.roleMap.other
          return Object.assign({}, m, { roleLabel: role.label, roleIcon: role.icon, roleColor: role.color })
        }),
        inviteCode: data.invite_code || '',
        loading: false
      })
    } catch (e) {
      console.error('获取家庭信息失败:', e)
      this.setData({ loading: false })
    }
  },

  copyInviteCode() {
    if (!this.data.inviteCode) return
    wx.setClipboardData({
      data: this.data.invite_code || this.data.inviteCode,
      success: () => wx.showToast({ title: '邀请码已复制', icon: 'success' })
    })
  },

  showJoinPanel() {
    this.setData({ showJoin: true })
  },

  hideJoinPanel() {
    this.setData({ showJoin: false })
  },

  onJoinCodeInput(e) {
    this.setData({ joinCode: e.detail.value })
  },

  async handleJoin() {
    if (!this.data.joinCode) {
      wx.showToast({ title: '请输入邀请码', icon: 'none' })
      return
    }
    try {
      await post('/api/v1/family/join', { invite_code: this.data.joinCode })
      wx.showToast({ title: '加入成功', icon: 'success' })
      this.setData({ showJoin: false, joinCode: '' })
      this.fetchData()
    } catch (e) {
      console.error('加入家庭失败:', e)
    }
  }
})

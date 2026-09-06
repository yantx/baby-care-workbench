const { get, post } = require('../../utils/request')

// 后端角色枚举：father / mother / grandparent / other
const ROLE_MAP = {
  mother: { label: '妈妈', icon: '👩', color: '#f48fb1' },
  father: { label: '爸爸', icon: '👨', color: '#64b5f6' },
  grandparent: { label: '祖辈', icon: '👵', color: '#ffb74d' },
  other: { label: '其他家人', icon: '🧑', color: '#90a4ae' }
}

Page({
  data: {
    family: null,
    members: [],
    babies: [],
    inviteCode: '',
    loading: true,
    hasFamily: false,
    // 创建家庭表单
    createForm: { name: '', role: 'mother', babyName: '', babyGender: 1, babyBirthday: '' },
    roleList: ['mother', 'father', 'grandparent', 'other'],
    // 加入家庭弹层
    showJoin: false,
    joinCode: '',
    joinRole: 'father'
  },

  onShow() {
    this.fetchData()
  },

  async fetchData() {
    try {
      const data = await get('/api/v1/families/current', {}, { silent: true })
      if (data && data.family && data.family.id) {
        const app = getApp()
        app.saveFamily(data)
        this.applyFamily(data)
      } else {
        this.setData({ hasFamily: false, loading: false })
      }
    } catch (e) {
      // 40401 尚未创建或加入家庭
      this.setData({ hasFamily: false, loading: false })
    }
  },

  applyFamily(data) {
    const roleOf = (role) => ROLE_MAP[role] || ROLE_MAP.other
    this.setData({
      hasFamily: true,
      loading: false,
      family: data.family,
      inviteCode: data.family.invite_code || '',
      babies: data.babies || [],
      members: (data.members || []).map((m) => {
        const r = roleOf(m.role)
        return Object.assign({}, m, { roleLabel: r.label, roleIcon: r.icon, roleColor: r.color })
      })
    })
  },

  // ---------- 创建家庭 ----------
  onFormInput(e) {
    const field = e.currentTarget.dataset.field
    this.setData({ ['createForm.' + field]: e.detail.value })
  },

  selectRole(e) {
    this.setData({ 'createForm.role': e.currentTarget.dataset.value })
  },

  selectGender(e) {
    this.setData({ 'createForm.babyGender': Number(e.currentTarget.dataset.value) })
  },

  onBirthdayChange(e) {
    this.setData({ 'createForm.babyBirthday': e.detail.value })
  },

  async handleCreate() {
    const f = this.data.createForm
    if (!f.name) {
      wx.showToast({ title: '请填写家庭名称', icon: 'none' })
      return
    }
    if (!f.babyName) {
      wx.showToast({ title: '请填写宝宝昵称', icon: 'none' })
      return
    }
    try {
      const data = await post('/api/v1/families', {
        name: f.name,
        role: f.role,
        baby_name: f.babyName,
        baby_gender: f.babyGender,
        baby_birthday: f.babyBirthday ? f.babyBirthday + 'T08:00:00+08:00' : null
      })
      getApp().saveFamily(data)
      this.applyFamily(data)
      wx.showToast({ title: '创建成功', icon: 'success' })
    } catch (e) {
      console.error('创建家庭失败:', e)
    }
  },

  // ---------- 加入家庭 ----------
  showJoinPanel() {
    this.setData({ showJoin: true })
  },

  hideJoinPanel() {
    this.setData({ showJoin: false })
  },

  onJoinCodeInput(e) {
    this.setData({ joinCode: e.detail.value })
  },

  selectJoinRole(e) {
    this.setData({ joinRole: e.currentTarget.dataset.value })
  },

  async handleJoin() {
    if (!this.data.joinCode || this.data.joinCode.length !== 8) {
      wx.showToast({ title: '请输入8位邀请码', icon: 'none' })
      return
    }
    try {
      const data = await post('/api/v1/families/join', {
        invite_code: this.data.joinCode,
        role: this.data.joinRole
      })
      getApp().saveFamily(data)
      this.applyFamily(data)
      this.setData({ showJoin: false, joinCode: '' })
      wx.showToast({ title: '加入成功', icon: 'success' })
    } catch (e) {
      console.error('加入家庭失败:', e)
    }
  },

  copyInviteCode() {
    if (!this.data.inviteCode) return
    wx.setClipboardData({
      data: this.data.inviteCode,
      success: () => wx.showToast({ title: '邀请码已复制', icon: 'success' })
    })
  },

  goHome() {
    wx.switchTab({ url: '/pages/home/home' })
  }
})

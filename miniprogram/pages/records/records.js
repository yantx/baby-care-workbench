const { get, del } = require('../../utils/request')

Page({
  data: {
    date: '',
    records: [],
    loading: true,
    filterType: '',
    filterOptions: [
      { value: '', label: '全部' },
      { value: 'feed', label: '喂养' },
      { value: 'sleep', label: '睡眠' },
      { value: 'diaper', label: '尿布' },
      { value: 'temp', label: '体温' },
      { value: 'medicine', label: '用药' }
    ]
  },

  onLoad() {
    this.setData({ date: this.today() })
  },

  onShow() {
    this.fetchData()
  },

  today() {
    const d = new Date()
    const pad = (n) => String(n).padStart(2, '0')
    return d.getFullYear() + '-' + pad(d.getMonth() + 1) + '-' + pad(d.getDate())
  },

  onDateChange(e) {
    this.setData({ date: e.detail.value }, () => this.fetchData())
  },

  switchFilter(e) {
    this.setData({ filterType: e.currentTarget.dataset.value }, () => this.fetchData())
  },

  async fetchData() {
    this.setData({ loading: true })
    try {
      const params = { date: this.data.date }
      if (this.data.filterType) params.type = this.data.filterType
      const records = await get('/api/v1/records', params)
      this.setData({
        records: (records || []).map((r) => ({
          id: r.id,
          icon: this.typeIcon(r.type),
          label: this.typeLabel(r.type),
          by: r.user_nickname || '家人',
          startTime: this.formatTime(r.start_time),
          endTime: this.formatTime(r.end_time),
          detail: this.detailText(r),
          note: r.note || ''
        })),
        loading: false
      })
    } catch (e) {
      console.error('获取记录失败:', e)
      this.setData({ loading: false })
    }
  },

  typeIcon(t) {
    return { feed: '🍼', sleep: '😴', diaper: '🧷', temp: '🌡️', medicine: '💊' }[t] || '📝'
  },

  typeLabel(t) {
    return { feed: '喂养', sleep: '睡眠', diaper: '换尿布', temp: '体温', medicine: '用药' }[t] || t
  },

  detailText(r) {
    const d = r.detail || {}
    switch (r.type) {
      case 'feed':
        if (d.method === 'bottle') return '瓶喂 ' + (d.amount || 0) + 'ml'
        return '亲喂 ' + (d.side === 'left' ? '左侧' : d.side === 'right' ? '右侧' : '两侧') + ' ' + (d.duration || 0) + '分钟'
      case 'sleep':
        return '睡了 ' + (d.duration_minutes || 0) + ' 分钟'
      case 'diaper':
        return { wet: '尿湿', dirty: '有便便', wet_dirty: '尿湿+便便', dry: '干燥' }[d.diaper_type] || ''
      case 'temp':
        return '体温 ' + d.temp + '℃'
      case 'medicine':
        return (d.name || '') + (d.dose ? ' ' + d.dose : '')
      default:
        return ''
    }
  },

  formatTime(ts) {
    if (!ts) return ''
    const d = new Date(ts.replace(/-/g, '/'))
    const pad = (n) => String(n).padStart(2, '0')
    return pad(d.getHours()) + ':' + pad(d.getMinutes())
  },

  async handleDelete(e) {
    const id = e.currentTarget.dataset.id
    const that = this
    wx.showModal({
      title: '删除记录',
      content: '确定删除这条记录吗？',
      success(res) {
        if (res.confirm) {
          del('/api/v1/records/' + id)
            .then(() => that.fetchData())
            .catch(() => {})
        }
      }
    })
  },

  goAdd() {
    wx.navigateTo({ url: '/pages/record/record' })
  }
})

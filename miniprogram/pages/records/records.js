const { get, del } = require('../../utils/request')
const dt = require('../../utils/datetime')

Page({
  data: {
    date: '',
    records: [],
    total: 0,
    loading: true,
    filterType: '',
    filterOptions: [
      { value: '', label: '全部' },
      { value: 'feeding', label: '喂养' },
      { value: 'sleep', label: '睡眠' },
      { value: 'diaper', label: '尿布' },
      { value: 'temperature', label: '体温' },
      { value: 'medicine', label: '用药' }
    ]
  },

  onLoad() {
    const d = new Date()
    const pad = (n) => String(n).padStart(2, '0')
    this.setData({ date: d.getFullYear() + '-' + pad(d.getMonth() + 1) + '-' + pad(d.getDate()) })
  },

  onShow() {
    this.fetchData()
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
      const params = { date: this.data.date, page: 1, page_size: 50 }
      if (this.data.filterType) params.type = this.data.filterType
      const resp = await get('/api/v1/records', params)
      this.setData({
        records: (resp.list || []).map((r) => this.formatRecord(r)),
        total: resp.total || 0,
        loading: false
      })
    } catch (e) {
      console.error('获取记录失败:', e)
      this.setData({ loading: false })
    }
  },

  formatRecord(r) {
    const typeMap = {
      feeding: { icon: '🍼', label: '喂养' },
      sleep: { icon: '😴', label: '睡眠' },
      diaper: { icon: '🧷', label: '换尿布' },
      temperature: { icon: '🌡️', label: '体温' },
      medicine: { icon: '💊', label: '用药' }
    }
    const t = typeMap[r.type] || { icon: '📝', label: r.type }
    let detail = r.content || ''
    if (!detail) {
      if (r.type === 'feeding' && r.amount_ml) detail = r.amount_ml + 'ml'
      else if (r.type === 'temperature' && r.temp_value) detail = r.temp_value + '℃'
      else detail = t.label
    }
    const durText = dt.durationText(r.started_at, r.ended_at)
    return {
      id: r.id,
      icon: t.icon,
      label: t.label,
      by: (r.recorder && r.recorder.nickname) || '家人',
      startTime: dt.hm(r.started_at),
      endTime: dt.hm(r.ended_at),
      durationText: durText ? '（' + durText + '）' : '',
      detail,
      note: r.note || ''
    }
  },

  handleDelete(e) {
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

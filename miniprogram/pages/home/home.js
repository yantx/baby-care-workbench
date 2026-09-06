const { get } = require('../../utils/request')
const ws = require('../../utils/ws')

Page({
  data: {
    baby: null,
    loading: true,
    summary: {
      feedCount: 0,
      feedAmount: 0,
      sleepMinutes: 0,
      diaperCount: 0,
      lastFeedText: '今天还没有喂养记录',
      lastSleepText: '今天还没有睡眠记录'
    },
    recentRecords: [],
    syncStatus: '' // 实时同步状态提示
  },

  onLoad() {
    this.onRecordChanged = (record) => {
      // 收到其他家庭成员的新记录 → 刷新看板并提示
      this.setData({ syncStatus: '有新记录同步' })
      this.fetchData()
      setTimeout(() => this.setData({ syncStatus: '' }), 2000)
    }
    ws.on('record.created', this.onRecordChanged)
    ws.on('record.updated', this.onRecordChanged)
    ws.on('record.deleted', this.onRecordChanged)
  },

  onUnload() {
    ws.off('record.created', this.onRecordChanged)
    ws.off('record.updated', this.onRecordChanged)
    ws.off('record.deleted', this.onRecordChanged)
  },

  onShow() {
    this.fetchData()
  },

  onPullDownRefresh() {
    this.fetchData().then(() => wx.stopPullDownRefresh())
  },

  async fetchData() {
    try {
      const today = new Date()
      const date = today.getFullYear() + '-' + String(today.getMonth() + 1).padStart(2, '0') + '-' + String(today.getDate()).padStart(2, '0')
      const [summary, records] = await Promise.all([
        get('/api/v1/records/summary', { date }),
        get('/api/v1/records', { date, limit: 20 })
      ])
      this.setData({
        summary: {
          feedCount: summary.feed_count || 0,
          feedAmount: summary.feed_amount || 0,
          sleepMinutes: summary.sleep_minutes || 0,
          diaperCount: summary.diaper_count || 0,
          lastFeedText: summary.last_feed_text || '今天还没有喂养记录',
          lastSleepText: summary.last_sleep_text || '今天还没有睡眠记录'
        },
        recentRecords: (records || []).map(this.formatRecord),
        loading: false
      })
    } catch (e) {
      console.error('获取看板数据失败:', e)
      this.setData({ loading: false })
    }
  },

  formatRecord(r) {
    const typeMap = {
      feed: { icon: '🍼', label: '喂养' },
      sleep: { icon: '😴', label: '睡眠' },
      diaper: { icon: '🧷', label: '换尿布' },
      temp: { icon: '🌡️', label: '体温' },
      medicine: { icon: '💊', label: '用药' }
    }
    const t = typeMap[r.type] || { icon: '📝', label: r.type }
    return {
      id: r.id,
      icon: t.icon,
      label: t.label,
      by: r.user_nickname || '家人',
      time: this.formatTime(r.start_time),
      detail: r.note || t.label
    }
  },

  formatTime(ts) {
    if (!ts) return ''
    const d = new Date(ts.replace(/-/g, '/'))
    return String(d.getHours()).padStart(2, '0') + ':' + String(d.getMinutes()).padStart(2, '0')
  },

  goRecord(e) {
    const type = e.currentTarget.dataset.type || ''
    wx.navigateTo({ url: '/pages/record/record' + (type ? '?type=' + type : '') })
  },

  goRecords() {
    wx.navigateTo({ url: '/pages/records/records' })
  }
})

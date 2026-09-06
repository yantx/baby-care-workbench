const { get } = require('../../utils/request')
const ws = require('../../utils/ws')
const dt = require('../../utils/datetime')

Page({
  data: {
    babyName: '',
    daysOld: '',
    summary: {
      feedingCount: 0,
      feedingTotalML: 0,
      sleepTotalMin: 0,
      diaperCount: 0,
      lastFeedText: '今天还没有喂养记录',
      lastSleepText: '今天还没有睡眠记录'
    },
    recentRecords: [],
    loading: true,
    syncStatus: '' // 实时同步状态提示
  },

  onLoad() {
    this.onRecordChanged = () => {
      // 收到其他家庭成员的记录变更 → 刷新看板并提示
      this.setData({ syncStatus: '⚡ 家人新记录已同步' })
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
    const app = getApp()
    // 每次进入首页都拉取最新家庭信息（days_old 由服务端计算，随日期变化）
    try {
      const f = await get('/api/v1/families/current', {}, { silent: true })
      if (f && f.family && f.family.id) {
        app.saveFamily(f)
      }
    } catch (e) {
      // 接口异常时回退本地缓存
    }
    const fam = app.globalData.family
    if (!fam || !fam.family || !fam.family.id) {
      wx.reLaunch({ url: '/pages/family/family' })
      return
    }

    try {
      const stats = await get('/api/v1/stats/today')
      const baby = (fam.babies && fam.babies[0]) || null
      this.setData({
        babyName: baby ? baby.name : '宝宝',
        daysOld: baby && baby.days_old ? '出生第 ' + baby.days_old + ' 天' : '',
        summary: {
          feedingCount: stats.feeding_count || 0,
          feedingTotalML: stats.feeding_total_ml || 0,
          sleepTotalMin: stats.sleep_total_min || 0,
          diaperCount: stats.diaper_count || 0,
          lastFeedText: stats.last_feeding_at ? '上次喂养 ' + dt.hm(stats.last_feeding_at) : '今天还没有喂养记录',
          lastSleepText: stats.last_sleep_end_at ? '上次醒来 ' + dt.hm(stats.last_sleep_end_at) : '今天还没有睡眠记录'
        },
        recentRecords: (stats.records || []).slice(0, 20).map((r) => this.formatRecord(r)),
        loading: false
      })
    } catch (e) {
      console.error('获取看板数据失败:', e)
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
    return {
      id: r.id,
      icon: t.icon,
      label: t.label,
      by: (r.recorder && r.recorder.nickname) || '家人',
      time: dt.range(r.started_at, r.ended_at),
      detail: r.note || this.detailText(r, t.label)
    }
  },

  detailText(r, fallback) {
    if (r.type === 'feeding' && r.amount_ml) return r.amount_ml + 'ml'
    if (r.type === 'temperature' && r.temp_value) return r.temp_value + '℃'
    if (r.content) return r.content
    return fallback
  },

  goRecord(e) {
    const type = e.currentTarget.dataset.type || ''
    wx.navigateTo({ url: '/pages/record/record' + (type ? '?type=' + type : '') })
  },

  goRecords() {
    wx.navigateTo({ url: '/pages/records/records' })
  }
})

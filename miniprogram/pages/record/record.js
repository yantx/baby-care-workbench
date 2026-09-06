const { post } = require('../../utils/request')

const TYPE_OPTIONS = {
  feed: { label: '喂养', icon: '🍼' },
  sleep: { label: '睡眠', icon: '😴' },
  diaper: { label: '换尿布', icon: '🧷' },
  temp: { label: '体温', icon: '🌡️' },
  medicine: { label: '用药', icon: '💊' }
}

Page({
  data: {
    type: 'feed',
    typeOptions: [],
    // 喂养
    feedMethod: 'breast', // breast | bottle
    feedAmount: 0,
    feedSide: 'left', // left | right | both
    feedDuration: 10,
    // 睡眠
    sleepStart: '',
    sleepEnd: '',
    // 尿布
    diaperType: 'wet', // wet | dirty | wet_dirty | dry
    // 体温
    tempValue: '',
    // 用药
    medicineName: '',
    medicineDose: '',
    // 通用
    note: '',
    saving: false
  },

  onLoad(options) {
    const now = this.nowTime()
    this.setData({
      typeOptions: Object.keys(TYPE_OPTIONS).map((k) => ({
        value: k,
        label: TYPE_OPTIONS[k].label,
        icon: TYPE_OPTIONS[k].icon
      })),
      type: options.type || 'feed',
      sleepStart: now,
      sleepEnd: now
    })
  },

  nowTime() {
    const d = new Date()
    const pad = (n) => String(n).padStart(2, '0')
    return d.getFullYear() + '-' + pad(d.getMonth() + 1) + '-' + pad(d.getDate()) + ' ' + pad(d.getHours()) + ':' + pad(d.getMinutes())
  },

  switchType(e) {
    this.setData({ type: e.currentTarget.dataset.value })
  },

  onInput(e) {
    const field = e.currentTarget.dataset.field
    this.setData({ [field]: e.detail.value })
  },

  // 分段选择器点击（tap 事件，取 data-value）
  onSegTap(e) {
    const field = e.currentTarget.dataset.field
    this.setData({ [field]: e.currentTarget.dataset.value })
  },

  onPicker(e) {
    const field = e.currentTarget.dataset.field
    this.setData({ [field]: e.detail.value })
  },

  async handleSave() {
    if (this.data.saving) return
    const payload = this.buildPayload()
    if (!payload) return

    this.setData({ saving: true })
    try {
      await post('/api/v1/records', payload)
      wx.showToast({ title: '记录成功', icon: 'success' })
      setTimeout(() => wx.navigateBack(), 800)
    } catch (e) {
      console.error('保存失败:', e)
    } finally {
      this.setData({ saving: false })
    }
  },

  buildPayload() {
    const d = this.data
    let detail = {}

    switch (d.type) {
      case 'feed':
        detail = {
          method: d.feedMethod,
          amount: d.feedMethod === 'bottle' ? Number(d.feedAmount) || 0 : 0,
          side: d.feedMethod === 'breast' ? d.feedSide : '',
          duration: Number(d.feedDuration) || 0
        }
        break
      case 'sleep':
        if (!d.sleepStart || !d.sleepEnd) {
          wx.showToast({ title: '请选择睡眠起止时间', icon: 'none' })
          return null
        }
        detail = { start: d.sleepStart, end: d.sleepEnd }
        break
      case 'diaper':
        detail = { diaperType: d.diaperType }
        break
      case 'temp': {
        const t = Number(d.tempValue)
        if (!t || t < 34 || t > 43) {
          wx.showToast({ title: '请输入有效体温(34-43℃)', icon: 'none' })
          return null
        }
        detail = { temp: t }
        if (t >= 38) {
          wx.showModal({
            title: '体温偏高提醒',
            content: '宝宝体温≥38℃，属于发热，建议及时就医并咨询医生。',
            showCancel: false
          })
        }
        break
      }
      case 'medicine':
        if (!d.medicineName) {
          wx.showToast({ title: '请填写药品名称', icon: 'none' })
          return null
        }
        detail = { name: d.medicineName, dose: d.medicineDose }
        break
    }

    return {
      type: d.type,
      start_time: d.type === 'sleep' ? d.sleepStart : this.nowTime(),
      detail,
      note: d.note || ''
    }
  }
})

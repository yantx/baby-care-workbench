const { post } = require('../../utils/request')

const TYPE_OPTIONS = {
  feeding: { label: '喂养', icon: '🍼' },
  sleep: { label: '睡眠', icon: '😴' },
  diaper: { label: '换尿布', icon: '🧷' },
  temperature: { label: '体温', icon: '🌡️' },
  medicine: { label: '用药', icon: '💊' }
}

Page({
  data: {
    type: 'feeding',
    typeOptions: [],
    // 喂养
    feedMethod: 'breast', // breast 亲喂 / bottle 瓶喂
    feedAmount: '',
    feedSide: 'left', // left / right / both
    feedDuration: '',
    // 睡眠
    sleepStartDate: '',
    sleepStartTime: '',
    sleepEndDate: '',
    sleepEndTime: '',
    // 尿布
    diaperType: 'wet', // wet / dirty / mixed / dry
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
    const now = new Date()
    const pad = (n) => String(n).padStart(2, '0')
    const today = now.getFullYear() + '-' + pad(now.getMonth() + 1) + '-' + pad(now.getDate())
    const nowHM = pad(now.getHours()) + ':' + pad(now.getMinutes())
    this.setData({
      typeOptions: Object.keys(TYPE_OPTIONS).map((k) => ({
        value: k,
        label: TYPE_OPTIONS[k].label,
        icon: TYPE_OPTIONS[k].icon
      })),
      type: TYPE_OPTIONS[options.type] ? options.type : 'feeding',
      sleepStartDate: today,
      sleepStartTime: nowHM,
      sleepEndDate: today,
      sleepEndTime: nowHM
    })
  },

  nowTime() {
    const d = new Date()
    const pad = (n) => String(n).padStart(2, '0')
    return d.getFullYear() + '-' + pad(d.getMonth() + 1) + '-' + pad(d.getDate()) + 'T' + pad(d.getHours()) + ':' + pad(d.getMinutes()) + ':00+08:00'
  },

  switchType(e) {
    this.setData({ type: e.currentTarget.dataset.value })
  },

  onInput(e) {
    const field = e.currentTarget.dataset.field
    this.setData({ [field]: e.detail.value })
  },

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
    const app = getApp()
    const family = app.globalData.family
    const baby = family && family.babies && family.babies[0]
    if (!baby) {
      wx.showToast({ title: '请先创建宝宝档案', icon: 'none' })
      return
    }
    const payload = this.buildPayload()
    if (!payload) return
    payload.baby_id = baby.id

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

    switch (d.type) {
      case 'feeding': {
        const payload = {
          type: 'feeding',
          started_at: this.nowTime(),
          details: { method: d.feedMethod }
        }
        if (d.feedMethod === 'bottle') {
          const amount = Number(d.feedAmount)
          if (!amount || amount <= 0) {
            wx.showToast({ title: '请输入奶量', icon: 'none' })
            return null
          }
          payload.amount_ml = amount
          payload.content = '瓶喂 ' + amount + 'ml'
        } else {
          const sideText = { left: '左侧', right: '右侧', both: '两侧' }[d.feedSide]
          const duration = Number(d.feedDuration) || 0
          payload.content = '亲喂·' + sideText + (duration ? ' ' + duration + '分钟' : '')
          payload.details.side = d.feedSide
          payload.details.duration = duration
        }
        payload.note = d.note || ''
        return payload
      }
      case 'sleep': {
        const d2 = this.data
        const start = d2.sleepStartDate + 'T' + d2.sleepStartTime + ':00+08:00'
        const end = d2.sleepEndDate + 'T' + d2.sleepEndTime + ':00+08:00'
        if (new Date(end.replace(/-/g, '/')) <= new Date(start.replace(/-/g, '/'))) {
          wx.showToast({ title: '醒来时间需晚于入睡时间', icon: 'none' })
          return null
        }
        return {
          type: 'sleep',
          started_at: start,
          ended_at: end,
          note: d.note || ''
        }
      }
      case 'diaper':
        return {
          type: 'diaper',
          started_at: this.nowTime(),
          content: { wet: '尿湿', dirty: '有便便', mixed: '尿湿+便便', dry: '干燥' }[d.diaperType],
          details: { diaper_type: d.diaperType },
          note: d.note || ''
        }
      case 'temperature': {
        const t = Number(d.tempValue)
        if (!t || t < 34 || t > 43) {
          wx.showToast({ title: '请输入有效体温(34-43℃)', icon: 'none' })
          return null
        }
        if (t >= 38) {
          wx.showModal({
            title: '体温偏高提醒',
            content: '宝宝体温≥38℃，属于发热，建议及时就医并咨询医生。',
            showCancel: false
          })
        }
        return {
          type: 'temperature',
          started_at: this.nowTime(),
          temp_value: t,
          note: d.note || ''
        }
      }
      case 'medicine': {
        if (!d.medicineName) {
          wx.showToast({ title: '请填写药品名称', icon: 'none' })
          return null
        }
        return {
          type: 'medicine',
          started_at: this.nowTime(),
          content: d.medicineName + (d.medicineDose ? ' ' + d.medicineDose : ''),
          details: { name: d.medicineName, dose: d.medicineDose },
          note: d.note || ''
        }
      }
    }
    return null
  }
})

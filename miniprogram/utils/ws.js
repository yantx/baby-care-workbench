// WebSocket 实时同步管理器：家庭多角色数据实时同步（核心差异化能力）
const app = getApp()

let task = null
let reconnectCount = 0
let heartbeatTimer = null
const listeners = {} // eventType -> [callback]

const MAX_RECONNECT = 10
const HEARTBEAT_INTERVAL = 25000

function wsUrl() {
  const base = app.globalData.baseUrl.replace(/^http/, 'ws')
  return base + '/ws?token=' + (app.globalData.token || '')
}

// 注册事件监听：on('record.created', cb)
function on(eventType, callback) {
  if (!listeners[eventType]) listeners[eventType] = []
  listeners[eventType].push(callback)
}

function off(eventType, callback) {
  if (!listeners[eventType]) return
  if (callback) {
    listeners[eventType] = listeners[eventType].filter((cb) => cb !== callback)
  } else {
    listeners[eventType] = []
  }
}

function dispatch(message) {
  const cbs = listeners[message.type] || []
  cbs.forEach((cb) => {
    try {
      cb(message.data)
    } catch (e) {
      console.error('[ws] listener error:', e)
    }
  })
  // 通配监听
  ;(listeners['*'] || []).forEach((cb) => {
    try {
      cb(message)
    } catch (e) {
      console.error('[ws] wildcard listener error:', e)
    }
  })
}

function connect() {
  if (!app.globalData.token) return
  close(true)

  task = wx.connectSocket({ url: wsUrl(), fail: () => scheduleReconnect() })

  task.onOpen(() => {
    reconnectCount = 0
    startHeartbeat()
    dispatch({ type: 'ws.open' })
  })

  task.onMessage((res) => {
    try {
      const message = JSON.parse(res.data)
      if (message.type === 'pong') return
      dispatch(message)
    } catch (e) {
      console.warn('[ws] 非JSON消息:', res.data)
    }
  })

  task.onClose(() => {
    stopHeartbeat()
    dispatch({ type: 'ws.close' })
    scheduleReconnect()
  })

  task.onError(() => {
    stopHeartbeat()
    scheduleReconnect()
  })
}

function scheduleReconnect() {
  if (reconnectCount >= MAX_RECONNECT) {
    console.warn('[ws] 超过最大重连次数，停止重连')
    return
  }
  reconnectCount++
  const delay = Math.min(1000 * Math.pow(2, reconnectCount), 30000) // 指数退避，最长30s
  setTimeout(() => {
    if (app.globalData.token) connect()
  }, delay)
}

function startHeartbeat() {
  stopHeartbeat()
  heartbeatTimer = setInterval(() => {
    send({ type: 'ping' })
  }, HEARTBEAT_INTERVAL)
}

function stopHeartbeat() {
  if (heartbeatTimer) {
    clearInterval(heartbeatTimer)
    heartbeatTimer = null
  }
}

function send(data) {
  if (task && app.globalData.token) {
    task.send({ data: JSON.stringify(data), fail: () => {} })
  }
}

function close(silent) {
  stopHeartbeat()
  if (task) {
    const t = task
    task = null
    if (silent) {
      // 静默关闭：先摘除回调避免触发重连
      t.onClose(() => {})
      t.onError(() => {})
    }
    t.close({ code: 1000 })
  }
}

module.exports = { connect, close, send, on, off }

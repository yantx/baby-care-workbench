// datetime.js 统一日期时间解析与格式化
// 后端返回 RFC3339 格式（2026-09-06T10:30:00+08:00）。
// 注意：小程序 iOS 环境不支持 new Date('YYYY-MM-DD HH:mm:ss')（需用 '/' 分隔），
// 而把含 'T' 的 ISO 串替换成 '/' 反而会产生非法日期 —— 这正是历史上出现 NaN 的根因。
// 因此这里按格式分流解析，所有页面统一使用本模块，禁止再手写 replace(/-/g,'/')。

function parse(ts) {
  if (!ts) return null
  if (ts instanceof Date) return isNaN(ts.getTime()) ? null : ts
  const s = String(ts)
  // ISO 格式（含 T）直接解析
  if (s.indexOf('T') > 0) {
    const d = new Date(s)
    if (!isNaN(d.getTime())) return d
  }
  // 普通格式 YYYY-MM-DD / YYYY-MM-DD HH:mm[:ss]，替换分隔符兼容 iOS
  const d2 = new Date(s.replace(/-/g, '/'))
  return isNaN(d2.getTime()) ? null : d2
}

const pad = (n) => String(n).padStart(2, '0')

// hm "HH:mm"；解析失败返回 ''
function hm(ts) {
  const d = parse(ts)
  return d ? pad(d.getHours()) + ':' + pad(d.getMinutes()) : ''
}

// mdhm "MM-DD HH:mm"（跨天记录用）
function mdhm(ts) {
  const d = parse(ts)
  return d ? pad(d.getMonth() + 1) + '-' + pad(d.getDate()) + ' ' + hm(d) : ''
}

// range "10:30~10:50"；同一时刻只返回一个时间
function range(start, end) {
  const s = hm(start)
  const e = hm(end)
  if (!s) return ''
  return e && e !== s ? s + '~' + e : s
}

// durationMin 起止间隔分钟数；无结束时间或解析失败返回 0
function durationMin(start, end) {
  const s = parse(start)
  const e = parse(end)
  if (!s || !e) return 0
  const min = Math.round((e.getTime() - s.getTime()) / 60000)
  return min > 0 ? min : 0
}

// durationText "20分钟" / "1小时5分"
function durationText(start, end) {
  const min = durationMin(start, end)
  if (!min) return ''
  if (min < 60) return min + '分钟'
  return Math.floor(min / 60) + '小时' + (min % 60 ? min % 60 + '分' : '')
}

module.exports = { parse, hm, mdhm, range, durationMin, durationText }

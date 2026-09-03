// 通用格式化:ISO 时间 → 业务时区 Asia/Shanghai 固定展示(YYYY-MM-DD HH:mm:ss)。
// 后端传 RFC3339 带偏移(存储 UTC);admin 业务时间统一按上海墙钟展示,
// 不随浏览器本地时区漂移;空值返回 —,非法值原样返回便于排查。
const SHANGHAI = new Intl.DateTimeFormat('en-US', {
  timeZone: 'Asia/Shanghai',
  year: 'numeric',
  month: '2-digit',
  day: '2-digit',
  hour: '2-digit',
  minute: '2-digit',
  second: '2-digit',
  hourCycle: 'h23',
})

function shanghaiParts(d: Date): Record<string, string> {
  const part: Record<string, string> = {}
  for (const p of SHANGHAI.formatToParts(d)) part[p.type] = p.value
  return part
}

export function fmtTime(iso: string | undefined | null): string {
  if (!iso) return '—'
  const d = new Date(iso)
  if (Number.isNaN(d.getTime())) return iso
  const p = shanghaiParts(d)
  return [p.year, p.month, p.day].join('-') + ' ' + [p.hour, p.minute, p.second].join(':')
}

// 业务时区日界:某时刻在上海墙钟的 YYYY-MM-DD;「今日」类筛选据此对齐业务日。
export function bizDateKey(input: string | number | Date): string {
  const d = input instanceof Date ? input : new Date(input)
  if (Number.isNaN(d.getTime())) return ''
  const p = shanghaiParts(d)
  return [p.year, p.month, p.day].join('-')
}

/** 金额紧凑展示:大于一万或一亿时使用单位,避免表格和卡片横向溢出。 */
export function fmtFee(n: number): string {
  if (!Number.isFinite(Number(n))) return '¥0'
  const value = Number(n)
  const absolute = Math.abs(value)
  if (absolute >= 100_000_000) return `¥${(value / 100_000_000).toFixed(2)}亿`
  if (absolute >= 10_000) return `¥${(value / 10_000).toFixed(2)}万`
  return `¥${value.toLocaleString('en-US', { minimumFractionDigits: 2, maximumFractionDigits: 2 })}`
}

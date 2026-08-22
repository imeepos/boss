// 通用格式化:ISO 时间 → 浏览器本地时区展示(截秒)。
// 后端传 RFC3339 带偏移(存储 UTC);字符串截断会原样露出 UTC 墙钟,
// 菲律宾用户会看到慢 8 小时的时间,必须经 Date 转本地时区。
export function fmtTime(iso: string | undefined | null): string {
  if (!iso) return '—'
  const d = new Date(iso)
  if (Number.isNaN(d.getTime())) return iso
  const p = (n: number) => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${p(d.getMonth() + 1)}-${p(d.getDate())} ${p(d.getHours())}:${p(d.getMinutes())}:${p(d.getSeconds())}`
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

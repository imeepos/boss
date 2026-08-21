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

export function fmtFee(n: number): string {
  return Number(n).toFixed(2)
}

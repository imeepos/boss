// 通用格式化:ISO 时间 → 本地展示(截秒)。
export function fmtTime(iso: string | undefined | null): string {
  if (!iso) return '—'
  return iso.replace('T', ' ').slice(0, 19)
}

export function fmtFee(n: number): string {
  return Number(n).toFixed(2)
}

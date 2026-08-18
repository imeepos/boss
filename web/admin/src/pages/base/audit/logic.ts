// 审计日志页纯逻辑:关键字(操作人/内容) + 类型 + 日期前缀 三条件筛选。

export interface AuditLog {
  logId: string
  time: string
  operator: string
  type: string
  action: string
  ip: string
}

export function filterAuditLogs(
  rows: AuditLog[],
  keyword: string,
  type: string,
  date: string,
): AuditLog[] {
  const kw = keyword.trim()
  return rows.filter((r) => {
    const hitKw = !kw || r.operator.includes(kw) || r.action.includes(kw)
    const hitType = !type || r.type === type
    const hitDate = !date || r.time.startsWith(date)
    return hitKw && hitType && hitDate
  })
}

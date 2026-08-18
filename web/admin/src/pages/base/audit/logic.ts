// 审计日志页纯逻辑:契约 Entry(audit.Entry)→ 页面行映射 + 三条件筛选。

/** 契约形状:pkg/audit Entry(GET /audit-logs → items)。 */
export interface AuditEntry {
  id: number
  accountId: number
  operator: string
  action: string
  targetType: string
  targetId: string
  detail: string
  ip: string
  createdAt: string
}

/** 页面行:列名对齐 docs/admin/audit.html 原型。 */
export interface AuditLog {
  logId: string
  time: string
  operator: string
  type: string
  action: string
  ip: string
}

/** Entry → 行:时间取 createdAt;类型=Action(数据/状态/权限变更);内容=目标+详情摘要。 */
export function toAuditLog(e: AuditEntry): AuditLog {
  return {
    logId: String(e.id),
    time: formatTime(e.createdAt),
    operator: e.operator || `#${e.accountId}`,
    type: e.action,
    action: summarizeAction(e),
    ip: e.ip,
  }
}

function summarizeAction(e: AuditEntry): string {
  const target = e.targetType + (e.targetId ? `#${e.targetId}` : '')
  const extra = detailSummary(e.detail)
  return extra ? `${target} ${extra}` : target
}

function detailSummary(detail: string): string {
  if (!detail || detail === '{}') return ''
  try {
    const obj = JSON.parse(detail) as Record<string, unknown>
    const parts = Object.entries(obj)
      .filter(([k]) => k !== 'op')
      .map(([k, v]) => `${k}=${String(v)}`)
    return parts.length ? `(${parts.join(', ')})` : ''
  } catch {
    return ''
  }
}

/** ISO 时间 → 展示文本(YYYY-MM-DD HH:mm:ss,去时区尾巴)。 */
export function formatTime(iso: string): string {
  if (!iso) return ''
  return iso.replace('T', ' ').replace(/(\.\d+|Z|[+-]\d{2}:\d{2})$/, '')
}

/** 筛选:关键字(操作人/内容) + 类型 + 日期前缀。 */
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

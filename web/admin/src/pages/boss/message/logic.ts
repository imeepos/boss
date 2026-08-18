// 消息中心页纯逻辑:类型映射 + 过滤,与 UI 解耦便于测试。
// 契约: GET/POST /worker-messages、GET/POST /notices、PUT /notices/{id}/toggle(worker.yaml)。

/** 师傅站内消息(worker.Message,level INFO/WARN/URGENT)。 */
export interface WorkerMessageEntry {
  id: number
  workerId: number
  level: string
  title: string
  content: string
  sentAt: string
  read: boolean
}

/** 师傅公告(worker.Notice,active 控制师傅端可见)。 */
export interface NoticeEntry {
  id: number
  title: string
  category: string
  active: boolean
  publishedAt: string
}

export const MESSAGE_LEVELS = ['INFO', 'WARN', 'URGENT'] as const

/** ISO 时间 → 本地短格式(YYYY-MM-DD HH:mm);空值返回 '-'。 */
export function fmtTime(iso: string): string {
  if (!iso) return '-'
  const d = new Date(iso)
  if (Number.isNaN(d.getTime())) return iso
  const p = (n: number) => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${p(d.getMonth() + 1)}-${p(d.getDate())} ${p(d.getHours())}:${p(d.getMinutes())}`
}

/** 容错数字解析:非数字/空返回 0。 */
export function toId(raw: string): number {
  const n = Number(raw.trim())
  return Number.isFinite(n) && n > 0 ? Math.floor(n) : 0
}

/** 师傅消息过滤:关键字(标题/内容) + 级别 + 已读状态。 */
export function filterMessages(
  rows: WorkerMessageEntry[],
  keyword: string,
  level: string,
  read: string,
): WorkerMessageEntry[] {
  const kw = keyword.trim().toLowerCase()
  return rows.filter((r) => {
    if (kw && !(`${r.title}${r.content}`.toLowerCase().includes(kw))) return false
    if (level && r.level !== level) return false
    if (read === 'read' && !r.read) return false
    if (read === 'unread' && r.read) return false
    return true
  })
}

/** 公告过滤:关键字(标题/分类);activeOnly=true 只看上架。 */
export function filterNotices(rows: NoticeEntry[], keyword: string, activeOnly: boolean): NoticeEntry[] {
  const kw = keyword.trim().toLowerCase()
  return rows.filter((r) => {
    if (activeOnly && !r.active) return false
    if (kw && !(`${r.title}${r.category}`.toLowerCase().includes(kw))) return false
    return true
  })
}

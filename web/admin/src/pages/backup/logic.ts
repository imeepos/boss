// 数据备份迁移页逻辑:契约映射 + 纯展示辅助(单测覆盖)。
// 契约: GET /backup/jobs → {items:[backup.Job], total};字段口径 docs/contract/fields.md 1.5.6。
import { getAuthToken } from '../../api/client'
import { apiBaseUrl } from '../../lib/serverConfig'

/** 后端 backup.Job(迁移 000095)。 */
export interface BackupJobEntry {
  id: number
  kind: 'backup' | 'restore'
  scope: 'all' | 'tables'
  tables: string[]
  status: 'running' | 'succeeded' | 'failed'
  fileName: string
  sizeBytes: number
  tableCount: number
  rowCount: number
  error?: string
  operator: string
  createdAt: string
  finishedAt?: string
}

export function toJob(e: BackupJobEntry): BackupJobEntry {
  return {
    id: Number(e.id),
    kind: e.kind,
    scope: e.scope,
    tables: Array.isArray(e.tables) ? e.tables : [],
    status: e.status,
    fileName: e.fileName ?? '',
    sizeBytes: Number(e.sizeBytes ?? 0),
    tableCount: Number(e.tableCount ?? 0),
    rowCount: Number(e.rowCount ?? 0),
    error: e.error,
    operator: e.operator ?? '',
    createdAt: e.createdAt ?? '',
    finishedAt: e.finishedAt,
  }
}

/** 字节数人性化:0 起 '0 B',保留 1 位小数。 */
export function formatBytes(n: number): string {
  if (!Number.isFinite(n) || n <= 0) return '0 B'
  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  let v = n
  let i = 0
  while (v >= 1024 && i < units.length - 1) {
    v /= 1024
    i++
  }
  return `${v >= 100 || i === 0 ? Math.round(v) : v.toFixed(1)} ${units[i]}`
}

/** RFC3339 → 'YYYY-MM-DD HH:mm'(本地时区,失败原样返回)。 */
export function formatTime(iso: string): string {
  if (!iso) return ''
  const d = new Date(iso)
  if (Number.isNaN(d.getTime())) return iso
  const p = (x: number) => String(x).padStart(2, '0')
  return `${d.getFullYear()}-${p(d.getMonth() + 1)}-${p(d.getDate())} ${p(d.getHours())}:${p(d.getMinutes())}`
}

/** 表选择器过滤:大小写不敏感子串。 */
export function filterTables(tables: string[], kw: string): string[] {
  const k = kw.trim().toLowerCase()
  if (!k) return tables
  return tables.filter((t) => t.toLowerCase().includes(k))
}

/** 42300(CodeResourceBusy):已有任务在执行。 */
export function isBusyError(e: unknown): boolean {
  return typeof e === 'object' && e !== null && 'code' in e && (e as { code: number }).code === 42300
}

/** 下载归档:带 token 拉 blob 后触发浏览器保存。 */
export async function downloadArchive(id: number, fileName: string): Promise<void> {
  const headers: Record<string, string> = {}
  const token = getAuthToken()
  if (token) headers.Authorization = `Bearer ${token}`
  const res = await fetch(`${apiBaseUrl()}/backup/jobs/${id}/file`, { headers })
  if (!res.ok) throw new Error(`HTTP ${res.status}`)
  const blob = await res.blob()
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = fileName || `backup-${id}.jsonl.gz`
  a.click()
  URL.revokeObjectURL(url)
}

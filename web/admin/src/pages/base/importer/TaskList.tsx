// 导入任务历史(sys 域,menu:importer):近 200 条导入记录,操作人/行数/时间。
import { useEffect, useState } from 'react'
import { apiFetch } from '../../../api/client'
import { useT } from '../../../i18n'
import type { Translations } from '../../../i18n/types'
import { formatTime } from '../../base/audit/logic'
import { EmptyState, ErrorBanner, ToolbarButton } from '../../../components/business/page-head'
import { Input } from '../../../components/ui/input'
import { Table, TableHeader, TableBody, TableRow, TableHead, TableCell } from '../../../components/ui/table'
import { buildTaskQuery, type TaskQueryFilters } from './taskQuery'

export interface ImportTaskRow {
  id: number
  kind: string
  operator: string
  total: number
  imported: number
  failed: number
  skipped: number
  detail?: string
  createdAt: string
}

/** kind 展示:geo/地址沿用旧文案;entity:<kind> 映射实体名,其余原样。 */
export function taskKindLabel(im: Translations['pages']['importer'], kind: string): string {
  if (kind === 'geo') return im.taskKindGeo
  if (kind.startsWith('entity:')) return im.entityNames[kind.slice('entity:'.length)] ?? kind
  if (kind === 'addr' || kind === 'addresses') return im.taskKindAddr
  return kind
}

export function ImportTaskList({ refreshKey = 0 }: { refreshKey?: number }) {
  const t = useT()
  const im = t.pages.importer
  const [rows, setRows] = useState<ImportTaskRow[]>([])
  const [error, setError] = useState('')
  const [kind, setKind] = useState('')
  const [operator, setOperator] = useState('')
  const [from, setFrom] = useState('')
  const [to, setTo] = useState('')

  const load = () => {
    setError('')
    const filters: TaskQueryFilters = { kind, operator, from, to }
    apiFetch<{ items: ImportTaskRow[] }>(`/import-tasks${buildTaskQuery(filters)}`)
      .then((d) => setRows(d?.items ?? []))
      .catch((e) => setError(e instanceof Error ? e.message : im.loadFail))
  }
  useEffect(load, [refreshKey]) // eslint-disable-line react-hooks/exhaustive-deps
  useEffect(() => {
    const refresh = () => {
      if (document.visibilityState === 'visible') load()
    }
    window.addEventListener('focus', refresh)
    document.addEventListener('visibilitychange', refresh)
    return () => {
      window.removeEventListener('focus', refresh)
      document.removeEventListener('visibilitychange', refresh)
    }
    // Filters are read from the current render when listeners are installed.
  }, [kind, operator, from, to]) // eslint-disable-line react-hooks/exhaustive-deps

  return (
    <div className="overflow-x-auto px-4 pb-4">
      <div className="flex flex-wrap items-center gap-2 pb-2.5">
        <h3 className="m-0 text-sm">{im.tasksTitle}</h3>
        <Input className="w-40" placeholder={im.taskKindFilter} value={kind} onChange={(e) => setKind(e.target.value)} />
        <Input className="w-32" placeholder={im.taskOperatorFilter} value={operator} onChange={(e) => setOperator(e.target.value)} />
        <Input type="date" className="w-40" value={from} onChange={(e) => setFrom(e.target.value)} aria-label={im.taskFromFilter} />
        <Input type="date" className="w-40" value={to} onChange={(e) => setTo(e.target.value)} aria-label={im.taskToFilter} />
        <div className="flex-1" />
        <ToolbarButton onClick={load}>{t.pages.audit.refresh}</ToolbarButton>
      </div>
      {error && <div className="mb-3"><ErrorBanner message={error} /></div>}
      <Table>
        <TableHeader>
          <TableRow>{im.taskColumns.map((c) => <TableHead key={c}>{c}</TableHead>)}</TableRow>
        </TableHeader>
        <TableBody>
          {rows.map((r) => (
            <TableRow key={r.id}>
              <TableCell>{r.id}</TableCell>
              <TableCell>{taskKindLabel(im, r.kind)}</TableCell>
              <TableCell>{r.operator || '—'}</TableCell>
              <TableCell>{r.total}</TableCell>
              <TableCell>{r.imported}</TableCell>
              <TableCell>{r.failed}</TableCell>
              <TableCell>{r.skipped}</TableCell>
              <TableCell>{formatTime(r.createdAt)}</TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
      {rows.length === 0 && !error && <EmptyState text={t.pages.company.empty} />}
    </div>
  )
}

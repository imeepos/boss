// 导入任务历史(sys 域,menu:importer):近 200 条导入记录,操作人/行数/时间。
import { useEffect, useState } from 'react'
import { apiFetch } from '../../../api/client'
import { useT } from '../../../i18n'
import type { Translations } from '../../../i18n/types'
import { formatTime } from '../../base/audit/logic'
import { EmptyState } from '../../../components/business'

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
  return im.taskKindAddr
}

export function ImportTaskList({ refreshKey = 0 }: { refreshKey?: number }) {
  const t = useT()
  const im = t.pages.importer
  const [rows, setRows] = useState<ImportTaskRow[]>([])
  const [error, setError] = useState('')
  const [kind, setKind] = useState('')

  const load = () => {
    setError('')
    const query = kind ? `?kind=${encodeURIComponent(kind)}` : ''
    apiFetch<{ items: ImportTaskRow[] }>(`/import-tasks${query}`)
      .then((d) => setRows(d?.items ?? []))
      .catch((e) => setError(e instanceof Error ? e.message : im.loadFail))
  }
  useEffect(load, [refreshKey]) // eslint-disable-line react-hooks/exhaustive-deps

  return (
    <div className="overflow-x-auto px-4 pb-4">
      <div className="flex items-center gap-2 pb-2.5">
        <h3 className="m-0 text-sm">{im.tasksTitle}</h3>
        <input className="h-8 w-44 rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-2.5 text-xs text-[var(--shell-input-text)] outline-none placeholder:text-[var(--shell-input-placeholder)]" placeholder={im.taskKindFilter} value={kind}
          onChange={(e) => setKind(e.target.value)} onKeyDown={(e) => { if (e.key === 'Enter') load() }} />
        <div className="flex-1" />
        <button className="h-8 cursor-pointer rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-4 text-[13px] text-[var(--shell-content-text)] hover:border-[var(--color-border-hover)] hover:text-[var(--shell-heading)]" onClick={load}>{t.pages.audit.refresh}</button>
      </div>
      <table className="w-full border-collapse text-[13px] text-[var(--shell-content-text)]">
        <thead className="h-11 px-3 text-left text-xs font-medium whitespace-nowrap border-b border-[var(--shell-side-border)] bg-[var(--shell-menu-hover-bg)] text-[var(--shell-group-title)]"><tr>{im.taskColumns.map((c) => <th key={c} className="h-11 px-3 text-left text-xs font-medium whitespace-nowrap border-b border-[var(--shell-side-border)] bg-[var(--shell-menu-hover-bg)] text-[var(--shell-group-title)]">{c}</th>)}</tr></thead>
        <tbody>
          {rows.map((r) => (
            <tr key={r.id}>
              <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{r.id}</td>
              <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{taskKindLabel(im, r.kind)}</td>
              <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{r.operator || '—'}</td>
              <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{r.total}</td>
              <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{r.imported}</td>
              <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{r.failed}</td>
              <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{r.skipped}</td>
              <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{formatTime(r.createdAt)}</td>
            </tr>
          ))}
        </tbody>
      </table>
      {rows.length === 0 && !error && <EmptyState text={t.pages.company.empty} />}
      {error && <div className="mx-4 mb-3 rounded-sm border border-[color-mix(in_srgb,var(--color-danger)_25%,transparent)] bg-[color-mix(in_srgb,var(--color-danger)_8%,transparent)] px-3 py-2 text-[13px] text-[var(--color-danger)]">{error}</div>}
    </div>
  )
}

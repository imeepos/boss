// 导入任务历史(sys 域,menu:importer):近 200 条导入记录,操作人/行数/时间。
import { useEffect, useState } from 'react'
import { apiFetch } from '../../../api/client'
import { useT } from '../../../i18n'
import { formatTime } from '../../base/audit/logic'

export interface ImportTaskRow {
  id: number
  kind: string
  operator: string
  imported: number
  failed: number
  detail?: string
  createdAt: string
}

export function ImportTaskList() {
  const t = useT()
  const im = t.pages.importer
  const [rows, setRows] = useState<ImportTaskRow[]>([])
  const [error, setError] = useState('')

  const load = () => {
    setError('')
    apiFetch<{ items: ImportTaskRow[] }>('/import-tasks')
      .then((d) => setRows(d?.items ?? []))
      .catch((e) => setError(e instanceof Error ? e.message : im.loadFail))
  }
  useEffect(load, []) // eslint-disable-line react-hooks/exhaustive-deps

  return (
    <div className="overflow-x-auto px-4 pb-4" style={{ margin: '0 16px 24px' }}>
      <div className="flex flex-wrap items-center gap-2 p-4" style={{ padding: '0 0 10px' }}>
        <h3 style={{ margin: 0, fontSize: 14 }}>{im.tasksTitle}</h3>
        <span className="spacer" />
        <button className="h-8 cursor-pointer rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-4 text-[13px] text-[var(--shell-content-text)] hover:border-[var(--color-border-hover)] hover:text-[var(--shell-heading)]" onClick={load}>{t.pages.audit.refresh}</button>
      </div>
      <table className="w-full border-collapse text-[13px] text-[var(--shell-content-text)]">
        <thead className="h-11 px-3 text-left text-xs font-medium whitespace-nowrap border-b border-[var(--shell-side-border)] bg-[var(--shell-menu-hover-bg)] text-[var(--shell-group-title)]"><tr>{im.taskColumns.map((c) => <th key={c} className="h-11 px-3 text-left text-xs font-medium whitespace-nowrap border-b border-[var(--shell-side-border)] bg-[var(--shell-menu-hover-bg)] text-[var(--shell-group-title)]">{c}</th>)}</tr></thead>
        <tbody>
          {rows.map((r) => (
            <tr key={r.id}>
              <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{r.id}</td>
              <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{r.kind === 'geo' ? im.taskKindGeo : im.taskKindAddr}</td>
              <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{r.operator || '—'}</td>
              <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{r.imported}</td>
              <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{r.failed}</td>
              <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{formatTime(r.createdAt)}</td>
            </tr>
          ))}
        </tbody>
      </table>
      {rows.length === 0 && !error && <div className="py-8 text-center text-[13px] text-[var(--shell-group-title)]">{t.pages.company.empty}</div>}
      {error && <div className="mx-4 mb-3 rounded-sm border border-[color-mix(in_srgb,var(--color-danger)_25%,transparent)] bg-[color-mix(in_srgb,var(--color-danger)_8%,transparent)] px-3 py-2 text-[13px] text-[var(--color-danger)]">{error}</div>}
    </div>
  )
}

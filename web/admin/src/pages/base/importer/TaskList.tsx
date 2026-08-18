// 导入任务历史(sys 域,menu:importer):近 200 条导入记录,操作人/行数/时间。
import { useEffect, useState } from 'react'
import { apiFetch } from '../../../api/client'
import { useT } from '../../../i18n'
import { formatTime } from '../../base/audit/logic'
import '../../org/org.css'

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
    <div className="org-table-wrap" style={{ margin: '0 16px 24px' }}>
      <div className="org-toolbar" style={{ padding: '0 0 10px' }}>
        <h3 style={{ margin: 0, fontSize: 14 }}>{im.tasksTitle}</h3>
        <span className="spacer" />
        <button className="org-btn" onClick={load}>{t.pages.audit.refresh}</button>
      </div>
      <table className="org-table">
        <thead><tr>{im.taskColumns.map((c) => <th key={c}>{c}</th>)}</tr></thead>
        <tbody>
          {rows.map((r) => (
            <tr key={r.id}>
              <td>{r.id}</td>
              <td>{r.kind === 'geo' ? im.taskKindGeo : im.taskKindAddr}</td>
              <td>{r.operator || '—'}</td>
              <td>{r.imported}</td>
              <td>{r.failed}</td>
              <td>{formatTime(r.createdAt)}</td>
            </tr>
          ))}
        </tbody>
      </table>
      {rows.length === 0 && !error && <div className="org-empty">{t.pages.company.empty}</div>}
      {error && <div className="org-error">{error}</div>}
    </div>
  )
}

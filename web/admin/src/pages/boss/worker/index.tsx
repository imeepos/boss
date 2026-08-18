// 师傅管理页:契约 GET /worker-groups + GET /workers?groupId(在职/离职状态)。
import { useEffect, useState } from 'react'
import { apiFetch } from '../../../api/client'
import { useT } from '../../../i18n'
import { PageHead, pagerTexts } from '../../org/shared'
import { Pagination } from '../../../components/Pagination'
import { fmtTime } from '../../../lib/format'
import { pageSlice, type WorkerGroupRow, type WorkerRow } from '../types'
import '../../org/org.css'

export default function WorkerPage() {
  const t = useT()
  const w = t.pages.workerPage
  const [rows, setRows] = useState<WorkerRow[]>([])
  const [groups, setGroups] = useState<WorkerGroupRow[]>([])
  const [error, setError] = useState('')
  const [groupId, setGroupId] = useState(0)
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(10)
  const [busy, setBusy] = useState(false)

  const load = () => {
    setError('')
    setBusy(true)
    apiFetch<{ items: WorkerRow[] }>('/workers', { query: { groupId: groupId || undefined } })
      .then((d) => setRows(d?.items ?? []))
      .catch((e) => setError(e instanceof Error ? e.message : w.loadFail))
      .finally(() => setBusy(false))
  }
  useEffect(() => {
    apiFetch<{ items: WorkerGroupRow[] }>('/worker-groups')
      .then((d) => setGroups(d?.items ?? []))
      .catch(() => setGroups([]))
    load()
  }, []) // eslint-disable-line react-hooks/exhaustive-deps

  const groupName = (id: number) => groups.find((g) => g.id === id)?.name ?? `#${id}`
  const slice = pageSlice(rows, page, pageSize)

  return (
    <div>
      <PageHead title={w.title} desc={w.desc} />
      <div className="org-card">
        <div className="org-toolbar">
          <select className="org-select" value={groupId ? String(groupId) : ''}
            onChange={(e) => { setGroupId(Number(e.target.value) || 0); setPage(1); load() }}>
            <option value="">{w.allGroup}</option>
            {groups.map((g) => <option key={g.id} value={g.id}>{g.name}</option>)}
          </select>
          <span className="spacer" />
          <button className="org-btn" disabled={busy} onClick={load}>{t.pages.audit.refresh}</button>
        </div>
        {error ? <div className="org-error">{error}</div> : (
          <div className="org-table-wrap">
            <table className="org-table">
              <thead><tr>{w.columns.map((x) => <th key={x}>{x}</th>)}</tr></thead>
              <tbody>
                {slice.map((r) => (
                  <tr key={r.id}>
                    <td>{r.staffNo}</td>
                    <td>{r.name}</td>
                    <td>{groupName(r.groupId)}</td>
                    <td>{r.phone || '—'}</td>
                    <td>{r.status === 1 ? w.active : w.left}</td>
                    <td>{fmtTime(r.joinedAt)}</td>
                  </tr>
                ))}
                {!slice.length && <tr><td colSpan={6}><div className="org-empty">{w.empty}</div></td></tr>}
              </tbody>
            </table>
          </div>
        )}
        <div className="org-footer">
          <Pagination total={rows.length} page={page} pageSize={pageSize}
            onPage={setPage} onSize={setPageSize} {...pagerTexts(w)} />
        </div>
      </div>
    </div>
  )
}

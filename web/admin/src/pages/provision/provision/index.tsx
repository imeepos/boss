// 下发任务页:契约 GET /provision-tasks(裸 items)+ POST /provision-tasks/{taskNo}/retry。
import { useEffect, useState } from 'react'
import { apiFetch } from '../../../api/client'
import { useT } from '../../../i18n'
import { PageHead, pagerTexts } from '../../org/shared'
import { StatusTag } from '../../../components/StatusTag'
import { Pagination } from '../../../components/Pagination'
import { pageSlice, type ProvisionTaskRow } from '../types'
import '../../org/org.css'

const STATUSES = ['PENDING', 'DOING', 'DONE', 'FAILED'] as const

export default function ProvisionTaskPage() {
  const t = useT()
  const p = t.pages.provisionPage
  const [rows, setRows] = useState<ProvisionTaskRow[]>([])
  const [error, setError] = useState('')
  const [status, setStatus] = useState('')
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(10)
  const [busy, setBusy] = useState(false)

  const load = () => {
    setError('')
    setBusy(true)
    apiFetch<{ items: ProvisionTaskRow[] }>('/provision-tasks')
      .then((x) => setRows(x?.items ?? []))
      .catch((e) => setError(e instanceof Error ? e.message : p.loadFail))
      .finally(() => setBusy(false))
  }
  useEffect(load, []) // eslint-disable-line react-hooks/exhaustive-deps

  const retry = async (taskNo: string) => {
    if (busy || !window.confirm(p.retryConfirm)) return
    setBusy(true)
    try {
      await apiFetch(`/provision-tasks/${encodeURIComponent(taskNo)}/retry`, { method: 'POST' })
      load()
    } catch (e) {
      setError(e instanceof Error ? e.message : p.actionFail)
      setBusy(false)
    }
  }

  const filtered = status ? rows.filter((x) => x.status === status) : rows
  const slice = pageSlice(filtered, page, pageSize)

  return (
    <div>
      <PageHead title={p.title} desc={p.desc} />
      <div className="org-card">
        <div className="org-toolbar">
          <select className="org-select" value={status}
            onChange={(e) => { setStatus(e.target.value); setPage(1) }}>
            <option value="">{p.allStatus}</option>
            {STATUSES.map((s) => <option key={s} value={s}>{s}</option>)}
          </select>
          <span className="spacer" />
          <button className="org-btn" disabled={busy} onClick={load}>{t.pages.audit.refresh}</button>
        </div>
        {error ? <div className="org-error">{error}</div> : (
          <div className="org-table-wrap">
            <table className="org-table">
              <thead><tr>{p.columns.map((x) => <th key={x}>{x}</th>)}</tr></thead>
              <tbody>
                {slice.map((x) => (
                  <tr key={x.id}>
                    <td>{x.taskNo}</td>
                    <td>{x.orderId ? `#${x.orderId}` : '—'}</td>
                    <td>{x.stageEvent || '—'}</td>
                    <td>#{x.loAccountId}</td>
                    <td>#{x.templateId}</td>
                    <td><StatusTag domain="task" value={x.status} /></td>
                    <td>
                      {x.status === 'FAILED' ? (
                        <span className="org-act">
                          <button disabled={busy} onClick={() => retry(x.taskNo)}>{p.retry}</button>
                        </span>
                      ) : '—'}
                    </td>
                  </tr>
                ))}
                {!slice.length && <tr><td colSpan={7}><div className="org-empty">{p.empty}</div></td></tr>}
              </tbody>
            </table>
          </div>
        )}
        <div className="org-footer">
          <Pagination total={filtered.length} page={page} pageSize={pageSize}
            onPage={setPage} onSize={setPageSize} {...pagerTexts(p)} />
        </div>
      </div>
    </div>
  )
}

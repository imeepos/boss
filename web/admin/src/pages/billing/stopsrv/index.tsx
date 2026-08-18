// 停复机执行页:契约 GET /stop-resume-tasks(customerId 过滤);失败任务 POST /stop-resume-tasks/:taskId/retry。
import { useEffect, useState } from 'react'
import { apiFetch } from '../../../api/client'
import { useT } from '../../../i18n'
import { PageHead, pagerTexts } from '../../org/shared'
import { StatusTag } from '../../../components/StatusTag'
import { Pagination } from '../../../components/Pagination'
import { pageSlice, type StopResumeTaskRow } from '../types'
import '../../org/org.css'

export default function StopSrvPage() {
  const t = useT()
  const s = t.pages.stopsrv
  const [rows, setRows] = useState<StopResumeTaskRow[]>([])
  const [error, setError] = useState('')
  const [customerId, setCustomerId] = useState('')
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(10)
  const [busy, setBusy] = useState(false)

  const load = () => {
    setError('')
    setBusy(true)
    apiFetch<{ items: StopResumeTaskRow[] }>('/stop-resume-tasks', {
      query: { customerId: customerId || undefined },
    })
      .then((d) => setRows(d?.items ?? []))
      .catch((e) => setError(e instanceof Error ? e.message : s.loadFail))
      .finally(() => setBusy(false))
  }
  useEffect(load, []) // eslint-disable-line react-hooks/exhaustive-deps

  const retry = async (taskId: number) => {
    if (busy) return
    if (!window.confirm(s.retryConfirm)) return
    setBusy(true)
    try {
      await apiFetch(`/stop-resume-tasks/${taskId}/retry`, { method: 'POST' })
      load()
    } catch (e) {
      setError(e instanceof Error ? e.message : s.actionFail)
    } finally {
      setBusy(false)
    }
  }

  const slice = pageSlice(rows, page, pageSize)

  return (
    <div>
      <PageHead title={s.title} desc={s.desc} />
      <div className="org-card">
        <div className="org-toolbar">
          <input className="org-input" type="number" placeholder={s.filterCustomer}
            value={customerId} onChange={(e) => { setCustomerId(e.target.value); setPage(1) }} />
          <span className="spacer" />
          <button className="org-btn" disabled={busy} onClick={load}>{t.pages.audit.refresh}</button>
        </div>
        {error ? <div className="org-error">{error}</div> : (
          <div className="org-table-wrap">
            <table className="org-table">
              <thead><tr>{s.columns.map((x) => <th key={x}>{x}</th>)}</tr></thead>
              <tbody>
                {slice.map((r) => (
                  <tr key={r.id}>
                    <td>#{r.id}</td>
                    <td>#{r.customerId}</td>
                    <td>#{r.loAccountId}</td>
                    <td>{r.action === 'STOP' ? s.actionStop : s.actionResume}</td>
                    <td><StatusTag domain="task" value={r.status} /></td>
                    <td>
                      {r.status === 'FAILED' ? (
                        <span className="org-act">
                          <button disabled={busy} onClick={() => retry(r.id)}>{s.retry}</button>
                        </span>
                      ) : '—'}
                    </td>
                  </tr>
                ))}
                {!slice.length && <tr><td colSpan={6}><div className="org-empty">{s.empty}</div></td></tr>}
              </tbody>
            </table>
          </div>
        )}
        <div className="org-footer">
          <Pagination total={rows.length} page={page} pageSize={pageSize}
            onPage={setPage} onSize={setPageSize} {...pagerTexts(s)} />
        </div>
      </div>
    </div>
  )
}

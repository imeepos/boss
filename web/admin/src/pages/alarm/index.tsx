// 告警列表页:契约 GET /alarms?resourceId + POST /alarms/:id/ack + POST /alarms/batch-retest。
import { useEffect, useState } from 'react'
import { apiFetch } from '../../api/client'
import { useT } from '../../i18n'
import { PageHead, pagerTexts } from '../org/shared'
import { StatusTag } from '../../components/StatusTag'
import { Pagination } from '../../components/Pagination'
import { Drawer } from '../../components/Drawer'
import { fmtTime } from '../../lib/format'
import { pageSlice, type AlarmRow } from '../quad/types'
import '../org/org.css'

export default function AlarmPage() {
  const t = useT()
  const a = t.pages.alarmPage
  const [rows, setRows] = useState<AlarmRow[]>([])
  const [error, setError] = useState('')
  const [hint, setHint] = useState('')
  const [resourceId, setResourceId] = useState('')
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(10)
  const [busy, setBusy] = useState(false)
  const [retestOpen, setRetestOpen] = useState(false)
  const [scope, setScope] = useState('')

  const load = () => {
    setError('')
    setBusy(true)
    apiFetch<{ items: AlarmRow[] }>('/alarms', { query: { resourceId: resourceId || undefined } })
      .then((d) => setRows(d?.items ?? []))
      .catch((e) => setError(e instanceof Error ? e.message : a.loadFail))
      .finally(() => setBusy(false))
  }
  useEffect(load, []) // eslint-disable-line react-hooks/exhaustive-deps

  const ack = async (alarmId: number) => {
    if (busy || !window.confirm(a.ackConfirm)) return
    setBusy(true)
    try {
      await apiFetch(`/alarms/${alarmId}/ack`, { method: 'POST' })
      load()
    } catch (e) {
      setError(e instanceof Error ? e.message : a.actionFail)
    } finally {
      setBusy(false)
    }
  }

  const retest = async () => {
    if (busy || !scope.trim()) return
    setBusy(true)
    setHint('')
    try {
      const r = await apiFetch<{ taskNo: string }>('/alarms/batch-retest', {
        method: 'POST', body: { scope: scope.trim() },
      })
      setHint(a.retestDone.replace('{no}', r?.taskNo ?? ''))
      setRetestOpen(false)
      setScope('')
    } catch (e) {
      setError(e instanceof Error ? e.message : a.actionFail)
    } finally {
      setBusy(false)
    }
  }

  const slice = pageSlice(rows, page, pageSize)

  return (
    <div>
      <PageHead title={a.title} desc={a.desc} />
      <div className="org-card">
        <div className="org-toolbar">
          <input className="org-input" type="number" placeholder={a.filterResource}
            value={resourceId} onChange={(e) => { setResourceId(e.target.value); setPage(1) }} />
          <span className="spacer" />
          <button className="org-btn" disabled={busy} onClick={load}>{t.pages.audit.refresh}</button>
          <button className="org-btn org-btn-primary" onClick={() => setRetestOpen(true)}>{a.batchRetest}</button>
        </div>
        {hint && <div style={{ padding: '4px 12px', color: '#1677ff', fontSize: 13 }}>{hint}</div>}
        {error ? <div className="org-error">{error}</div> : (
          <div className="org-table-wrap">
            <table className="org-table">
              <thead><tr>{a.columns.map((x) => <th key={x}>{x}</th>)}</tr></thead>
              <tbody>
                {slice.map((x) => (
                  <tr key={x.id}>
                    <td>{x.alarmNo || `#${x.id}`}</td>
                    <td><StatusTag domain="alarmLevel" value={x.level} /></td>
                    <td>{x.source}</td>
                    <td>{x.content || '—'}</td>
                    <td><StatusTag domain="alarmStatus" value={x.status} /></td>
                    <td>{fmtTime(x.createdAt)}</td>
                    <td>
                      {x.status === 'OPEN' ? (
                        <span className="org-act">
                          <button disabled={busy} onClick={() => ack(x.id)}>{a.ack}</button>
                        </span>
                      ) : '—'}
                    </td>
                  </tr>
                ))}
                {!slice.length && <tr><td colSpan={7}><div className="org-empty">{a.empty}</div></td></tr>}
              </tbody>
            </table>
          </div>
        )}
        <div className="org-footer">
          <Pagination total={rows.length} page={page} pageSize={pageSize}
            onPage={setPage} onSize={setPageSize} {...pagerTexts(a)} />
        </div>
      </div>
      {retestOpen && (
        <Drawer title={a.batchRetest} onClose={() => setRetestOpen(false)}
          footer={
            <>
              <button className="org-btn" onClick={() => setRetestOpen(false)}>{t.pages.company.cancel}</button>
              <button className="org-btn org-btn-primary" disabled={busy || !scope.trim()} onClick={retest}>
                {busy ? t.pages.account.submitting : t.pages.company.save}
              </button>
            </>
          }>
          <div className="org-form">
            <div className="org-field">
              <label><span className="req">*</span>{a.fScope}</label>
              <input className="org-input" value={scope} placeholder={a.pScope}
                onChange={(e) => setScope(e.target.value)} />
              {!scope.trim() && scope !== '' && <span className="acc-err">{a.eScope}</span>}
            </div>
          </div>
        </Drawer>
      )}
    </div>
  )
}

// 预占与释放页:契约 GET /reserves?portId、POST /reserves/:reserveId/release。
import { useEffect, useState } from 'react'
import { apiFetch } from '../../../api/client'
import { useT } from '../../../i18n'
import { PageHead, pagerTexts } from '../../org/shared'
import { StatusTag } from '../../../components/StatusTag'
import { Pagination } from '../../../components/Pagination'
import { pageSlice, type ReserveRow } from '../types'
import '../../org/org.css'

export default function ReservePage() {
  const t = useT()
  const r = t.pages.reservePage
  const [rows, setRows] = useState<ReserveRow[]>([])
  const [error, setError] = useState('')
  const [portId, setPortId] = useState('')
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(10)
  const [busy, setBusy] = useState(false)

  const load = () => {
    setError('')
    setBusy(true)
    apiFetch<{ items: ReserveRow[] }>('/reserves', {
      query: { portId: portId || undefined },
    })
      .then((d) => setRows(d?.items ?? []))
      .catch((e) => setError(e instanceof Error ? e.message : r.loadFail))
      .finally(() => setBusy(false))
  }
  useEffect(load, []) // eslint-disable-line react-hooks/exhaustive-deps

  const release = async (reserveId: number) => {
    if (busy || !window.confirm(r.releaseConfirm)) return
    setBusy(true)
    try {
      await apiFetch(`/reserves/${reserveId}/release`, { method: 'POST' })
      load()
    } catch (e) {
      setError(e instanceof Error ? e.message : r.actionFail)
    } finally {
      setBusy(false)
    }
  }

  const slice = pageSlice(rows, page, pageSize)

  return (
    <div>
      <PageHead title={r.title} desc={r.desc} />
      <div className="org-card">
        <div className="org-toolbar">
          <input className="org-input" type="number" placeholder={r.filterPort}
            value={portId} onChange={(e) => { setPortId(e.target.value); setPage(1) }} />
          <span className="spacer" />
          <button className="org-btn" disabled={busy} onClick={load}>{t.pages.audit.refresh}</button>
        </div>
        {error ? <div className="org-error">{error}</div> : (
          <div className="org-table-wrap">
            <table className="org-table">
              <thead><tr>{r.columns.map((x) => <th key={x}>{x}</th>)}</tr></thead>
              <tbody>
                {slice.map((x) => (
                  <tr key={x.id}>
                    <td>#{x.id}</td>
                    <td>#{x.portId}</td>
                    <td>{x.orderId ? `#${x.orderId}` : '—'}</td>
                    <td><StatusTag domain="reserve" value={x.status} /></td>
                    <td>
                      {x.status === 'HELD' ? (
                        <span className="org-act">
                          <button disabled={busy} onClick={() => release(x.id)}>{r.release}</button>
                        </span>
                      ) : '—'}
                    </td>
                  </tr>
                ))}
                {!slice.length && <tr><td colSpan={5}><div className="org-empty">{r.empty}</div></td></tr>}
              </tbody>
            </table>
          </div>
        )}
        <div className="org-footer">
          <Pagination total={rows.length} page={page} pageSize={pageSize}
            onPage={setPage} onSize={setPageSize} {...pagerTexts(r)} />
        </div>
      </div>
    </div>
  )
}

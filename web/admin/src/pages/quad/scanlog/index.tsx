// 扫码绑定记录页:契约 GET /scan-logs?orderId。
import { useEffect, useState } from 'react'
import { apiFetch } from '../../../api/client'
import { useT } from '../../../i18n'
import { PageHead, pagerTexts } from '../../org/shared'
import { StatusTag } from '../../../components/StatusTag'
import { Pagination } from '../../../components/Pagination'
import { pageSlice, type ScanLogRow } from '../types'
import '../../org/org.css'

export default function ScanLogPage() {
  const t = useT()
  const s = t.pages.scanlogPage
  const [rows, setRows] = useState<ScanLogRow[]>([])
  const [error, setError] = useState('')
  const [orderId, setOrderId] = useState('')
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(10)
  const [busy, setBusy] = useState(false)

  const load = () => {
    setError('')
    setBusy(true)
    apiFetch<{ items: ScanLogRow[] }>('/scan-logs', { query: { orderId: orderId || undefined } })
      .then((d) => setRows(d?.items ?? []))
      .catch((e) => setError(e instanceof Error ? e.message : s.loadFail))
      .finally(() => setBusy(false))
  }
  useEffect(load, []) // eslint-disable-line react-hooks/exhaustive-deps

  const slice = pageSlice(rows, page, pageSize)

  return (
    <div>
      <PageHead title={s.title} desc={s.desc} />
      <div className="org-card">
        <div className="org-toolbar">
          <input className="org-input" type="number" placeholder={s.filterOrder}
            value={orderId} onChange={(e) => { setOrderId(e.target.value); setPage(1) }} />
          <span className="spacer" />
          <button className="org-btn" disabled={busy} onClick={load}>{t.pages.audit.refresh}</button>
        </div>
        {error ? <div className="org-error">{error}</div> : (
          <div className="org-table-wrap">
            <table className="org-table">
              <thead><tr>{s.columns.map((x) => <th key={x}>{x}</th>)}</tr></thead>
              <tbody>
                {slice.map((x) => (
                  <tr key={x.id}>
                    <td>#{x.id}</td>
                    <td>#{x.orderId}</td>
                    <td>{x.workerName || (x.workerId ? `#${x.workerId}` : '—')}</td>
                    <td>{x.tagId ? `#${x.tagId}` : '—'}</td>
                    <td><StatusTag domain="scan" value={x.result} /></td>
                  </tr>
                ))}
                {!slice.length && <tr><td colSpan={5}><div className="org-empty">{s.empty}</div></td></tr>}
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

// 欠费停复机页:契约 GET /arrears;操作 POST /arrears/:customerId/stop|resume(W6 即时生效)。
import { useEffect, useState } from 'react'
import { apiFetch } from '../../../api/client'
import { useT } from '../../../i18n'
import { PageHead, pagerTexts } from '../../org/shared'
import { Pagination } from '../../../components/Pagination'
import { pageSlice, type ArrearsRow } from '../types'
import { fmtFee } from '../../../lib/format'
import '../../org/org.css'

export default function ArrearsPage() {
  const t = useT()
  const a = t.pages.arrearsPage
  const [rows, setRows] = useState<ArrearsRow[]>([])
  const [error, setError] = useState('')
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(10)
  const [busy, setBusy] = useState(false)

  const load = () => {
    setError('')
    setBusy(true)
    apiFetch<{ items: ArrearsRow[] }>('/arrears')
      .then((d) => setRows(d?.items ?? []))
      .catch((e) => setError(e instanceof Error ? e.message : a.loadFail))
      .finally(() => setBusy(false))
  }
  useEffect(load, []) // eslint-disable-line react-hooks/exhaustive-deps

  const act = async (customerId: number, action: 'stop' | 'resume') => {
    if (busy) return
    const confirmMsg = action === 'stop' ? a.stopConfirm : a.resumeConfirm
    if (!window.confirm(confirmMsg)) return
    setBusy(true)
    try {
      await apiFetch(`/arrears/${customerId}/${action}`, { method: 'POST' })
      load()
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
          <span className="spacer" />
          <button className="org-btn" disabled={busy} onClick={load}>{t.pages.audit.refresh}</button>
        </div>
        {error ? <div className="org-error">{error}</div> : (
          <div className="org-table-wrap">
            <table className="org-table">
              <thead><tr>{a.columns.map((x) => <th key={x}>{x}</th>)}</tr></thead>
              <tbody>
                {slice.map((r) => (
                  <tr key={r.customerId}>
                    <td>{r.customer || `#${r.customerId}`}</td>
                    <td>{fmtFee(r.amount)}</td>
                    <td>{r.days}</td>
                    <td>{r.status || '—'}</td>
                    <td>
                      <span className="org-act">
                        <button disabled={busy} onClick={() => act(r.customerId, 'stop')}>{a.stop}</button>
                        <span className="sep">|</span>
                        <button disabled={busy} onClick={() => act(r.customerId, 'resume')}>{a.resume}</button>
                      </span>
                    </td>
                  </tr>
                ))}
                {!slice.length && <tr><td colSpan={5}><div className="org-empty">{a.empty}</div></td></tr>}
              </tbody>
            </table>
          </div>
        )}
        <div className="org-footer">
          <Pagination total={rows.length} page={page} pageSize={pageSize}
            onPage={setPage} onSize={setPageSize} {...pagerTexts(a)} />
        </div>
      </div>
    </div>
  )
}

// 报障与投诉页(CS 域):契约 GET /complaints(裸列表)+ POST /complaints/:ticketNo/close。
import { useEffect, useState } from 'react'
import { apiFetch } from '../../../api/client'
import { useT } from '../../../i18n'
import { PageHead, pagerTexts } from '../../org/shared'
import { StatusTag } from '../../../components/StatusTag'
import { Pagination } from '../../../components/Pagination'
import { pageSlice, type ComplaintRow } from '../types'
import '../../org/org.css'

export default function ComplaintPage() {
  const t = useT()
  const c = t.pages.complaintPage
  const [rows, setRows] = useState<ComplaintRow[]>([])
  const [error, setError] = useState('')
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(10)
  const [busy, setBusy] = useState(false)

  const load = () => {
    setError('')
    setBusy(true)
    apiFetch<ComplaintRow[]>('/complaints')
      .then((x) => setRows(Array.isArray(x) ? x : []))
      .catch((e) => setError(e instanceof Error ? e.message : c.loadFail))
      .finally(() => setBusy(false))
  }
  useEffect(load, []) // eslint-disable-line react-hooks/exhaustive-deps

  const close = async (ticketNo: string) => {
    if (busy || !window.confirm(c.closeConfirm)) return
    setBusy(true)
    try {
      await apiFetch(`/complaints/${encodeURIComponent(ticketNo)}/close`, { method: 'POST' })
      load()
    } catch (e) {
      setError(e instanceof Error ? e.message : c.actionFail)
    } finally {
      setBusy(false)
    }
  }

  const slice = pageSlice(rows, page, pageSize)

  return (
    <div>
      <PageHead title={c.title} desc={c.desc} />
      <div className="org-card">
        <div className="org-toolbar">
          <span className="spacer" />
          <button className="org-btn" disabled={busy} onClick={load}>{t.pages.audit.refresh}</button>
        </div>
        {error ? <div className="org-error">{error}</div> : (
          <div className="org-table-wrap">
            <table className="org-table">
              <thead><tr>{c.columns.map((x) => <th key={x}>{x}</th>)}</tr></thead>
              <tbody>
                {slice.map((x) => (
                  <tr key={x.id}>
                    <td>{x.ticketNo}</td>
                    <td>#{x.customerId}</td>
                    <td>{x.orderId ? `#${x.orderId}` : '—'}</td>
                    <td>{c.types[x.type] ?? x.type}</td>
                    <td><StatusTag domain="complaint" value={x.status} /></td>
                    <td>
                      {x.status !== 'CLOSED' ? (
                        <span className="org-act">
                          <button disabled={busy} onClick={() => close(x.ticketNo)}>{c.close}</button>
                        </span>
                      ) : '—'}
                    </td>
                  </tr>
                ))}
                {!slice.length && <tr><td colSpan={6}><div className="org-empty">{c.empty}</div></td></tr>}
              </tbody>
            </table>
          </div>
        )}
        <div className="org-footer">
          <Pagination total={rows.length} page={page} pageSize={pageSize}
            onPage={setPage} onSize={setPageSize} {...pagerTexts(c)} />
        </div>
      </div>
    </div>
  )
}

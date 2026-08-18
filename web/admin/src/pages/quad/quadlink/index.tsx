// 关联查询页:契约 GET /quad-links;列名以 fields.md §5.1 为准。
import { useEffect, useState } from 'react'
import { apiFetch } from '../../../api/client'
import { useT } from '../../../i18n'
import { PageHead, pagerTexts } from '../../org/shared'
import { StatusTag } from '../../../components/StatusTag'
import { Pagination } from '../../../components/Pagination'
import { pageSlice, type QuadLinkRow } from '../types'
import '../../org/org.css'

export default function QuadLinkPage() {
  const t = useT()
  const q = t.pages.quadLinkPage
  const [rows, setRows] = useState<QuadLinkRow[]>([])
  const [error, setError] = useState('')
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(10)
  const [busy, setBusy] = useState(false)

  const load = () => {
    setError('')
    setBusy(true)
    apiFetch<{ items: QuadLinkRow[] }>('/quad-links')
      .then((d) => setRows(d?.items ?? []))
      .catch((e) => setError(e instanceof Error ? e.message : q.loadFail))
      .finally(() => setBusy(false))
  }
  useEffect(load, []) // eslint-disable-line react-hooks/exhaustive-deps

  const slice = pageSlice(rows, page, pageSize)

  return (
    <div>
      <PageHead title={q.title} desc={q.desc} />
      <div className="org-card">
        <div className="org-toolbar">
          <span className="spacer" />
          <button className="org-btn" disabled={busy} onClick={load}>{t.pages.audit.refresh}</button>
        </div>
        {error ? <div className="org-error">{error}</div> : (
          <div className="org-table-wrap">
            <table className="org-table">
              <thead><tr>{q.columns.map((x) => <th key={x}>{x}</th>)}</tr></thead>
              <tbody>
                {slice.map((x) => (
                  <tr key={x.id}>
                    <td>#{x.id}</td>
                    <td>#{x.assetId}</td>
                    <td>#{x.customerId}</td>
                    <td>#{x.portId}</td>
                    <td>#{x.addressId}</td>
                    <td>{x.legalEntityName || `#${x.legalEntityId}`}</td>
                    <td><StatusTag domain="quad" value={x.status} /></td>
                  </tr>
                ))}
                {!slice.length && <tr><td colSpan={7}><div className="org-empty">{q.empty}</div></td></tr>}
              </tbody>
            </table>
          </div>
        )}
        <div className="org-footer">
          <Pagination total={rows.length} page={page} pageSize={pageSize}
            onPage={setPage} onSize={setPageSize} {...pagerTexts(q)} />
        </div>
      </div>
    </div>
  )
}

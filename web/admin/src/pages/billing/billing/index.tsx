// 出账管理页:列名以 fields.md §3.3 + billing.html 为准;契约 GET /bills(customerId 过滤)。
import { useEffect, useState } from 'react'
import { apiFetch } from '../../../api/client'
import { useT } from '../../../i18n'
import { DetailDrawer, PageHead, pagerTexts } from '../../org/shared'
import { StatusTag } from '../../../components/StatusTag'
import { Pagination } from '../../../components/Pagination'
import { pageSlice, type BillRow } from '../types'
import { InvoicePanel } from './invoices'
import { BillingRunModal, INVOICES_REFRESH } from './run-modal'
import { fmtFee } from '../../../lib/format'
import '../../org/org.css'

export default function BillPage() {
  const t = useT()
  const b = t.pages.billPage
  const [rows, setRows] = useState<BillRow[]>([])
  const [error, setError] = useState('')
  const [customerId, setCustomerId] = useState('')
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(10)
  const [detail, setDetail] = useState<BillRow | null>(null)
  const [runOpen, setRunOpen] = useState(false)
  const [busy, setBusy] = useState(false)

  const load = () => {
    setError('')
    setBusy(true)
    apiFetch<{ items: BillRow[] }>('/bills', {
      query: { customerId: customerId || undefined },
    })
      .then((d) => setRows(d?.items ?? []))
      .catch((e) => setError(e instanceof Error ? e.message : b.loadFail))
      .finally(() => setBusy(false))
  }
  useEffect(load, []) // eslint-disable-line react-hooks/exhaustive-deps

  const slice = pageSlice(rows, page, pageSize)

  return (
    <div>
      <PageHead title={b.title} desc={b.desc} />
      <div className="org-card">
        <div className="org-toolbar">
          <input className="org-input" type="number" placeholder={b.filterCustomer}
            value={customerId} onChange={(e) => { setCustomerId(e.target.value); setPage(1) }} />
          <span className="spacer" />
          <button className="org-btn org-btn-primary" onClick={() => setRunOpen(true)}>{b.run.btn}</button>
          <button className="org-btn" disabled={busy} onClick={load}>{t.pages.audit.refresh}</button>
        </div>
        {error ? <div className="org-error">{error}</div> : (
          <div className="org-table-wrap">
            <table className="org-table">
              <thead><tr>{b.columns.map((x) => <th key={x}>{x}</th>)}</tr></thead>
              <tbody>
                {slice.map((r) => (
                  <tr key={r.billId}>
                    <td>{r.billNo}</td>
                    <td>{r.customerName || `#${r.customerId}`}</td>
                    <td>{r.period}</td>
                    <td>{fmtFee(r.amount)}</td>
                    <td><StatusTag domain="bill" value={r.status} /></td>
                    <td>
                      <span className="org-act">
                        <button onClick={() => setDetail(r)}>{b.detail}</button>
                      </span>
                    </td>
                  </tr>
                ))}
                {!slice.length && <tr><td colSpan={6}><div className="org-empty">{b.empty}</div></td></tr>}
              </tbody>
            </table>
          </div>
        )}
        <div className="org-footer">
          <Pagination total={rows.length} page={page} pageSize={pageSize}
            onPage={setPage} onSize={setPageSize} {...pagerTexts(b)} />
        </div>
      </div>
      <InvoicePanel />
      <BillingRunModal open={runOpen} onClose={() => setRunOpen(false)} onDone={() => { load(); window.dispatchEvent(new CustomEvent(INVOICES_REFRESH)) }} />
      {detail && (
        <DetailDrawer
          title={b.detail}
          closeText={t.pages.company.cancel}
          onClose={() => setDetail(null)}
          items={[
            { k: b.columns[0], v: detail.billNo },
            { k: b.columns[1], v: detail.customerName || `#${detail.customerId}` },
            { k: b.columns[2], v: detail.period },
            { k: b.columns[3], v: fmtFee(detail.amount) },
            { k: b.columns[4], v: detail.status },
            { k: 'legalEntity', v: detail.legalEntityName || `#${detail.legalEntityId}` },
            { k: 'region', v: detail.regionName || `#${detail.regionId}` },
          ]}
        />
      )}
    </div>
  )
}

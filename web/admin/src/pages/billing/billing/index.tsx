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
      <div className="mb-4 rounded-md border border-[var(--shell-card-border)] bg-[var(--shell-card-bg)] shadow-[var(--shell-card-shadow)]">
        <div className="flex flex-wrap items-center gap-2 p-4">
          <input className="h-8 rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-2.5 text-[13px] text-[var(--shell-content-text)] outline-none placeholder:text-[var(--shell-input-placeholder)] focus:border-[var(--color-border-focus)]" type="number" placeholder={b.filterCustomer}
            value={customerId} onChange={(e) => { setCustomerId(e.target.value); setPage(1) }} />
          <span className="spacer" />
          <button className="h-8 cursor-pointer rounded-sm border-none bg-[var(--shell-fab-bg)] px-4 text-[13px] text-[var(--shell-fab-icon)] hover:bg-[var(--shell-fab-bg-hover)]" onClick={() => setRunOpen(true)}>{b.run.btn}</button>
          <button className="h-8 cursor-pointer rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-4 text-[13px] text-[var(--shell-content-text)] hover:border-[var(--color-border-hover)] hover:text-[var(--shell-heading)]" disabled={busy} onClick={load}>{t.pages.audit.refresh}</button>
        </div>
        {error ? <div className="mx-4 mb-3 rounded-sm border border-[color-mix(in_srgb,var(--color-danger)_25%,transparent)] bg-[color-mix(in_srgb,var(--color-danger)_8%,transparent)] px-3 py-2 text-[13px] text-[var(--color-danger)]">{error}</div> : (
          <div className="overflow-x-auto px-4 pb-4">
            <table className="w-full border-collapse text-[13px] text-[var(--shell-content-text)]">
              <thead className="h-11 px-3 text-left text-xs font-medium whitespace-nowrap border-b border-[var(--shell-side-border)] bg-[var(--shell-menu-hover-bg)] text-[var(--shell-group-title)]"><tr>{b.columns.map((x) => <th key={x} className="h-11 px-3 text-left text-xs font-medium whitespace-nowrap border-b border-[var(--shell-side-border)] bg-[var(--shell-menu-hover-bg)] text-[var(--shell-group-title)]">{x}</th>)}</tr></thead>
              <tbody>
                {slice.map((r) => (
                  <tr key={r.billId}>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{r.billNo}</td>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{r.customerName || `#${r.customerId}`}</td>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{r.period}</td>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{fmtFee(r.amount)}</td>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]"><StatusTag domain="bill" value={r.status} /></td>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">
                      <span className="inline-flex items-center">
                        <button onClick={() => setDetail(r)}>{b.detail}</button>
                      </span>
                    </td>
                  </tr>
                ))}
                {!slice.length && <tr><td colSpan={6} className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]"><div className="py-8 text-center text-[13px] text-[var(--shell-group-title)]">{b.empty}</div></td></tr>}
              </tbody>
            </table>
          </div>
        )}
        <div className="flex justify-end px-4 py-3 text-xs text-[var(--shell-group-title)]">
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

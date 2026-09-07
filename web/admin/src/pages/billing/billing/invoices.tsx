// 发票面板(TAX/AG-04):契约 GET /invoices + 作废/重开/人工回填(http_tax.go)。
// 入 billing 页(domain-map 裁定:发票无专用页);manual 通道回填税局票号。
import { useEffect, useState } from 'react'
import { toast } from 'sonner'
import { apiFetch } from '../../../api/client'
import { useT } from '../../../i18n'
import { Pagination } from '../../../components/Pagination'
import { ResourcePicker } from '../../../components/ResourcePicker'
import { searchCustomers } from '../../../api/pickers'
import { pagerTexts } from '../../org/shared'
import { fmtFee } from '../../../lib/format'
import type { InvoiceRow } from '../types'
import { INVOICES_REFRESH } from './run-modal'
import { pageSlice } from '../types'
import { TableStateRow } from '../../../components/business'

export function InvoicePanel() {
  const t = useT()
  const v = t.pages.billPage.invoice
  const [rows, setRows] = useState<InvoiceRow[]>([])
  const [error, setError] = useState('')
  const [customerId, setCustomerId] = useState('')
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(10)
  const [busy, setBusy] = useState(false)
  const [act, setAct] = useState<{ kind: 'void' | 'reissue' | 'backfill'; row: InvoiceRow } | null>(null)
  const [input, setInput] = useState('')
  const [actError, setActError] = useState('')

  const load = () => {
    setError('')
    apiFetch<{ items: InvoiceRow[] }>('/invoices', { query: { customerId: customerId || undefined } })
      .then((d) => setRows(d?.items ?? []))
      .catch((e) => setError(e instanceof Error ? e.message : v.loadFail))
  }
  useEffect(load, []) // eslint-disable-line react-hooks/exhaustive-deps
  useEffect(() => {
    const refresh = () => load()
    window.addEventListener(INVOICES_REFRESH, refresh)
    return () => window.removeEventListener(INVOICES_REFRESH, refresh)
  }, []) // eslint-disable-line react-hooks/exhaustive-deps

  const run = () => {
    if (!act || busy) return
    setActError(''); setBusy(true)
    const url = `/invoices/${act.row.id}/${act.kind === 'void' ? 'void' : act.kind === 'reissue' ? 'reissue' : 'tax-backfill'}`
    const body = act.kind === 'void' ? { reason: input.trim() } : act.kind === 'backfill' ? { taxNo: input.trim() } : undefined
    apiFetch(url, { method: 'POST', ...(body ? { body } : {}) })
      .then(() => {
        toast.success(act.kind === 'void' ? v.voidOk : act.kind === 'reissue' ? v.reissueOk : v.backfillOk)
        setAct(null); setInput(''); load()
      })
      .catch((e) => setActError(e instanceof Error ? e.message : v.actFail))
      .finally(() => setBusy(false))
  }

  const slice = pageSlice(rows, page, pageSize)
  const actText = act?.kind === 'void' ? v.voidConfirm : act?.kind === 'reissue' ? v.reissueConfirm : v.backfillTip

  return (
    <div className="mt-3 mb-4 rounded-md border border-[var(--shell-card-border)] bg-[var(--shell-card-bg)] shadow-[var(--shell-card-shadow)]">
      <div className="flex flex-wrap items-center gap-2 p-4">
        <strong>{v.title}</strong>
        <ResourcePicker
          value={customerId}
          onChange={(v) => { setCustomerId(v); setPage(1) }}
          search={searchCustomers}
          toOption={(c) => ({ value: String(c.id), label: `${c.name} · ${c.phone || c.customerCode}` })}
          ariaLabel={v.filterCustomer}
          emptyLabel={t.pages.pickers.common.all}
          searchPlaceholder={t.pages.pickers.common.placeholder}
          errorText={v.loadFail}
        />
        <span className="spacer" />
        <button className="h-8 cursor-pointer rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-4 text-[13px] text-[var(--shell-content-text)] hover:border-[var(--color-border-hover)] hover:text-[var(--shell-heading)]" disabled={busy} onClick={load}>{t.pages.audit.refresh}</button>
      </div>
      {error ? <div className="mx-4 mb-3 rounded-sm border border-[color-mix(in_srgb,var(--color-danger)_25%,transparent)] bg-[color-mix(in_srgb,var(--color-danger)_8%,transparent)] px-3 py-2 text-[13px] text-[var(--color-danger)]">{error}</div> : (
        <div className="overflow-x-auto px-4 pb-4">
          <table className="w-full border-collapse text-[13px] text-[var(--shell-content-text)]">
            <thead className="h-11 px-3 text-left text-xs font-medium whitespace-nowrap border-b border-[var(--shell-side-border)] bg-[var(--shell-menu-hover-bg)] text-[var(--shell-group-title)]"><tr>{v.columns.map((x) => <th key={x} className="h-11 px-3 text-left text-xs font-medium whitespace-nowrap border-b border-[var(--shell-side-border)] bg-[var(--shell-menu-hover-bg)] text-[var(--shell-group-title)]">{x}</th>)}</tr></thead>
            <tbody>
              {slice.map((r) => (
                <tr key={r.id}>
                  <td className="break-all rounded-sm bg-black/5 px-2 py-1.5 font-mono text-xs">{r.invoiceNo}</td>
                  <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{r.customerName || `#${r.customerId}`}</td>
                  <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{fmtFee(r.totalAmount)}</td>
                  <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{r.status}</td>
                  <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{r.taxJurisdiction || '—'}</td>
                  <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{r.taxNo || r.taxStatus}</td>
                  <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">
                    <span className="inline-flex items-center">
                      {r.status === 'ISSUED' && <button onClick={() => { setAct({ kind: 'void', row: r }); setInput(''); setActError('') }}>{v.voidBtn}</button>}
                      {r.status === 'ISSUED' && r.taxStatus !== 'ISSUED' && <button onClick={() => { setAct({ kind: 'backfill', row: r }); setInput(''); setActError('') }}>{v.backfillBtn}</button>}
                      {r.status === 'VOIDED' && <button onClick={() => { setAct({ kind: 'reissue', row: r }); setActError('') }}>{v.reissueBtn}</button>}
                    </span>
                  </td>
                </tr>
              ))}
              {!slice.length && <TableStateRow colSpan={7} loading={busy} text={v.empty} />}
            </tbody>
          </table>
        </div>
      )}
      <div className="flex justify-end px-4 py-3 text-xs text-[var(--shell-group-title)]">
        <Pagination total={rows.length} page={page} pageSize={pageSize}
          onPage={setPage} onSize={setPageSize} {...pagerTexts(t.pages.billPage)} />
      </div>

      {act && (
        <div className="fixed inset-0 z-page-modal flex items-center justify-center bg-black/45">
          <div className="w-90 rounded-md bg-[var(--shell-card-bg)] p-5">
            <p>{actText}</p>
            {act.kind !== 'reissue' && (
              <input className="h-8 rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-2.5 text-[13px] text-[var(--shell-content-text)] outline-none placeholder:text-[var(--shell-input-placeholder)] focus:border-[var(--color-border-focus)]" value={input} autoFocus
                placeholder={act.kind === 'void' ? v.voidReasonPh : v.taxNoPh}
                onChange={(e) => setInput(e.target.value)} />
            )}
            {actError && <p className="mt-2 rounded-sm border border-[color-mix(in_srgb,var(--color-danger)_25%,transparent)] bg-[color-mix(in_srgb,var(--color-danger)_8%,transparent)] px-3 py-2 text-[13px] text-[var(--color-danger)]">{actError}</p>}
            <div className="mt-4 flex justify-end gap-2">
              <button className="h-8 cursor-pointer rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-4 text-[13px] text-[var(--shell-content-text)] hover:border-[var(--color-border-hover)] hover:text-[var(--shell-heading)]" onClick={() => setAct(null)}>{t.pages.company.cancel}</button>
              <button className="h-8 cursor-pointer rounded-sm border-none bg-[var(--shell-fab-bg)] px-4 text-[13px] text-[var(--shell-fab-icon)] hover:bg-[var(--shell-fab-bg-hover)]" disabled={busy || (act.kind !== 'reissue' && !input.trim())} onClick={run}>
                {busy ? t.pages.account.submitting : v.confirm}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}

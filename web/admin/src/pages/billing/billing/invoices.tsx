// 发票面板(TAX/AG-04):契约 GET /invoices + 作废/重开/人工回填(http_tax.go)。
// 入 billing 页(domain-map 裁定:发票无专用页);manual 通道回填税局票号。
import { useEffect, useState } from 'react'
import { apiFetch } from '../../../api/client'
import { useT } from '../../../i18n'
import { Pagination } from '../../../components/Pagination'
import { pagerTexts } from '../../org/shared'
import { fmtFee } from '../../../lib/format'
import type { InvoiceRow } from '../types'
import { INVOICES_REFRESH } from './run-modal'
import { pageSlice } from '../types'

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
      .then(() => { setAct(null); setInput(''); load() })
      .catch((e) => setActError(e instanceof Error ? e.message : v.actFail))
      .finally(() => setBusy(false))
  }

  const slice = pageSlice(rows, page, pageSize)
  const actText = act?.kind === 'void' ? v.voidConfirm : act?.kind === 'reissue' ? v.reissueConfirm : v.backfillTip

  return (
    <div className="org-card" style={{ marginTop: 12 }}>
      <div className="org-toolbar">
        <strong>{v.title}</strong>
        <input className="org-input" type="number" placeholder={v.filterCustomer}
          value={customerId} onChange={(e) => { setCustomerId(e.target.value); setPage(1) }} />
        <span className="spacer" />
        <button className="org-btn" disabled={busy} onClick={load}>{t.pages.audit.refresh}</button>
      </div>
      {error ? <div className="org-error">{error}</div> : (
        <div className="org-table-wrap">
          <table className="org-table">
            <thead><tr>{v.columns.map((x) => <th key={x}>{x}</th>)}</tr></thead>
            <tbody>
              {slice.map((r) => (
                <tr key={r.id}>
                  <td className="mono">{r.invoiceNo}</td>
                  <td>{r.customerName || `#${r.customerId}`}</td>
                  <td>{fmtFee(r.totalAmount)}</td>
                  <td>{r.status}</td>
                  <td>{r.taxJurisdiction || '—'}</td>
                  <td>{r.taxNo || r.taxStatus}</td>
                  <td>
                    <span className="org-act">
                      {r.status === 'ISSUED' && <button onClick={() => { setAct({ kind: 'void', row: r }); setInput(''); setActError('') }}>{v.voidBtn}</button>}
                      {r.status === 'ISSUED' && r.taxStatus !== 'ISSUED' && <button onClick={() => { setAct({ kind: 'backfill', row: r }); setInput(''); setActError('') }}>{v.backfillBtn}</button>}
                      {r.status === 'VOIDED' && <button onClick={() => { setAct({ kind: 'reissue', row: r }); setActError('') }}>{v.reissueBtn}</button>}
                    </span>
                  </td>
                </tr>
              ))}
              {!slice.length && <tr><td colSpan={7}><div className="org-empty">{v.empty}</div></td></tr>}
            </tbody>
          </table>
        </div>
      )}
      <div className="org-footer">
        <Pagination total={rows.length} page={page} pageSize={pageSize}
          onPage={setPage} onSize={setPageSize} {...pagerTexts(t.pages.billPage)} />
      </div>

      {act && (
        <div className="fixed inset-0 z-[120] flex items-center justify-center bg-black/45">
          <div className="w-90 rounded-md bg-[var(--shell-card-bg)] p-5">
            <p>{actText}</p>
            {act.kind !== 'reissue' && (
              <input className="org-input" value={input} autoFocus
                placeholder={act.kind === 'void' ? v.voidReasonPh : v.taxNoPh}
                onChange={(e) => setInput(e.target.value)} />
            )}
            {actError && <p className="org-error" style={{ margin: '8px 0 0' }}>{actError}</p>}
            <div className="mt-4 flex justify-end gap-2">
              <button className="org-btn" onClick={() => setAct(null)}>{t.pages.company.cancel}</button>
              <button className="org-btn org-btn-primary" disabled={busy || (act.kind !== 'reissue' && !input.trim())} onClick={run}>
                {busy ? t.pages.account.submitting : v.confirm}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}

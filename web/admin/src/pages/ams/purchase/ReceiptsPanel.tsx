// 入库单区:GET /procurement/receipts 列表(采购单页内独立卡片);DRAFT 行提供驳回入口。
import { useCallback, useEffect, useState } from 'react'
import { apiFetch } from '../../../api/client'
import { useT } from '../../../i18n'
import { StatusTag } from '../../../components/StatusTag'
import { TableStateRow } from '../../../components/business'
import { type ReceiptRow } from '../types'
import { canRejectReceipt } from './purchaseLogic'
import { RejectReceiptDrawer } from './RejectReceiptDrawer'

const th = 'h-9 px-2 text-left text-xs font-medium whitespace-nowrap border-b border-[var(--shell-side-border)] bg-[var(--shell-menu-hover-bg)] text-[var(--shell-group-title)]'
const td = 'h-9 px-2 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]'
const smallBtn = 'h-7 cursor-pointer rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-2 text-[12px] text-[var(--color-danger)] hover:border-[var(--color-border-hover)] disabled:cursor-not-allowed disabled:opacity-50'
const errBanner = 'rounded-sm border border-[color-mix(in_srgb,var(--color-danger)_25%,transparent)] bg-[color-mix(in_srgb,var(--color-danger)_8%,transparent)] px-3 py-2 text-[13px] text-[var(--color-danger)]'

export function ReceiptsPanel() {
  const t = useT()
  const d = t.pages.purchasePage
  const [rows, setRows] = useState<ReceiptRow[]>([])
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)
  const [rejecting, setRejecting] = useState<ReceiptRow | null>(null)

  const load = useCallback(() => {
    setError('')
    setBusy(true)
    apiFetch<{ items: ReceiptRow[] }>('/procurement/receipts')
      .then((x) => setRows(x?.items ?? []))
      .catch((e) => setError(e instanceof Error ? e.message : d.loadFail))
      .finally(() => setBusy(false))
  }, []) // eslint-disable-line react-hooks/exhaustive-deps
  useEffect(load, [load])

  const cols = [d.colReceiptNo, d.colOrderNo, d.colStatus, d.colReceivedAt, d.colActions]
  return (
    <div className="mb-4 rounded-md border border-[var(--shell-card-border)] bg-[var(--shell-card-bg)] shadow-[var(--shell-card-shadow)]">
      <div className="flex flex-wrap items-center gap-2 p-4">
        <div className="text-[13px] font-medium text-[var(--shell-heading)]">{d.receiptsTitle}</div>
        <span className="spacer" />
        <button type="button" onClick={load}
          className="h-8 cursor-pointer rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-4 text-[13px] text-[var(--shell-content-text)] hover:border-[var(--color-border-hover)] hover:text-[var(--shell-heading)]">
          {t.pages.audit.refresh}
        </button>
      </div>
      {error ? (
        <div className="mx-4 mb-3"><div className={errBanner}>{error}</div></div>
      ) : (
        <div className="overflow-x-auto px-4 pb-4">
          <table className="w-full border-collapse text-[13px] text-[var(--shell-content-text)]">
            <thead><tr>{cols.map((x) => <th key={x} className={th}>{x}</th>)}</tr></thead>
            <tbody>
              {rows.map((r) => (
                <tr key={r.id}>
                  <td className={td + ' font-mono'}>{r.receiptNo || '#' + r.id}</td>
                  <td className={td + ' font-mono'}>{r.orderNo || '#' + r.orderId}</td>
                  <td className={td}><StatusTag domain="receipt" value={r.status} /></td>
                  <td className={td}>{r.receivedAt || '—'}</td>
                  <td className={td}>
                    <span className="inline-flex items-center gap-2">
                      {canRejectReceipt(r.status) && (
                        <button className={smallBtn} disabled={busy} onClick={() => setRejecting(r)}>{d.reject}</button>
                      )}
                      {!canRejectReceipt(r.status) && <span className="text-[var(--shell-group-title)]">—</span>}
                    </span>
                  </td>
                </tr>
              ))}
              {!rows.length && <TableStateRow colSpan={5} loading={busy} text={d.receiptsEmpty} />}
            </tbody>
          </table>
        </div>
      )}
      {rejecting && (
        <RejectReceiptDrawer receipt={rejecting} onClose={() => setRejecting(null)} onSaved={load} />
      )}
    </div>
  )
}

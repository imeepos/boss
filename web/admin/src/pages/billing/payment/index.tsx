// 缴费管理页:列名按 fields.md 裁剪;契约 GET /payments(billId 过滤)。
// 柜面收款入口(纪要 2026-08-28):menu:payment:cash 权限持有者可登记现金/柜面收款。
import { useEffect, useState } from 'react'
import { toast } from 'sonner'
import { apiFetch } from '../../../api/client'
import { useT } from '../../../i18n'
import { PageHead, pagerTexts } from '../../org/shared'
import { StatusTag } from '../../../components/StatusTag'
import { Pagination } from '../../../components/Pagination'
import { pageSlice, type PaymentRow } from '../types'
import { fmtFee } from '../../../lib/format'
import { TableStateRow, ErrorBanner } from '../../../components/business'
import { useConfirm } from '../../../components/ConfirmDialog'
import { useProfile } from '../../../layouts/profile'
import { CounterPaymentForm } from './CounterPaymentForm'

export default function PaymentPage() {
  const t = useT()
  const p = t.pages.payment
  const profile = useProfile()
  const canCollect = (profile.permissionCodes ?? []).includes('menu:payment:cash')
  const [rows, setRows] = useState<PaymentRow[]>([])
  const [error, setError] = useState('')
  const confirm = useConfirm()
  const [notice, setNotice] = useState('')
  const [formOpen, setFormOpen] = useState(false)
  const [billId, setBillId] = useState('')
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(10)
  const [busy, setBusy] = useState(false)

  const load = () => {
    setError('')
    setBusy(true)
    apiFetch<{ items: PaymentRow[] }>('/payments', {
      query: { billId: billId || undefined },
    })
      .then((d) => setRows(d?.items ?? []))
      .catch((e) => setError(e instanceof Error ? e.message : p.loadFail))
      .finally(() => setBusy(false))
  }
  useEffect(load, []) // eslint-disable-line react-hooks/exhaustive-deps

  // 全额退款(000112):SUCCESS 流水行内入口;REFUNDED 留痕终态不可再退。
  const refund = async (id: number) => {
    if (busy) return
    if (!(await confirm(p.refundConfirm, { danger: true }))) return
    setBusy(true); setError('')
    try {
      await apiFetch(`/payments/${id}/refund`, { method: 'POST' })
      toast.success(p.refundOk)
      load()
    } catch (e) {
      setError(e instanceof Error ? e.message : p.loadFail)
      setBusy(false)
    }
  }

  const slice = pageSlice(rows, page, pageSize)

  return (
    <div>
      <PageHead title={p.title} desc={p.desc} />
      {notice && <div className="mb-4 rounded-md border border-[color-mix(in_srgb,var(--color-success)_25%,transparent)] bg-[color-mix(in_srgb,var(--color-success)_8%,transparent)] px-4 py-2 text-[13px] text-[var(--color-success)]">{notice}</div>}
      <div className="mb-4 rounded-md border border-[var(--shell-card-border)] bg-[var(--shell-card-bg)] shadow-[var(--shell-card-shadow)]">
        <div className="flex flex-wrap items-center gap-2 p-4">
          <input className="h-8 rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-2.5 text-[13px] text-[var(--shell-content-text)] outline-none placeholder:text-[var(--shell-input-placeholder)] focus:border-[var(--color-border-focus)]" type="number" placeholder={p.filterBill}
            value={billId} onChange={(e) => { setBillId(e.target.value); setPage(1) }} />
          <span className="spacer" />
          {canCollect && (
            <button className="h-8 cursor-pointer rounded-sm bg-[var(--color-brand-bg)] px-4 text-[13px] text-white hover:opacity-90" onClick={() => { setNotice(''); setFormOpen(true) }}>{p.addBtn}</button>
          )}
          <button className="h-8 cursor-pointer rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-4 text-[13px] text-[var(--shell-content-text)] hover:border-[var(--color-border-hover)] hover:text-[var(--shell-heading)]" disabled={busy} onClick={load}>{t.pages.audit.refresh}</button>
        </div>
        {error ? <ErrorBanner message={error} /> : (
          <div className="overflow-x-auto px-4 pb-4">
            <table className="w-full border-collapse text-[13px] text-[var(--shell-content-text)]">
              <thead className="h-11 px-3 text-left text-xs font-medium whitespace-nowrap border-b border-[var(--shell-side-border)] bg-[var(--shell-menu-hover-bg)] text-[var(--shell-group-title)]"><tr>{p.columns.map((x) => <th key={x} className="h-11 px-3 text-left text-xs font-medium whitespace-nowrap border-b border-[var(--shell-side-border)] bg-[var(--shell-menu-hover-bg)] text-[var(--shell-group-title)]">{x}</th>)}</tr></thead>
              <tbody>
                {slice.map((r) => (
                  <tr key={r.id}>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{r.payNo}</td>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">#{r.billId}</td>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{fmtFee(r.amount)}</td>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{p.methods[r.method] ?? r.method}</td>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]"><StatusTag domain="payment" value={r.status} /></td>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{r.siteName || '-'}</td>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{r.operatorName || '-'}</td>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">
                      {r.status === 'SUCCESS' && canCollect && <button className="px-1 text-xs text-[var(--color-danger)] bg-none border-none cursor-pointer hover:underline" disabled={busy} onClick={() => refund(r.id)}>{p.refundBtn}</button>}
                      {r.status !== 'SUCCESS' && <span className="text-[var(--shell-group-title)]">—</span>}
                    </td>
                  </tr>
                ))}
                {!slice.length && <TableStateRow colSpan={8} loading={busy} text={p.empty} />}
              </tbody>
            </table>
          </div>
        )}
        <div className="flex justify-end px-4 py-3 text-xs text-[var(--shell-group-title)]">
          <Pagination total={rows.length} page={page} pageSize={pageSize}
            onPage={setPage} onSize={setPageSize} {...pagerTexts(p)} />
        </div>
      </div>
      {formOpen && (
        <CounterPaymentForm
          onClose={() => setFormOpen(false)}
          onDone={(payNo) => { setFormOpen(false); setNotice(p.success.replace('{payNo}', payNo)); load() }}
        />
      )}
    </div>
  )
}

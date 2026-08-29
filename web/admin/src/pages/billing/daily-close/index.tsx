// 柜台日结页(纪要 2026-08-28-柜面现金收款,周敏口径):
// 当日 cash 流水按网点+操作员汇总,收入/退款分列(退款按流水发生日归属);
// 实点金额当日回填(perm menu:payment:cash),不平由后端落 [paycheck] DIFF 日志;
// 明细区逐笔下钻含 REFUNDED 凭证与退款原因。
import { useEffect, useState } from 'react'
import { apiFetch } from '../../../api/client'
import { useT } from '../../../i18n'
import { PageHead, pagerTexts } from '../../org/shared'
import { StatusTag } from '../../../components/StatusTag'
import { Pagination } from '../../../components/Pagination'
import { pageSlice, type DailyCloseRow, type PaymentRow } from '../types'
import { fmtFee } from '../../../lib/format'
import { TableStateRow } from '../../../components/business'
import { useProfile } from '../../../layouts/profile'

const today = () => new Date().toISOString().slice(0, 10)
const inputCls = 'h-8 rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-2.5 text-[13px] text-[var(--shell-content-text)] outline-none placeholder:text-[var(--shell-input-placeholder)] focus:border-[var(--color-border-focus)]'
const thCls = 'h-11 px-3 text-left text-xs font-medium whitespace-nowrap border-b border-[var(--shell-side-border)] bg-[var(--shell-menu-hover-bg)] text-[var(--shell-group-title)]'
const tdCls = 'h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]'

export default function DailyClosePage() {
  const t = useT()
  const p = t.pages.dailyClose
  const profile = useProfile()
  const canCollect = (profile.permissionCodes ?? []).includes('menu:payment:cash')
  const [date, setDate] = useState(today())
  const [rows, setRows] = useState<DailyCloseRow[]>([])
  const [items, setItems] = useState<PaymentRow[]>([])
  const [error, setError] = useState('')
  const [notice, setNotice] = useState('')
  const [counted, setCounted] = useState<Record<string, string>>({})
  const [savingKey, setSavingKey] = useState('')
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(10)
  const [busy, setBusy] = useState(false)

  const load = () => {
    setError('')
    setBusy(true)
    Promise.all([
      apiFetch<{ items: DailyCloseRow[] }>('/daily-closings/summary', { query: { date } }),
      apiFetch<{ items: PaymentRow[] }>('/daily-closings/items', { query: { date } }),
    ])
      .then(([s, i]) => { setRows(s?.items ?? []); setItems(i?.items ?? []) })
      .catch((e) => setError(e instanceof Error ? e.message : p.loadFail))
      .finally(() => setBusy(false))
  }
  useEffect(load, [date]) // eslint-disable-line react-hooks/exhaustive-deps

  const key = (r: DailyCloseRow) => `${r.siteName}\u0000${r.operatorName}`
  const saveCounted = async (r: DailyCloseRow) => {
    const v = Number(counted[key(r)])
    if (!Number.isFinite(v) || v < 0) return
    setSavingKey(key(r))
    setNotice('')
    try {
      const d = await apiFetch<{ closing: { balanced: boolean; diffAmount: number } }>('/daily-closings', {
        method: 'POST',
        body: { date, siteName: r.siteName, operatorName: r.operatorName, countedAmount: v },
      })
      setNotice(d?.closing.balanced ? p.saved : p.diff.replace('{diff}', String(d?.closing.diffAmount ?? '')))
      load()
    } catch (e) {
      setError(e instanceof Error ? e.message : p.loadFail)
    } finally {
      setSavingKey('')
    }
  }

  const slice = pageSlice(items, page, pageSize)

  return (
    <div>
      <PageHead title={p.title} desc={p.desc} />
      {notice && <div className="mb-4 rounded-md border border-[color-mix(in_srgb,var(--color-success)_25%,transparent)] bg-[color-mix(in_srgb,var(--color-success)_8%,transparent)] px-4 py-2 text-[13px] text-[var(--color-success)]">{notice}</div>}
      <div className="mb-4 rounded-md border border-[var(--shell-card-border)] bg-[var(--shell-card-bg)] shadow-[var(--shell-card-shadow)]">
        <div className="flex flex-wrap items-center gap-2 p-4">
          <label className="text-[13px] text-[var(--shell-content-text)]">{p.date}</label>
          <input className={inputCls} type="date" value={date} onChange={(e) => setDate(e.target.value)} />
          <span className="spacer" />
          <button className="h-8 cursor-pointer rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-4 text-[13px] text-[var(--shell-content-text)] hover:border-[var(--color-border-hover)] hover:text-[var(--shell-heading)]" disabled={busy} onClick={load}>{t.pages.audit.refresh}</button>
        </div>
        {error ? <div className="mx-4 mb-3 rounded-sm border border-[color-mix(in_srgb,var(--color-danger)_25%,transparent)] bg-[color-mix(in_srgb,var(--color-danger)_8%,transparent)] px-3 py-2 text-[13px] text-[var(--color-danger)]">{error}</div> : (
          <div className="overflow-x-auto px-4 pb-4">
            <table className="w-full border-collapse text-[13px] text-[var(--shell-content-text)]">
              <thead><tr>{p.columns.map((x) => <th key={x} className={thCls}>{x}</th>)}</tr></thead>
              <tbody>
                {rows.map((r) => {
                  const k = key(r)
                  // 写侧本人约束:非 sysadmin 只能回填自己名下的日结(后端同强校验)。
                  const canFillRow = canCollect && (profile.roleCode === 'sysadmin' || r.operatorName === profile.username)
                  return (
                    <tr key={k}>
                      <td className={tdCls}>{r.siteName}</td>
                      <td className={tdCls}>{r.operatorName}</td>
                      <td className={tdCls}>{fmtFee(r.inAmount)}</td>
                      <td className={tdCls}>{fmtFee(r.refundAmount)}</td>
                      <td className={tdCls}>{fmtFee(r.netAmount)}</td>
                      <td className={tdCls}>
                        {canFillRow ? (
                          <input className={`${inputCls} w-28`} type="number" min="0" step="0.01" placeholder={p.countedPlaceholder}
                            defaultValue={r.countedAmount} onChange={(e) => setCounted((m) => ({ ...m, [k]: e.target.value }))} />
                        ) : (r.countedAmount !== undefined ? fmtFee(r.countedAmount) : p.noCounted)}
                      </td>
                      <td className={tdCls}>
                        {r.countedAmount !== undefined
                          ? <span className="text-[var(--color-success)]">{p.balanced}</span>
                          : canFillRow && (
                            <button className="h-7 cursor-pointer rounded-sm bg-[var(--color-brand-bg)] px-3 text-xs text-white hover:opacity-90 disabled:cursor-not-allowed disabled:opacity-60"
                              disabled={savingKey === k || counted[k] === undefined} onClick={() => saveCounted(r)}>
                              {savingKey === k ? p.saving : p.save}
                            </button>
                          )}
                      </td>
                    </tr>
                  )
                })}
                {!rows.length && <TableStateRow colSpan={p.columns.length} loading={busy} text={p.empty} />}
              </tbody>
            </table>
          </div>
        )}
      </div>
      <div className="rounded-md border border-[var(--shell-card-border)] bg-[var(--shell-card-bg)] shadow-[var(--shell-card-shadow)]">
        <div className="p-4 text-[14px] font-medium text-[var(--shell-heading)]">{p.detailTitle}</div>
        <div className="overflow-x-auto px-4 pb-4">
          <table className="w-full border-collapse text-[13px] text-[var(--shell-content-text)]">
            <thead><tr>{p.detailColumns.map((x) => <th key={x} className={thCls}>{x}</th>)}</tr></thead>
            <tbody>
              {slice.map((r) => (
                <tr key={r.id}>
                  <td className={tdCls}>{r.payNo}</td>
                  <td className={tdCls}>{r.billId ? `#${r.billId}` : '-'}</td>
                  <td className={tdCls}>{fmtFee(r.amount)}</td>
                  <td className={tdCls}><StatusTag domain="payment" value={r.status} /></td>
                  <td className={tdCls}>{r.refundReason || '-'}</td>
                  <td className={tdCls}>{r.siteName || '-'}</td>
                  <td className={tdCls}>{r.operatorName || '-'}</td>
                </tr>
              ))}
              {!slice.length && <TableStateRow colSpan={p.detailColumns.length} loading={busy} text={p.empty} />}
            </tbody>
          </table>
        </div>
        <div className="flex justify-end px-4 py-3 text-xs text-[var(--shell-group-title)]">
          <Pagination total={items.length} page={page} pageSize={pageSize}
            onPage={setPage} onSize={setPageSize} {...pagerTexts(t.pages.payment)} />
        </div>
      </div>
    </div>
  )
}

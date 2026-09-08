// 柜台日结页(纪要 2026-08-28-柜面现金收款,周敏口径):
// 当日 cash 流水按网点+操作员汇总,收入/退款分列(退款按流水发生日归属);
// 实点金额当日回填(perm menu:payment:cash,非 sysadmin 仅本人行可写,后端同强校验),
// 不平由后端落 [paycheck] DIFF 日志;明细区逐笔下钻含 REFUNDED 凭证与退款原因,账单列人读化。
import { TableStateRow, ErrorBanner, ToolbarButton } from '../../../components/business'
import { useEffect, useMemo, useState } from 'react'
import { toast } from 'sonner'
import { apiFetch } from '../../../api/client'
import { useT } from '../../../i18n'
import { PageHead, pagerTexts } from '../../org/shared'
import { StatusTag } from '../../../components/StatusTag'
import { Pagination } from '../../../components/Pagination'
import { Card, CardFooter } from '../../../components/ui/card'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '../../../components/ui/table'
import { pageSlice, type BillRow, type DailyCloseRow, type PaymentRow } from '../types'
import { BillRef } from '../BillRef'
import { fmtFee } from '../../../lib/format'
import { useProfile } from '../../../layouts/profile'

const today = () => new Date().toISOString().slice(0, 10)
const inputCls = 'h-8 rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-2.5 text-[13px] text-[var(--shell-content-text)] outline-none placeholder:text-[var(--shell-input-placeholder)] focus:border-[var(--color-border-focus)]'

export default function DailyClosePage() {
  const t = useT()
  const p = t.pages.dailyClose
  const profile = useProfile()
  const canCollect = (profile.permissionCodes ?? []).includes('menu:payment:cash')
  const [date, setDate] = useState(today())
  const [rows, setRows] = useState<DailyCloseRow[]>([])
  const [items, setItems] = useState<PaymentRow[]>([])
  const [bills, setBills] = useState<BillRow[]>([])
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

  // 账单人读映射:/bills 全量拉(既有先例),失败静默降级 #id,不阻断日结只读主流程。
  useEffect(() => {
    apiFetch<{ items: BillRow[] }>('/bills', {})
      .then((d) => setBills(d?.items ?? []))
      .catch(() => setBills([]))
  }, [])
  const billMap = useMemo(() => new Map(bills.map((b) => [b.billId, b])), [bills])

  const key = (r: DailyCloseRow) => `${r.siteName}\u0000${r.operatorName}`
  const saveCounted = async (r: DailyCloseRow) => {
    const v = Number(counted[key(r)])
    if (!Number.isFinite(v) || v < 0) { setError(p.countedInvalid); return }
    setSavingKey(key(r))
    setNotice('')
    try {
      const d = await apiFetch<{ closing: { balanced: boolean; diffAmount: number } }>('/daily-closings', {
        method: 'POST',
        body: { date, siteName: r.siteName, operatorName: r.operatorName, countedAmount: v },
      })
      const msg = d?.closing.balanced ? p.saved : p.diff.replace('{diff}', String(d?.closing.diffAmount ?? ''))
      toast.success(msg)
      setNotice(msg)
      load()
    } catch (e) {
      toast.error(e instanceof Error ? e.message : p.loadFail)
      setError(e instanceof Error ? e.message : p.loadFail)
    } finally {
      setSavingKey('')
    }
  }

  const totals = useMemo(() => rows.reduce((acc, r) => ({
    in: acc.in + (r.inAmount ?? 0),
    refund: acc.refund + (r.refundAmount ?? 0),
    net: acc.net + (r.netAmount ?? 0),
    counted: acc.counted + (r.countedAmount ?? 0),
  }), { in: 0, refund: 0, net: 0, counted: 0 }), [rows])

  const slice = pageSlice(items, page, pageSize)

  return (
    <div>
      <PageHead title={p.title} desc={p.desc} />
      {notice && <div className="mb-4 rounded-md border border-[color-mix(in_srgb,var(--color-success)_25%,transparent)] bg-[color-mix(in_srgb,var(--color-success)_8%,transparent)] px-4 py-2 text-[13px] text-[var(--color-success)]">{notice}</div>}
      <Card>
        <div className="flex flex-wrap items-center gap-2 p-4">
          <label className="text-[13px] text-[var(--shell-content-text)]">{p.date}</label>
          <input className={inputCls} type="date" value={date} onChange={(e) => setDate(e.target.value)} />
          <span className="spacer" />
          <ToolbarButton disabled={busy} onClick={load}>{t.pages.audit.refresh}</ToolbarButton>
        </div>
        {error ? <ErrorBanner message={error} /> : (
          <div className="px-4 pb-4">
            <Table>
              <TableHeader><TableRow>{p.columns.map((x) => <TableHead key={x}>{x}</TableHead>)}</TableRow></TableHeader>
              <TableBody>
                {rows.map((r) => {
                  const k = key(r)
                  // 写侧本人约束:非 sysadmin 只能回填自己名下的日结(后端同强校验)。
                  const canFillRow = canCollect && (profile.roleCode === 'sysadmin' || r.operatorName === profile.username)
                  return (
                    <TableRow key={k}>
                      <TableCell>{r.siteName}</TableCell>
                      <TableCell>{r.operatorName}</TableCell>
                      <TableCell>{fmtFee(r.inAmount)}</TableCell>
                      <TableCell>{fmtFee(r.refundAmount)}</TableCell>
                      <TableCell>{fmtFee(r.netAmount)}</TableCell>
                      <TableCell>
                        {canFillRow && r.countedAmount === undefined ? (
                          <input className={`${inputCls} w-28`} type="number" min="0" step="0.01" placeholder={p.countedPlaceholder}
                            onChange={(e) => setCounted((m) => ({ ...m, [k]: e.target.value }))} />
                        ) : (r.countedAmount !== undefined ? fmtFee(r.countedAmount) : p.noCounted)}
                      </TableCell>
                      <TableCell>
                        {r.countedAmount !== undefined
                          ? <span className="text-[var(--color-success)]">{p.balanced}</span>
                          : canFillRow && (
                            <button className="h-7 cursor-pointer rounded-sm bg-[var(--color-brand-bg)] px-3 text-xs text-white hover:opacity-90 disabled:cursor-not-allowed disabled:opacity-60"
                              disabled={savingKey === k || counted[k] === undefined} onClick={() => saveCounted(r)}>
                              {savingKey === k ? p.saving : p.save}
                            </button>
                          )}
                      </TableCell>
                    </TableRow>
                  )
                })}
                {rows.length > 0 && (
                  <TableRow>
                    <TableCell className="font-medium text-[var(--shell-heading)]">{p.total}</TableCell>
                    <TableCell />
                    <TableCell className="font-medium">{fmtFee(totals.in)}</TableCell>
                    <TableCell className="font-medium">{fmtFee(totals.refund)}</TableCell>
                    <TableCell className="font-medium">{fmtFee(totals.net)}</TableCell>
                    <TableCell className="font-medium">{fmtFee(totals.counted)}</TableCell>
                    <TableCell />
                  </TableRow>
                )}
                {!rows.length && <TableStateRow colSpan={p.columns.length} loading={busy} text={p.empty} />}
              </TableBody>
            </Table>
          </div>
        )}
      </Card>
      <Card className="mb-0">
        <div className="p-4 text-[14px] font-medium text-[var(--shell-heading)]">{p.detailTitle}</div>
        <div className="px-4 pb-4">
          <Table>
            <TableHeader><TableRow>{p.detailColumns.map((x) => <TableHead key={x}>{x}</TableHead>)}</TableRow></TableHeader>
            <TableBody>
              {slice.map((r) => (
                <TableRow key={r.id}>
                  <TableCell className="font-mono">{r.payNo}</TableCell>
                  <TableCell>{r.billId ? <BillRef billId={r.billId} map={billMap} noBillText={t.pages.payment.noBill} /> : '-'}</TableCell>
                  <TableCell>{fmtFee(r.amount)}</TableCell>
                  <TableCell><StatusTag domain="payment" value={r.status} /></TableCell>
                  <TableCell>{r.refundReason || '-'}</TableCell>
                  <TableCell>{r.siteName || '-'}</TableCell>
                  <TableCell>{r.operatorName || '-'}</TableCell>
                </TableRow>
              ))}
              {!slice.length && <TableStateRow colSpan={p.detailColumns.length} loading={busy} text={p.empty} />}
            </TableBody>
          </Table>
        </div>
        <CardFooter>
          <Pagination total={items.length} page={page} pageSize={pageSize}
            onPage={setPage} onSize={setPageSize} {...pagerTexts(p)} />
        </CardFooter>
      </Card>
    </div>
  )
}

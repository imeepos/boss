// 缴费管理页:列名按 fields.md 裁剪;契约 GET /payments(billId 过滤)。
// 柜面收款入口(纪要 2026-08-28):menu:payment:cash 权限持有者可登记现金/柜面收款。
// 账单列人读化:/bills 全量(既有先例)建 billId→账单号+客户名 映射,缺失回退 #id。
// 全额退款走 RefundDialog(后端 reason 必填,旧空体提交必 422)。
import { useEffect, useMemo, useState } from 'react'
import { toast } from 'sonner'
import { apiFetch } from '../../../api/client'
import { useT } from '../../../i18n'
import { PageHead, pagerTexts } from '../../org/shared'
import { StatusTag } from '../../../components/StatusTag'
import { Pagination } from '../../../components/Pagination'
import { Card, CardFooter } from '../../../components/ui/card'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '../../../components/ui/table'
import { pageSlice, type BillRow, type PaymentRow } from '../types'
import { BillRef } from '../BillRef'
import { SimplePicker } from '../../../components/pickers/SimplePicker'
import { fmtFee } from '../../../lib/format'
import { TableStateRow, ErrorBanner, ActionLink, ToolbarButton } from '../../../components/business'
import { useProfile } from '../../../layouts/profile'
import { CounterPaymentForm } from './CounterPaymentForm'
import { RefundDialog } from './RefundDialog'

export default function PaymentPage() {
  const t = useT()
  const p = t.pages.payment
  const profile = useProfile()
  const canCollect = (profile.permissionCodes ?? []).includes('menu:payment:cash')
  const [rows, setRows] = useState<PaymentRow[]>([])
  const [error, setError] = useState('')
  const [notice, setNotice] = useState('')
  const [formOpen, setFormOpen] = useState(false)
  const [refund, setRefund] = useState<PaymentRow | null>(null)
  const [billId, setBillId] = useState('')
  const [bills, setBills] = useState<BillRow[]>([])
  const [billsErr, setBillsErr] = useState(false)
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

  // 账单过滤选项数据源:GET /bills(items 信封,无 keyword/分页参数;全量拉先例 billing/billing)。
  const loadBills = () => {
    setBillsErr(false)
    apiFetch<{ items: BillRow[] }>('/bills', {})
      .then((d) => setBills(d?.items ?? []))
      .catch(() => setBillsErr(true))
  }
  useEffect(loadBills, []) // eslint-disable-line react-hooks/exhaustive-deps
  const billOptions = bills.map((b) => ({
    value: String(b.billId),
    label: b.billNo + ' · ' + b.customerName + ' · ' + b.period,
  }))
  const billMap = useMemo(() => new Map(bills.map((b) => [b.billId, b])), [bills])

  const slice = pageSlice(rows, page, pageSize)

  return (
    <div>
      <PageHead title={p.title} desc={p.desc} />
      {notice && <div className="mb-4 rounded-md border border-[color-mix(in_srgb,var(--color-success)_25%,transparent)] bg-[color-mix(in_srgb,var(--color-success)_8%,transparent)] px-4 py-2 text-[13px] text-[var(--color-success)]">{notice}</div>}
      <Card>
        <div className="flex flex-wrap items-center gap-2 p-4">
          <SimplePicker
            value={billId}
            onChange={(v) => { setBillId(v); setPage(1) }}
            options={billOptions}
            ariaLabel={p.filterBill}
            clearable
            clearLabel={t.pages.pickers.common.clear}
            emptyLabel={t.pages.pickers.common.all}
            error={billsErr}
            onRetry={loadBills}
            errorText={p.billLoadFail}
            minWidth={260}
          />
          <span className="spacer" />
          {canCollect && (
            <ToolbarButton primary onClick={() => { setNotice(''); setFormOpen(true) }}>{p.addBtn}</ToolbarButton>
          )}
          <ToolbarButton disabled={busy} onClick={load}>{t.pages.audit.refresh}</ToolbarButton>
        </div>
        {error ? <ErrorBanner message={error} /> : (
          <div className="px-4 pb-4">
            <Table>
              <TableHeader><TableRow>{p.columns.map((x) => <TableHead key={x}>{x}</TableHead>)}</TableRow></TableHeader>
              <TableBody>
                {slice.map((r) => (
                  <TableRow key={r.id}>
                    <TableCell className="font-mono">{r.payNo}</TableCell>
                    <TableCell><BillRef billId={r.billId} map={billMap} noBillText={p.noBill} /></TableCell>
                    <TableCell>{fmtFee(r.amount)}</TableCell>
                    <TableCell>{p.methods[r.method] ?? r.method}</TableCell>
                    <TableCell><StatusTag domain="payment" value={r.status} /></TableCell>
                    <TableCell>{r.siteName || '-'}</TableCell>
                    <TableCell>{r.operatorName || '-'}</TableCell>
                    <TableCell>
                      {r.status === 'SUCCESS' && canCollect && (
                        <ActionLink onClick={() => setRefund(r)} label={p.refundBtn} />
                      )}
                      {r.status !== 'SUCCESS' && <span className="text-[var(--shell-group-title)]">—</span>}
                    </TableCell>
                  </TableRow>
                ))}
                {!slice.length && <TableStateRow colSpan={8} loading={busy} text={p.empty} />}
              </TableBody>
            </Table>
          </div>
        )}
        <CardFooter>
          <Pagination total={rows.length} page={page} pageSize={pageSize}
            onPage={setPage} onSize={setPageSize} {...pagerTexts(p)} />
        </CardFooter>
      </Card>
      {formOpen && (
        <CounterPaymentForm
          onClose={() => setFormOpen(false)}
          onDone={(payNo) => {
            setFormOpen(false)
            toast.success(p.success.replace('{payNo}', payNo))
            setNotice(p.success.replace('{payNo}', payNo))
            load()
          }}
        />
      )}
      {refund && (
        <RefundDialog
          paymentId={refund.id}
          payNo={refund.payNo}
          onClose={() => setRefund(null)}
          onDone={() => { setRefund(null); load() }}
        />
      )}
    </div>
  )
}

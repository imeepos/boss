// 工程应付台账页(P-INFRA-1 W6/F8,000219;挂 billing 分组,menu:payables)。
// 应付由 SETTLED 结算单自动生成,本页承载台账列表/详情/付款登记/核减/发票登记。
// 后端 /odn/payables 无分页参数(limit 上限拉取),前端本地分页+总条数(登记汇报)。
import { useCallback, useEffect, useMemo, useState } from 'react'
import { apiFetch } from '../../../api/client'
import { useT } from '../../../i18n'
import { Badge } from '../../../components/ui/badge'
import { Dropdown } from '../../../components/Dropdown'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '../../../components/ui/table'
import { Card, CardFooter } from '../../../components/ui/card'
import { EmptyState, ErrorBanner, ToolbarButton } from '../../../components/business/page-head'
import { Pagination } from '../../../components/Pagination'
import { PageHead, pagerTexts } from '../../org/shared'
import { fmtTime } from '../../../lib/format'
import { pageSlice } from '../types'
import { PAYABLE_VARIANT, fmtMoney, type Payable, type PayableDetail } from './types'
import { PayRegistrationForms } from './RegistrationForms'

const STATUS_KEYS = ['OPEN', 'PARTIAL', 'PAID', 'VOIDED'] as const

export default function PayablesPage() {
  const t = useT()
  const p = t.pages.payablesPage
  const [rows, setRows] = useState<Payable[]>([])
  const [error, setError] = useState('')
  const [status, setStatus] = useState('')
  const [projectId, setProjectId] = useState('')
  const [detail, setDetail] = useState<PayableDetail | null>(null)
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(10)
  const [busy, setBusy] = useState(false)

  const load = useCallback(async () => {
    setError('')
    setBusy(true)
    try {
      const query: Record<string, string | number | undefined> = { limit: 200 }
      if (status) query.status = status
      setRows((await apiFetch<Payable[]>('/odn/payables', { query })) ?? [])
    } catch (e) { setError(e instanceof Error ? e.message : p.loadFail) } finally { setBusy(false) }
  }, [status, p.loadFail])
  useEffect(() => { void load() }, [load])

  const openDetail = async (id: number) => {
    setError('')
    setBusy(true)
    try {
      const d = await apiFetch<PayableDetail>('/odn/payables/' + id)
      setDetail(d ?? null)
    } catch (e) { setError(e instanceof Error ? e.message : p.loadFail) } finally { setBusy(false) }
  }

  const projectOptions = useMemo(() => {
    const seen = new Map<number, string>()
    rows.forEach((r) => seen.set(r.projectId, r.projectNo))
    return [{ value: '', label: p.filterAllProject },
      ...[...seen.entries()].map(([id, no]) => ({ value: String(id), label: no }))]
  }, [rows, p.filterAllProject])

  const filtered = useMemo(
    () => (projectId ? rows.filter((r) => String(r.projectId) === projectId) : rows),
    [rows, projectId],
  )
  const slice = pageSlice(filtered, page, pageSize)
  const ap = detail?.payable
  const canOperate = ap != null && ap.status !== 'VOIDED'
  const statusText = (s: string) => p.statusTexts[s] ?? s

  return <div>
    <PageHead title={p.title} desc={p.desc} />
    <Card>
      <div className='flex flex-wrap items-center gap-2 p-4'>
        <Dropdown value={status} ariaLabel={p.filterStatus} placeholder={p.filterAllStatus}
          options={[{ value: '', label: p.filterAllStatus }, ...STATUS_KEYS.map((s) => ({ value: s, label: statusText(s) }))]}
          onChange={(v) => { setStatus(v); setPage(1) }} />
        <Dropdown value={projectId} ariaLabel={p.filterProject} placeholder={p.filterAllProject}
          options={projectOptions} onChange={(v) => { setProjectId(v); setPage(1) }} />
        <span className='spacer' />
        <ToolbarButton disabled={busy} onClick={() => void load()}>{t.pages.audit.refresh}</ToolbarButton>
      </div>
      {error && <ErrorBanner message={error} />}
      {slice.length === 0 ? <div className='px-4 pb-6'><EmptyState text={p.empty} /></div> : (
        <div className='px-4 pb-4'>
          <Table>
            <TableHeader><TableRow>{p.columns.map((c) => <TableHead key={c}>{c}</TableHead>)}</TableRow></TableHeader>
            <TableBody>
              {slice.map((r) => (
                <TableRow key={r.id} className={detail?.payable.id === r.id ? 'bg-[var(--shell-menu-hover-bg)]' : ''}>
                  <TableCell className='font-mono'>{r.payableNo}</TableCell>
                  <TableCell className='font-mono'>{r.settlementNo}</TableCell>
                  <TableCell className='font-mono'>{r.projectNo}</TableCell>
                  <TableCell>{r.contractorName || '-'}</TableCell>
                  <TableCell>{fmtMoney(r.payableAmount)}</TableCell>
                  <TableCell>{fmtMoney(r.deductedAmount)}</TableCell>
                  <TableCell>{fmtMoney(r.paidAmount)}</TableCell>
                  <TableCell>{fmtMoney(r.balance)}</TableCell>
                  <TableCell><Badge variant={PAYABLE_VARIANT[r.status] ?? 'default'}>{statusText(r.status)}</Badge></TableCell>
                  <TableCell>
                    <button className='px-1 text-xs text-[var(--color-text-link)] bg-none border-none cursor-pointer hover:underline'
                      onClick={() => void openDetail(r.id)}>
                      {detail?.payable.id === r.id ? p.detailRefresh : p.detail}
                    </button>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </div>
      )}
      <CardFooter>
        <Pagination total={filtered.length} page={page} pageSize={pageSize}
          onPage={setPage} onSize={setPageSize} {...pagerTexts(p)} />
      </CardFooter>
    </Card>
    {ap && <div className='mt-3 space-y-3'>
      <Card className='mb-0 p-4'>
        <div className='mb-2 text-sm font-semibold text-[var(--shell-heading)]'>{p.detailTitle} {ap.payableNo}</div>
        <div className='grid grid-cols-2 gap-2 text-xs md:grid-cols-4'>
          <span>{p.settlement}:<span className='font-mono'>{ap.settlementNo}</span></span>
          <span>{p.project}:<span className='font-mono'>{ap.projectNo}</span></span>
          <span>{p.contractor}:{ap.contractorName || '-'}</span>
          <span>{p.createdAt}:{fmtTime(ap.createdAt)}</span>
          <span>{p.payableAmount}:{fmtMoney(ap.payableAmount)}</span>
          <span>{p.deducted}:{fmtMoney(ap.deductedAmount)}</span>
          <span>{p.paid}:{fmtMoney(ap.paidAmount)}</span>
          <span>{p.balance}:{fmtMoney(ap.balance)}{ap.status === 'VOIDED' ? p.voidedNote : ''}</span>
        </div>
        {ap.voidReason && <div className='mt-2 text-xs text-[var(--color-danger)]'>{p.voidReason}:{ap.voidReason}</div>}
      </Card>
      {canOperate && <PayRegistrationForms payableId={ap.id} onDone={() => { void openDetail(ap.id); void load() }} />}
      <LogCard title={p.paymentsTitle} emptyText={p.paymentsEmpty} cols={p.paymentCols}
        empty={!detail || detail.payments.length === 0}
        rows={(detail?.payments ?? []).map((x) => [x.paymentNo, fmtMoney(x.amount), p.methodTexts[x.method] ?? x.method, fmtTime(x.paidAt), x.reference || '-', x.note || '-'])} />
      <div className='grid grid-cols-1 gap-3 lg:grid-cols-2'>
        <LogCard title={p.deductionsTitle} emptyText={p.deductionsEmpty} cols={p.deductionCols}
          empty={!detail || detail.deductions.length === 0}
          rows={(detail?.deductions ?? []).map((x) => [fmtMoney(x.amount), x.reason, fmtTime(x.createdAt)])} />
        <LogCard title={p.invoicesLogTitle} emptyText={p.invoicesLogEmpty} cols={p.invoiceCols}
          empty={!detail || detail.invoices.length === 0}
          rows={(detail?.invoices ?? []).map((x) => [x.invoiceNo, fmtMoney(x.amount), x.invoicedAt || '-'])} />
      </div>
    </div>}
  </div>
}

function LogCard({ title, emptyText, cols, empty, rows }: {
  title: string
  emptyText: string
  cols: string[]
  empty: boolean
  rows: string[][]
}) {
  return (
    <Card className='mb-0 p-4'>
      <div className='mb-2 text-sm font-semibold text-[var(--shell-heading)]'>{title}</div>
      {empty ? <EmptyState text={emptyText} /> : (
        <Table>
          <TableHeader><TableRow>{cols.map((c) => <TableHead key={c}>{c}</TableHead>)}</TableRow></TableHeader>
          <TableBody>
            {rows.map((cells, i) => (
              <TableRow key={i}>{cells.map((c, j) => <TableCell key={j}>{c}</TableCell>)}</TableRow>
            ))}
          </TableBody>
        </Table>
      )}
    </Card>
  )
}

// 工程应付台账页(P-INFRA-1 W6/F8,000219;挂 billing 分组,menu:payables)。
// 应付由 SETTLED 结算单自动生成,本页承载台账列表/详情/付款登记/核减/发票登记。
// 文案为字面量:沿用 W1/W4 先例,中央登记仅登记菜单项。
import { useCallback, useEffect, useState } from 'react'
import { toast } from 'sonner'
import { apiFetch } from '../../api/client'
import { Input } from '../../components/ui/input'
import { Badge } from '../../components/ui/badge'
import { Dropdown } from '../../components/Dropdown'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '../../components/ui/table'
import { EmptyState, ErrorBanner, ToolbarButton } from '../../components/business/page-head'

const CARD = 'rounded-md border border-[var(--shell-card-border)] bg-[var(--shell-card-bg)] shadow-[var(--shell-card-shadow)]'
const FIELD = 'flex flex-col gap-1'
const LABEL = 'text-xs text-[var(--shell-content-text)]'

interface Payable {
  id: number
  payableNo: string
  settlementId: number
  settlementNo: string
  projectId: number
  projectNo: string
  contractorId: number
  contractorName: string
  payableAmount: number
  deductedAmount: number
  paidAmount: number
  balance: number
  status: string
  voidReason: string
  createdAt: string
}
interface PayablePayment { id: number; payableId: number; paymentNo: string; amount: number; method: string; paidAt: string; reference: string; note: string; createdAt: string }
interface PayableDeduction { id: number; payableId: number; amount: number; reason: string; createdAt: string }
interface PayableInvoice { id: number; payableId: number; invoiceNo: string; amount: number; invoicedAt?: string; note: string; createdAt: string }
interface PayableDetail { payable: Payable; payments: PayablePayment[]; deductions: PayableDeduction[]; invoices: PayableInvoice[] }

const P_TEXT: Record<string, string> = { OPEN: '未付', PARTIAL: '部分付款', PAID: '已付清', VOIDED: '已冲销' }
const P_VARIANT: Record<string, 'default' | 'success' | 'warning' | 'danger'> = { OPEN: 'warning', PARTIAL: 'warning', PAID: 'success', VOIDED: 'danger' }
const M_TEXT: Record<string, string> = { TRANSFER: '银行转账', CASH: '现金', CHEQUE: '支票', OTHER: '其他' }

function fmtMoney(v: number | null | undefined): string {
  return (v ?? 0).toLocaleString('zh-CN', { minimumFractionDigits: 2, maximumFractionDigits: 2 })
}

export default function PayablesPage() {
  const [rows, setRows] = useState<Payable[]>([])
  const [error, setError] = useState('')
  const [status, setStatus] = useState('')
  const [projectId, setProjectId] = useState('')
  const [detail, setDetail] = useState<PayableDetail | null>(null)
  const [pay, setPay] = useState({ amount: '', method: 'TRANSFER', paidAt: '', reference: '', note: '' })
  const [deduct, setDeduct] = useState({ amount: '', reason: '' })
  const [invoice, setInvoice] = useState({ invoiceNo: '', amount: '', invoicedAt: '', note: '' })
  const [busy, setBusy] = useState(false)

  const load = useCallback(async () => {
    setError('')
    try {
      const query: Record<string, string | number | undefined> = { limit: 200 }
      if (status) query.status = status
      if (projectId.trim()) query.projectId = Number(projectId)
      setRows((await apiFetch<Payable[]>('/odn/payables', { query })) ?? [])
    } catch (e) { setError(e instanceof Error ? e.message : '加载失败') }
  }, [status, projectId])
  useEffect(() => { void load() }, [load])

  const openDetail = async (id: number) => {
    setError('')
    try {
      const d = await apiFetch<PayableDetail>('/odn/payables/' + id)
      setDetail(d ?? null)
    } catch (e) { setError(e instanceof Error ? e.message : '加载失败') }
  }

  const act = async (fn: () => Promise<unknown>, okMsg: string, id: number) => {
    setBusy(true); setError('')
    try { await fn(); toast.success(okMsg); await openDetail(id); await load() }
    catch (e) { setError(e instanceof Error ? e.message : '操作失败') } finally { setBusy(false) }
  }

  const registerPay = () => {
    if (!detail) return
    const id = detail.payable.id
    void act(() => apiFetch('/odn/payables/' + id + '/payments', { method: 'POST', body: {
      amount: Number(pay.amount), method: pay.method, paidAt: pay.paidAt.trim(),
      reference: pay.reference.trim(), note: pay.note.trim() } }), '付款已登记', id)
      .then(() => setPay({ amount: '', method: 'TRANSFER', paidAt: '', reference: '', note: '' }))
  }
  const registerDeduct = () => {
    if (!detail) return
    const id = detail.payable.id
    if (!deduct.reason.trim()) { setError('核减原因必填'); return }
    void act(() => apiFetch('/odn/payables/' + id + '/deductions', { method: 'POST', body: {
      amount: Number(deduct.amount), reason: deduct.reason.trim() } }), '核减已登记', id)
      .then(() => setDeduct({ amount: '', reason: '' }))
  }
  const registerInvoice = () => {
    if (!detail) return
    const id = detail.payable.id
    if (!invoice.invoiceNo.trim()) { setError('发票号必填'); return }
    void act(() => apiFetch('/odn/payables/' + id + '/invoices', { method: 'POST', body: {
      invoiceNo: invoice.invoiceNo.trim(), amount: Number(invoice.amount),
      invoicedAt: invoice.invoicedAt.trim(), note: invoice.note.trim() } }), '发票已登记', id)
      .then(() => setInvoice({ invoiceNo: '', amount: '', invoicedAt: '', note: '' }))
  }

  const setPayField = (k: string, v: string) => setPay((m) => ({ ...m, [k]: v }))
  const setDeductField = (k: string, v: string) => setDeduct((m) => ({ ...m, [k]: v }))
  const setInvoiceField = (k: string, v: string) => setInvoice((m) => ({ ...m, [k]: v }))

  const ap = detail?.payable
  const canOperate = ap != null && ap.status !== 'VOIDED'

  return <div>
    <div className='mb-3 flex flex-wrap items-end justify-between gap-2'>
      <div className='flex flex-wrap items-end gap-2'>
        <Dropdown value={status} ariaLabel='状态筛选' placeholder='全部状态' options={Object.keys(P_TEXT).map((s) => ({ value: s, label: P_TEXT[s] }))} onChange={setStatus} />
        <label className={FIELD}><span className={LABEL}>项目 ID</span><Input className='w-28' value={projectId} onChange={(e) => setProjectId(e.target.value.replace(/[^0-9]/g, ''))} placeholder='按项目过滤' /></label>
      </div>
      <ToolbarButton onClick={() => void load()}>刷新</ToolbarButton>
    </div>
    {error && <ErrorBanner message={error} className='mb-3' />}
    <section className={CARD + ' overflow-hidden'}>
      {rows.length === 0 ? <EmptyState text='暂无应付(SETTLED 结算单自动生成应付记录)' /> : <div className='overflow-x-auto'><Table>
        <TableHeader><TableRow><TableHead>应付单号</TableHead><TableHead>来源结算单</TableHead><TableHead>项目</TableHead><TableHead>承包商</TableHead><TableHead>应付金额</TableHead><TableHead>核减</TableHead><TableHead>已付</TableHead><TableHead>未付余额</TableHead><TableHead>状态</TableHead><TableHead>操作</TableHead></TableRow></TableHeader>
        <TableBody>
          {rows.map((r) => <TableRow key={r.id} className={detail?.payable.id === r.id ? 'bg-[var(--shell-row-active)]' : ''}>
            <TableCell className='font-mono'>{r.payableNo}</TableCell>
            <TableCell className='font-mono'>{r.settlementNo}</TableCell>
            <TableCell className='font-mono'>{r.projectNo}</TableCell>
            <TableCell>{r.contractorName || '-'}</TableCell>
            <TableCell>{fmtMoney(r.payableAmount)}</TableCell>
            <TableCell>{fmtMoney(r.deductedAmount)}</TableCell>
            <TableCell>{fmtMoney(r.paidAmount)}</TableCell>
            <TableCell>{fmtMoney(r.balance)}</TableCell>
            <TableCell><Badge variant={P_VARIANT[r.status] ?? 'default'}>{P_TEXT[r.status] ?? r.status}</Badge></TableCell>
            <TableCell><button className='text-[var(--color-text-link)]' onClick={() => void openDetail(r.id)}>{detail?.payable.id === r.id ? '刷新' : '详情'}</button></TableCell>
          </TableRow>)}
        </TableBody>
      </Table></div>}
    </section>
    {ap && <div className='mt-3 space-y-3'>
      <div className={CARD + ' p-4'}>
        <div className='mb-2 text-sm font-semibold'>应付详情 {ap.payableNo}</div>
        <div className='grid grid-cols-2 gap-2 text-xs md:grid-cols-4'>
          <span>来源结算单:<span className='font-mono'>{ap.settlementNo}</span></span>
          <span>项目:<span className='font-mono'>{ap.projectNo}</span></span>
          <span>承包商:{ap.contractorName || '-'}</span>
          <span>生成时间:{ap.createdAt}</span>
          <span>应付金额:{fmtMoney(ap.payableAmount)}</span>
          <span>核减合计:{fmtMoney(ap.deductedAmount)}</span>
          <span>已付合计:{fmtMoney(ap.paidAmount)}</span>
          <span>未付余额:{fmtMoney(ap.balance)}{ap.status === 'VOIDED' ? '(已冲销,余额可为负=超付)' : ''}</span>
        </div>
        {ap.voidReason && <div className='mt-2 text-xs text-[var(--color-text-danger)]'>冲销原因:{ap.voidReason}</div>}
      </div>
      {canOperate && <div className='grid grid-cols-1 gap-3 lg:grid-cols-3'>
        <div className={CARD + ' p-4'}>
          <div className='mb-2 text-sm font-semibold'>登记付款</div>
          <div className='space-y-2'>
            <label className={FIELD}><span className={LABEL}>金额</span><Input value={pay.amount} onChange={(e) => setPayField('amount', e.target.value.replace(/[^0-9.]/g, ''))} inputMode='decimal' /></label>
            <Dropdown value={pay.method} ariaLabel='付款方式' options={Object.keys(M_TEXT).map((m) => ({ value: m, label: M_TEXT[m] }))} onChange={(v) => setPayField('method', v)} />
            <label className={FIELD}><span className={LABEL}>付款时间(YYYY-MM-DD HH:MM:SS,留空=当前)</span><Input value={pay.paidAt} onChange={(e) => setPayField('paidAt', e.target.value)} placeholder='2026-09-07 12:00:00' /></label>
            <label className={FIELD}><span className={LABEL}>凭证号</span><Input value={pay.reference} onChange={(e) => setPayField('reference', e.target.value)} placeholder='可空' /></label>
            <label className={FIELD}><span className={LABEL}>备注</span><Input value={pay.note} onChange={(e) => setPayField('note', e.target.value)} placeholder='可空' /></label>
            <div className='flex justify-end'><ToolbarButton primary disabled={busy || !(Number(pay.amount) > 0)} onClick={registerPay}>登记付款</ToolbarButton></div>
          </div>
        </div>
        <div className={CARD + ' p-4'}>
          <div className='mb-2 text-sm font-semibold'>核减(原因必填留痕)</div>
          <div className='space-y-2'>
            <label className={FIELD}><span className={LABEL}>金额</span><Input value={deduct.amount} onChange={(e) => setDeductField('amount', e.target.value.replace(/[^0-9.]/g, ''))} inputMode='decimal' /></label>
            <label className={FIELD}><span className={LABEL}>原因</span><Input value={deduct.reason} onChange={(e) => setDeductField('reason', e.target.value)} placeholder='如 复审工程量核减' /></label>
            <div className='flex justify-end'><ToolbarButton primary disabled={busy || !(Number(deduct.amount) > 0)} onClick={registerDeduct}>登记核减</ToolbarButton></div>
          </div>
        </div>
        <div className={CARD + ' p-4'}>
          <div className='mb-2 text-sm font-semibold'>发票登记</div>
          <div className='space-y-2'>
            <label className={FIELD}><span className={LABEL}>发票号</span><Input value={invoice.invoiceNo} onChange={(e) => setInvoiceField('invoiceNo', e.target.value)} /></label>
            <label className={FIELD}><span className={LABEL}>金额</span><Input value={invoice.amount} onChange={(e) => setInvoiceField('amount', e.target.value.replace(/[^0-9.]/g, ''))} inputMode='decimal' /></label>
            <label className={FIELD}><span className={LABEL}>开票日(YYYY-MM-DD)</span><Input value={invoice.invoicedAt} onChange={(e) => setInvoiceField('invoicedAt', e.target.value)} placeholder='可空' /></label>
            <label className={FIELD}><span className={LABEL}>备注</span><Input value={invoice.note} onChange={(e) => setInvoiceField('note', e.target.value)} placeholder='可空' /></label>
            <div className='flex justify-end'><ToolbarButton primary disabled={busy || !(Number(invoice.amount) > 0) || !invoice.invoiceNo.trim()} onClick={registerInvoice}>登记发票</ToolbarButton></div>
          </div>
        </div>
      </div>}
      <div className={CARD + ' p-4'}>
        <div className='mb-2 text-sm font-semibold'>付款流水</div>
        {(detail?.payments.length ?? 0) === 0 ? <EmptyState text='暂无付款' /> : <div className='overflow-x-auto'><Table>
          <TableHeader><TableRow><TableHead>流水号</TableHead><TableHead>金额</TableHead><TableHead>方式</TableHead><TableHead>付款时间</TableHead><TableHead>凭证号</TableHead><TableHead>备注</TableHead></TableRow></TableHeader>
          <TableBody>
            {detail!.payments.map((p) => <TableRow key={p.id}>
              <TableCell className='font-mono'>{p.paymentNo}</TableCell>
              <TableCell>{fmtMoney(p.amount)}</TableCell>
              <TableCell>{M_TEXT[p.method] ?? p.method}</TableCell>
              <TableCell>{p.paidAt}</TableCell>
              <TableCell>{p.reference || '-'}</TableCell>
              <TableCell>{p.note || '-'}</TableCell>
            </TableRow>)}
          </TableBody>
        </Table></div>}
      </div>
      <div className='grid grid-cols-1 gap-3 lg:grid-cols-2'>
        <div className={CARD + ' p-4'}>
          <div className='mb-2 text-sm font-semibold'>核减明细</div>
          {(detail?.deductions.length ?? 0) === 0 ? <EmptyState text='暂无核减' /> : <Table>
            <TableHeader><TableRow><TableHead>金额</TableHead><TableHead>原因</TableHead><TableHead>登记时间</TableHead></TableRow></TableHeader>
            <TableBody>
              {detail!.deductions.map((d) => <TableRow key={d.id}>
                <TableCell>{fmtMoney(d.amount)}</TableCell>
                <TableCell>{d.reason}</TableCell>
                <TableCell>{d.createdAt}</TableCell>
              </TableRow>)}
            </TableBody>
          </Table>}
        </div>
        <div className={CARD + ' p-4'}>
          <div className='mb-2 text-sm font-semibold'>发票登记</div>
          {(detail?.invoices.length ?? 0) === 0 ? <EmptyState text='暂无发票' /> : <Table>
            <TableHeader><TableRow><TableHead>发票号</TableHead><TableHead>金额</TableHead><TableHead>开票日</TableHead></TableRow></TableHeader>
            <TableBody>
              {detail!.invoices.map((v) => <TableRow key={v.id}>
                <TableCell className='font-mono'>{v.invoiceNo}</TableCell>
                <TableCell>{fmtMoney(v.amount)}</TableCell>
                <TableCell>{v.invoicedAt || '-'}</TableCell>
              </TableRow>)}
            </TableBody>
          </Table>}
        </div>
      </div>
    </div>}
  </div>
}
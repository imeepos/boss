// 渠道对账页:双页签——渠道对账(GET /reconciliations) + 账实核对(GET /billing/ledger-recon)。
// 行动作(录入流水/结算)失败走 toast 完整透出原因;录入流水抽屉统一 FormField/Input。
import { useEffect, useState } from 'react'
import { toast } from 'sonner'
import { apiFetch } from '../../../api/client'
import { useT } from '../../../i18n'
import { PageHead, pagerTexts } from '../../org/shared'
import { StatusTag } from '../../../components/StatusTag'
import { Drawer } from '../../../components/Drawer'
import { Pagination } from '../../../components/Pagination'
import { Card, CardFooter } from '../../../components/ui/card'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '../../../components/ui/table'
import { FormField } from '../../../components/business/form-field'
import { Input } from '../../../components/ui/input'
import { pageSlice, type ReconRow, type LedgerReconRow, type LedgerReconSummary } from '../types'
import { fmtFee, fmtTime } from '../../../lib/format'
import { useConfirm } from '../../../components/ConfirmDialog'
import { TableStateRow, TabBar } from '../../../components/business'
import { ActionLink, ActionSep, ToolbarButton } from '../../../components/business/page-head'

const PERIOD_RE = /^\d{4}-(0[1-9]|1[0-2])$/

function currentPeriod(): string {
  const d = new Date()
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}`
}

export default function PayCheckPage() {
  const t = useT()
  const confirmDialog = useConfirm()
  const p = t.pages.paycheck
  const [tab, setTab] = useState<'channel' | 'ledger'>('channel')

  // 渠道对账状态
  const [rows, setRows] = useState<ReconRow[]>([])
  const [chPage, setChPage] = useState(1)
  const [chSize, setChSize] = useState(10)
  const [busy, setBusy] = useState(false)
  // 渠道流水录入(P1-E:POST /reconciliations/:batchNo/statement,单行录入)。
  const [stmt, setStmt] = useState<{ batchNo: string; ref: string; amount: string } | null>(null)
  const [stmtError, setStmtError] = useState('')

  // 账实核对状态(服务端分页)
  const [period, setPeriod] = useState(currentPeriod())
  const [periodInput, setPeriodInput] = useState(currentPeriod())
  const [ledgerRows, setLedgerRows] = useState<LedgerReconRow[]>([])
  const [ledgerTotal, setLedgerTotal] = useState(0)
  const [summary, setSummary] = useState<LedgerReconSummary | null>(null)
  const [lgPage, setLgPage] = useState(1)
  const [lgSize, setLgSize] = useState(20)
  const [error, setError] = useState('')

  const loadChannel = () => {
    setError('')
    setBusy(true)
    apiFetch<{ items: ReconRow[] }>('/reconciliations')
      .then((d) => setRows(d?.items ?? []))
      .catch((e) => setError(e instanceof Error ? e.message : p.loadFail))
      .finally(() => setBusy(false))
  }
  useEffect(loadChannel, []) // eslint-disable-line react-hooks/exhaustive-deps

  const loadLedger = () => {
    if (!PERIOD_RE.test(periodInput)) {
      setError(p.ledgerPeriodInvalid)
      return
    }
    setError('')
    setBusy(true)
    setPeriod(periodInput)
    apiFetch<{ items: LedgerReconRow[]; total: number; summary: LedgerReconSummary }>(
      '/billing/ledger-recon',
      { query: { period: periodInput, page: lgPage, pageSize: lgSize } },
    )
      .then((d) => {
        setLedgerRows(d?.items ?? [])
        setLedgerTotal(d?.total ?? 0)
        setSummary(d?.summary ?? null)
      })
      .catch((e) => setError(e instanceof Error ? e.message : p.loadFail))
      .finally(() => setBusy(false))
  }
  useEffect(() => {
    if (tab === 'ledger' && PERIOD_RE.test(period)) loadLedger()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [tab, lgPage, lgSize])

  const settle = async (batchNo: string) => {
    if (busy) return
    if (!(await confirmDialog(p.settleConfirm, { danger: true }))) return
    setBusy(true)
    try {
      await apiFetch(`/reconciliations/${encodeURIComponent(batchNo)}/settle`, { method: 'POST' })
      toast.success(p.settleOk)
      loadChannel()
    } catch (e) {
      toast.error(e instanceof Error ? e.message : p.actionFail)
    } finally {
      setBusy(false)
    }
  }

  // 自动对账:按当日建批并从已配置源拉流水比对(manual 渠道只建批)。
  const runAuto = async () => {
    if (busy) return
    setBusy(true); setError('')
    try {
      const d = await apiFetch<{ date: string }>('/reconciliations/auto', { method: 'POST' })
      toast.success(p.autoOk.replace('{date}', d?.date ?? ''))
      loadChannel()
    } catch (e) {
      toast.error(e instanceof Error ? e.message : p.actionFail)
    } finally {
      setBusy(false)
    }
  }

  const saveStatement = async () => {
    if (!stmt || busy) return
    const amount = Number(stmt.amount)
    if (!stmt.ref.trim() || !Number.isFinite(amount) || amount === 0) return
    setBusy(true); setStmtError('')
    try {
      await apiFetch(`/reconciliations/${encodeURIComponent(stmt.batchNo)}/statement`, {
        method: 'POST',
        body: { rows: [{ channelRef: stmt.ref.trim(), amount }] },
      })
      toast.success(p.statementOk)
      setStmt(null)
      loadChannel()
    } catch (e) {
      const msg = e instanceof Error ? e.message : p.actionFail
      setStmtError(msg)
      toast.error(msg)
    } finally {
      setBusy(false)
    }
  }

  const slice = pageSlice(rows, chPage, chSize)
  const diffCount = summary
    ? Object.entries(summary.byKind ?? {}).reduce((n, [k, v]) => (k === 'MATCH' ? n : n + v), 0)
    : 0

  return (
    <div>
      <PageHead title={p.title} desc={p.desc} />
      <Card>
        <div className="px-4 pt-3">
          <TabBar
            tabs={[{ key: 'channel' as const, label: p.tabChannel }, { key: 'ledger' as const, label: p.tabLedger }]}
            value={tab}
            onChange={(key) => { setTab(key); setError('') }}
          />
        </div>
        <div className="flex items-center gap-1 mb-3">
          {tab === 'ledger' && (
            <>
              <Input className="w-40" placeholder={p.ledgerPeriod}
                value={periodInput} onChange={(e) => setPeriodInput(e.target.value)}
                onKeyDown={(e) => { if (e.key === 'Enter') { setLgPage(1); loadLedger() } }} />
              <ToolbarButton disabled={busy} onClick={() => { setLgPage(1); loadLedger() }}>{p.ledgerQuery}</ToolbarButton>
            </>
          )}
          <span className="spacer" />
          <ToolbarButton disabled={busy} onClick={() => (tab === 'channel' ? loadChannel() : loadLedger())}>{t.pages.audit.refresh}</ToolbarButton>
          {tab === 'channel' && <ToolbarButton primary disabled={busy} onClick={runAuto}>{p.autoBtn}</ToolbarButton>}
        </div>
        {error ? <div className="mx-4 mb-3 rounded-sm border border-[color-mix(in_srgb,var(--color-danger)_25%,transparent)] bg-[color-mix(in_srgb,var(--color-danger)_8%,transparent)] px-3 py-2 text-[13px] text-[var(--color-danger)]"><span className="break-all">{error}</span></div>
          : tab === 'channel' ? (
          <div className="px-4 pb-4">
            <Table>
              <TableHeader><TableRow>{p.columns.map((x) => <TableHead key={x}>{x}</TableHead>)}</TableRow></TableHeader>
              <TableBody>
                {slice.map((r) => (
                  <TableRow key={r.id}>
                    <TableCell className="font-mono">{r.batchNo}</TableCell>
                    <TableCell>{r.channel}</TableCell>
                    <TableCell>{fmtFee(r.channelAmount)}</TableCell>
                    <TableCell>{fmtFee(r.systemAmount)}</TableCell>
                    <TableCell>{fmtFee(r.diff)}</TableCell>
                    <TableCell><StatusTag domain="recon" value={r.status} /></TableCell>
                    <TableCell>
                      {r.status === 'DIFF_PENDING' ? (
                        <span className="inline-flex items-center">
                          <ActionLink testId={`statement-${r.batchNo}`} onClick={() => { setStmtError(''); setStmt({ batchNo: r.batchNo, ref: '', amount: '' }) }} label={p.statementBtn} />
                          <ActionSep />
                          <ActionLink onClick={() => settle(r.batchNo)} label={p.settle} />
                        </span>
                      ) : fmtTime(r.settledAt ?? '')}
                    </TableCell>
                  </TableRow>
                ))}
                {!slice.length && <TableStateRow colSpan={7} loading={busy} text={p.empty} />}
              </TableBody>
            </Table>
          </div>
        ) : (
          <>
            {summary && (
              <div className="flex flex-wrap items-center gap-4 px-4 pb-3 text-[13px] text-[var(--shell-content-text)]">
                <span>{p.ledgerBillsTotal}: <b className="text-[var(--shell-heading)]">{fmtFee(summary.billsTotal)}</b></span>
                <span>{p.ledgerPaidTotal}: <b className="text-[var(--shell-heading)]">{fmtFee(summary.paidTotal)}</b></span>
                <span>{p.ledgerInvoiceTotal}: <b className="text-[var(--shell-heading)]">{fmtFee(summary.invoiceTotal)}</b></span>
                <span className={diffCount > 0 ? 'text-[var(--color-danger)]' : 'text-[var(--color-success)]'}>
                  {p.ledgerDiffCount.replace('{count}', String(diffCount))}
                </span>
              </div>
            )}
            <div className="px-4 pb-4">
              <Table>
                <TableHeader><TableRow>{p.ledgerColumns.map((x) => <TableHead key={x}>{x}</TableHead>)}</TableRow></TableHeader>
                <TableBody>
                  {ledgerRows.map((r) => (
                    <TableRow key={r.billId}>
                      <TableCell className="font-mono">{r.billNo}</TableCell>
                      <TableCell>{r.customerName}</TableCell>
                      <TableCell>{r.legalEntityName}</TableCell>
                      <TableCell>{fmtFee(r.billAmount)}</TableCell>
                      <TableCell>{fmtFee(r.paidAmount)}</TableCell>
                      <TableCell>{fmtFee(r.invoiceAmount)}</TableCell>
                      <TableCell><StatusTag domain="ledgerRecon" value={r.diffKind} /></TableCell>
                      <TableCell>{r.invoiceNo || '—'}</TableCell>
                      <TableCell>{r.taxStatus || '—'}</TableCell>
                    </TableRow>
                  ))}
                  {!ledgerRows.length && <TableStateRow colSpan={9} loading={busy} text={p.empty} />}
                </TableBody>
              </Table>
            </div>
          </>
        )}
        <CardFooter>
          {tab === 'channel' ? (
            <Pagination total={rows.length} page={chPage} pageSize={chSize}
              onPage={setChPage} onSize={setChSize} {...pagerTexts(p)} />
          ) : (
            <Pagination total={ledgerTotal} page={lgPage} pageSize={lgSize}
              onPage={setLgPage} onSize={setLgSize} {...pagerTexts(p)} />
          )}
        </CardFooter>
      </Card>
      {stmt && (
        <Drawer title={p.statementTitle + ' · ' + stmt.batchNo} onClose={() => setStmt(null)}
          footer={
            <>
              <ToolbarButton onClick={() => setStmt(null)}>{t.pages.company.cancel}</ToolbarButton>
              <ToolbarButton primary disabled={busy || !stmt.ref.trim() || !(Number(stmt.amount) > 0)} onClick={saveStatement}>{busy ? t.pages.account.submitting : t.pages.company.save}</ToolbarButton>
            </>
          }>
          <div className="flex flex-col gap-3.5">
            <FormField label={p.statementRef} required>
              <Input value={stmt.ref} placeholder={t.pages.paycheck.statementRefPh}
                onChange={(e) => setStmt({ ...stmt, ref: e.target.value })} />
            </FormField>
            <FormField label={p.statementAmount} required error={stmtError || undefined}>
              <Input inputMode="decimal" value={stmt.amount} placeholder="0.00"
                onChange={(e) => setStmt({ ...stmt, amount: e.target.value })} />
            </FormField>
          </div>
        </Drawer>
      )}
    </div>
  )
}

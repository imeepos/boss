// 渠道对账页:双页签——渠道对账(GET /reconciliations) + 账实核对(GET /billing/ledger-recon)。
import { useEffect, useState } from 'react'
import { toast } from 'sonner'
import { apiFetch } from '../../../api/client'
import { useT } from '../../../i18n'
import { PageHead, pagerTexts } from '../../org/shared'
import { StatusTag } from '../../../components/StatusTag'
import { Drawer } from '../../../components/Drawer'
import { Pagination } from '../../../components/Pagination'
import { pageSlice, type ReconRow, type LedgerReconRow, type LedgerReconSummary } from '../types'
import { fmtFee, fmtTime } from '../../../lib/format'
import { useConfirm } from '../../../components/ConfirmDialog'
import { TableStateRow, TabBar } from '../../../components/business'

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
      setError(e instanceof Error ? e.message : p.actionFail)
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
      setError(e instanceof Error ? e.message : p.actionFail)
    } finally {
      setBusy(false)
    }
  }

  const saveStatement = async () => {
    if (!stmt || busy) return
    const amount = Number(stmt.amount)
    if (!stmt.ref.trim() || !Number.isFinite(amount) || amount === 0) return
    setBusy(true); setError('')
    try {
      await apiFetch(`/reconciliations/${encodeURIComponent(stmt.batchNo)}/statement`, {
        method: 'POST',
        body: { rows: [{ channelRef: stmt.ref.trim(), amount }] },
      })
      toast.success(p.statementOk)
      setStmt(null)
      loadChannel()
    } catch (e) {
      setError(e instanceof Error ? e.message : p.actionFail)
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
      <div className="mb-4 rounded-md border border-[var(--shell-card-border)] bg-[var(--shell-card-bg)] shadow-[var(--shell-card-shadow)]">
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
              <input className="h-8 rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-2.5 text-[13px] text-[var(--shell-content-text)] outline-none placeholder:text-[var(--shell-input-placeholder)] focus:border-[var(--color-border-focus)] w-40" placeholder={p.ledgerPeriod}
                value={periodInput} onChange={(e) => setPeriodInput(e.target.value)}
                onKeyDown={(e) => { if (e.key === 'Enter') { setLgPage(1); loadLedger() } }} />
              <button className="h-8 cursor-pointer rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-4 text-[13px] text-[var(--shell-content-text)] hover:border-[var(--color-border-hover)] hover:text-[var(--shell-heading)]" disabled={busy} onClick={() => { setLgPage(1); loadLedger() }}>{p.ledgerQuery}</button>
            </>
          )}
          <span className="spacer" />
          <button className="h-8 cursor-pointer rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-4 text-[13px] text-[var(--shell-content-text)] hover:border-[var(--color-border-hover)] hover:text-[var(--shell-heading)]" disabled={busy}
            onClick={() => (tab === 'channel' ? loadChannel() : loadLedger())}>{t.pages.audit.refresh}</button>
          {tab === 'channel' && <button className="h-8 cursor-pointer rounded-sm border-none bg-[var(--shell-fab-bg)] px-4 text-[13px] text-[var(--shell-fab-icon)] hover:bg-[var(--shell-fab-bg-hover)]" disabled={busy} onClick={runAuto}>{p.autoBtn}</button>}
        </div>
        {error ? <div className="mx-4 mb-3 rounded-sm border border-[color-mix(in_srgb,var(--color-danger)_25%,transparent)] bg-[color-mix(in_srgb,var(--color-danger)_8%,transparent)] px-3 py-2 text-[13px] text-[var(--color-danger)]">{error}</div>
          : tab === 'channel' ? (
          <div className="overflow-x-auto px-4 pb-4">
            <table className="w-full border-collapse text-[13px] text-[var(--shell-content-text)]">
              <thead className="h-11 px-3 text-left text-xs font-medium whitespace-nowrap border-b border-[var(--shell-side-border)] bg-[var(--shell-menu-hover-bg)] text-[var(--shell-group-title)]"><tr>{p.columns.map((x) => <th key={x} className="h-11 px-3 text-left text-xs font-medium whitespace-nowrap border-b border-[var(--shell-side-border)] bg-[var(--shell-menu-hover-bg)] text-[var(--shell-group-title)]">{x}</th>)}</tr></thead>
              <tbody>
                {slice.map((r) => (
                  <tr key={r.id}>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{r.batchNo}</td>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{r.channel}</td>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{fmtFee(r.channelAmount)}</td>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{fmtFee(r.systemAmount)}</td>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{fmtFee(r.diff)}</td>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]"><StatusTag domain="recon" value={r.status} /></td>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">
                      {r.status === 'DIFF_PENDING' ? (
                        <span className="inline-flex items-center">
                          <button disabled={busy} onClick={() => setStmt({ batchNo: r.batchNo, ref: '', amount: '' })}>{p.statementBtn}</button>
                          <span className="mx-1 text-[var(--shell-side-border)]">|</span>
                          <button disabled={busy} onClick={() => settle(r.batchNo)}>{p.settle}</button>
                        </span>
                      ) : fmtTime(r.settledAt ?? '')}
                    </td>
                  </tr>
                ))}
                {!slice.length && <TableStateRow colSpan={7} loading={busy} text={p.empty} />}
              </tbody>
            </table>
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
            <div className="overflow-x-auto px-4 pb-4">
              <table className="w-full border-collapse text-[13px] text-[var(--shell-content-text)]">
                <thead className="h-11 px-3 text-left text-xs font-medium whitespace-nowrap border-b border-[var(--shell-side-border)] bg-[var(--shell-menu-hover-bg)] text-[var(--shell-group-title)]"><tr>{p.ledgerColumns.map((x) => <th key={x} className="h-11 px-3 text-left text-xs font-medium whitespace-nowrap border-b border-[var(--shell-side-border)] bg-[var(--shell-menu-hover-bg)] text-[var(--shell-group-title)]">{x}</th>)}</tr></thead>
                <tbody>
                  {ledgerRows.map((r) => (
                    <tr key={r.billId}>
                      <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{r.billNo}</td>
                      <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{r.customerName}</td>
                      <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{r.legalEntityName}</td>
                      <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{fmtFee(r.billAmount)}</td>
                      <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{fmtFee(r.paidAmount)}</td>
                      <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{fmtFee(r.invoiceAmount)}</td>
                      <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]"><StatusTag domain="ledgerRecon" value={r.diffKind} /></td>
                      <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{r.invoiceNo || '—'}</td>
                      <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{r.taxStatus || '—'}</td>
                    </tr>
                  ))}
                  {!ledgerRows.length && <TableStateRow colSpan={9} loading={busy} text={p.empty} />}
                </tbody>
              </table>
            </div>
          </>
        )}
        <div className="flex justify-end px-4 py-3 text-xs text-[var(--shell-group-title)]">
          {tab === 'channel' ? (
            <Pagination total={rows.length} page={chPage} pageSize={chSize}
              onPage={setChPage} onSize={setChSize} {...pagerTexts(p)} />
          ) : (
            <Pagination total={ledgerTotal} page={lgPage} pageSize={lgSize}
              onPage={setLgPage} onSize={setLgSize} {...pagerTexts(p)} />
          )}
        </div>
      </div>
      {stmt && (
        <Drawer title={p.statementTitle + ' · ' + stmt.batchNo} onClose={() => setStmt(null)}
          footer={
            <>
              <button className="h-8 cursor-pointer rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-4 text-[13px] text-[var(--shell-content-text)] hover:border-[var(--color-border-hover)] hover:text-[var(--shell-heading)]" onClick={() => setStmt(null)}>{t.pages.company.cancel}</button>
              <button className="h-8 cursor-pointer rounded-sm border-none bg-[var(--shell-fab-bg)] px-4 text-[13px] text-[var(--shell-fab-icon)] hover:bg-[var(--shell-fab-bg-hover)]" disabled={busy || !stmt.ref.trim() || !(Number(stmt.amount) > 0)} onClick={saveStatement}>{busy ? t.pages.account.submitting : t.pages.company.save}</button>
            </>
          }>
          <div className="flex flex-col gap-3.5">
            <label className="flex flex-col gap-1.5 text-[13px]">
              <span>{p.statementRef}</span>
              <input className="h-8 rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-2.5 text-[13px] text-[var(--shell-content-text)] outline-none focus:border-[var(--color-border-focus)]" value={stmt.ref} onChange={(e) => setStmt({ ...stmt, ref: e.target.value })} />
            </label>
            <label className="flex flex-col gap-1.5 text-[13px]">
              <span>{p.statementAmount}</span>
              <input className="h-8 rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-2.5 text-[13px] text-[var(--shell-content-text)] outline-none focus:border-[var(--color-border-focus)]" inputMode="decimal" value={stmt.amount} onChange={(e) => setStmt({ ...stmt, amount: e.target.value })} />
            </label>
          </div>
        </Drawer>
      )}
    </div>
  )
}

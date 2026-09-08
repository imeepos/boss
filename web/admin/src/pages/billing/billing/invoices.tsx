// 发票面板(TAX/AG-04):契约 GET /invoices + 作废/重开/人工回填(http_tax.go)。
// 入 billing 页(domain-map 裁定:发票无专用页);manual 通道回填税局票号。
// 状态/属地文案走本页 i18n 映射(components/StatusTag 注册表无 invoice 域,不越界改公共件)。
import { useEffect, useState } from 'react'
import { toast } from 'sonner'
import { apiFetch } from '../../../api/client'
import { useT } from '../../../i18n'
import { Pagination } from '../../../components/Pagination'
import { ResourcePicker } from '../../../components/ResourcePicker'
import { searchCustomers } from '../../../api/pickers'
import { pagerTexts } from '../../org/shared'
import { fmtFee, fmtTime } from '../../../lib/format'
import { Drawer } from '../../../components/Drawer'
import {
  Dialog, DialogContent, DialogFooter, DialogHeader, DialogTitle,
} from '../../../components/ui/dialog'
import { FormField } from '../../../components/business/form-field'
import { Input } from '../../../components/ui/input'
import { SubmitButton, TableStateRow } from '../../../components/business'
import { ActionLink, ErrorBanner, ToolbarButton } from '../../../components/business/page-head'
import { Card, CardFooter } from '../../../components/ui/card'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '../../../components/ui/table'
import type { InvoiceRow } from '../types'
import { useCustomerPin } from '../useCustomerPin'
import { INVOICES_REFRESH } from './run-modal'
import { pageSlice } from '../types'

interface TaxEventRow { id: number; event: string; taxStatusAfter: string; taxNo: string; failReason: string; externalId: string; createdAt: string }

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
  // 税局写口与轨迹(P1-E:对齐 POST tax-submit/retry/replay + GET tax-events)。
  const [taxBusy, setTaxBusy] = useState(false)
  const [eventsFor, setEventsFor] = useState<InvoiceRow | null>(null)
  const [events, setEvents] = useState<TaxEventRow[]>([])

  const taxAct = async (row: InvoiceRow, action: 'tax-submit' | 'tax-retry' | 'tax-replay') => {
    if (taxBusy) return
    setTaxBusy(true)
    try {
      await apiFetch(`/invoices/${row.id}/${action}`, { method: 'POST' })
      toast.success(action === 'tax-submit' ? v.taxSubmitOk : action === 'tax-retry' ? v.taxRetryOk : v.taxReplayOk)
      load()
    } catch (e) {
      // 行内直发动作无弹层承载错误,必须 toast 透出(原实现在 actError,弹层未开时静默)。
      toast.error(e instanceof Error ? e.message : v.actFail)
    } finally {
      setTaxBusy(false)
    }
  }

  const openEvents = async (row: InvoiceRow) => {
    setEventsFor(row)
    setEvents([])
    try {
      const d = await apiFetch<{ items: TaxEventRow[] }>(`/invoices/${row.id}/tax-events`)
      setEvents(d?.items ?? [])
    } catch (e) {
      toast.error(e instanceof Error ? e.message : v.loadFail)
    }
  }

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
      .then(() => {
        toast.success(act.kind === 'void' ? v.voidOk : act.kind === 'reissue' ? v.reissueOk : v.backfillOk)
        setAct(null); setInput(''); load()
      })
      .catch((e) => setActError(e instanceof Error ? e.message : v.actFail))
      .finally(() => setBusy(false))
  }

  const slice = pageSlice(rows, page, pageSize)
  const actText = act?.kind === 'void' ? v.voidConfirm : act?.kind === 'reissue' ? v.reissueConfirm : v.backfillTip
  const statusText = (s: string) => v.statusTexts[s] ?? s
  // 钉选回显(W0 基线交接项):已选客户名经详情接口取,保证触发器不回显裸编号。
  const pinnedCustomer = useCustomerPin(customerId)
  const taxStatusText = (s: string) => v.taxStatusTexts[s] ?? s
  const jurisdictionText = (s: string) => v.jurisdictionTexts[s] ?? s

  return (
    <Card>
      <div className="flex flex-wrap items-center gap-2 p-4">
        <strong>{v.title}</strong>
        <ResourcePicker
          value={customerId}
          onChange={(v) => { setCustomerId(v); setPage(1) }}
          search={searchCustomers}
          toOption={(c) => ({ value: String(c.id), label: `${c.name} · ${c.phone || c.customerCode}` })}
          ariaLabel={v.filterCustomer}
          emptyLabel={t.pages.pickers.common.all}
          searchPlaceholder={t.pages.pickers.common.placeholder}
          errorText={v.loadFail}
          pinnedOptions={pinnedCustomer}
        />
        <span className="spacer" />
        <ToolbarButton disabled={busy} onClick={load}>{t.pages.audit.refresh}</ToolbarButton>
      </div>
      {error ? <ErrorBanner message={error} /> : (
        <div className="px-4 pb-4">
          <Table>
            <TableHeader><TableRow>{v.columns.map((x) => <TableHead key={x}>{x}</TableHead>)}</TableRow></TableHeader>
            <TableBody>
              {slice.map((r) => (
                <TableRow key={r.id}>
                  <TableCell className="break-all font-mono text-xs">{r.invoiceNo}</TableCell>
                  <TableCell>{r.customerName || `#${r.customerId}`}</TableCell>
                  <TableCell>{fmtFee(r.totalAmount)}</TableCell>
                  <TableCell>{statusText(r.status)}</TableCell>
                  <TableCell>{r.taxJurisdiction ? jurisdictionText(r.taxJurisdiction) : '—'}</TableCell>
                  <TableCell>{r.taxNo || (r.taxStatus ? taxStatusText(r.taxStatus) : '—')}</TableCell>
                  <TableCell>
                    <span className="inline-flex flex-wrap items-center">
                      {r.status === 'ISSUED' && <ActionLink onClick={() => { setAct({ kind: 'void', row: r }); setInput(''); setActError('') }} label={v.voidBtn} />}
                      {r.status === 'ISSUED' && r.taxStatus !== 'ISSUED' && <ActionLink onClick={() => { setAct({ kind: 'backfill', row: r }); setInput(''); setActError('') }} label={v.backfillBtn} />}
                      {r.status === 'VOIDED' && <ActionLink onClick={() => { setAct({ kind: 'reissue', row: r }); setActError('') }} label={v.reissueBtn} />}
                      {r.taxStatus === 'PENDING' && (
                        <button disabled={taxBusy} onClick={() => taxAct(r, 'tax-submit')}
                          className="px-1 text-xs text-[var(--color-text-link)] bg-none border-none cursor-pointer hover:underline disabled:cursor-not-allowed disabled:opacity-50">{v.taxSubmitBtn}</button>
                      )}
                      {r.taxStatus === 'FAILED' && (
                        <button disabled={taxBusy} onClick={() => taxAct(r, 'tax-retry')}
                          className="px-1 text-xs text-[var(--color-text-link)] bg-none border-none cursor-pointer hover:underline disabled:cursor-not-allowed disabled:opacity-50">{v.taxRetryBtn}</button>
                      )}
                      {r.taxStatus === 'FAILED' && (
                        <button disabled={taxBusy} onClick={() => taxAct(r, 'tax-replay')}
                          className="px-1 text-xs text-[var(--color-text-link)] bg-none border-none cursor-pointer hover:underline disabled:cursor-not-allowed disabled:opacity-50">{v.taxReplayBtn}</button>
                      )}
                      <ActionLink onClick={() => openEvents(r)} label={v.eventsBtn} />
                    </span>
                  </TableCell>
                </TableRow>
              ))}
              {!slice.length && <TableStateRow colSpan={7} loading={busy} text={v.empty} />}
            </TableBody>
          </Table>
        </div>
      )}
      <CardFooter>
        <Pagination total={rows.length} page={page} pageSize={pageSize}
          onPage={setPage} onSize={setPageSize} {...pagerTexts(t.pages.billPage)} />
      </CardFooter>

      {act && (
        <Dialog open onOpenChange={(open) => { if (!open && !busy) setAct(null) }}>
          <DialogContent className="max-w-sm">
            <DialogHeader>
              <DialogTitle>{act.kind === 'void' ? v.voidBtn : act.kind === 'reissue' ? v.reissueBtn : v.backfillBtn}</DialogTitle>
            </DialogHeader>
            <p className="m-0 text-[13px] leading-6 text-[var(--shell-content-text)]">{actText}</p>
            {act.kind !== 'reissue' && (
              <FormField label={act.kind === 'void' ? v.voidReasonLabel : v.taxNoLabel} required error={actError || undefined}>
                <Input value={input} autoFocus
                  placeholder={act.kind === 'void' ? v.voidReasonPh : v.taxNoPh}
                  onChange={(e) => setInput(e.target.value)} />
              </FormField>
            )}
            {act.kind === 'reissue' && actError && <p className="break-all m-0 text-[13px] text-[var(--color-danger)]">{actError}</p>}
            <DialogFooter>
              <ToolbarButton onClick={() => setAct(null)}>{t.pages.company.cancel}</ToolbarButton>
              <SubmitButton state={busy ? 'loading' : 'idle'} disabled={busy || (act.kind !== 'reissue' && !input.trim())} onClick={run}
                labels={{ idle: v.confirm, loading: t.pages.account.submitting, success: v.confirm, failed: v.confirm }} />
            </DialogFooter>
          </DialogContent>
        </Dialog>
      )}

      {eventsFor && (
        <Drawer title={v.taxEventsTitle + ' · ' + eventsFor.invoiceNo} onClose={() => setEventsFor(null)}>
          {events.length === 0 ? <p className="m-0 text-[13px] text-[var(--shell-group-title)]">{v.taxEventsEmpty}</p> : (
            <ul className="m-0 flex list-none flex-col gap-2 p-0 text-[13px]">
              {events.map((ev) => (
                <li key={ev.id} className="rounded-sm border border-[var(--shell-side-border)] px-3 py-2">
                  <span className="font-medium text-[var(--shell-heading)]">{ev.event}</span>
                  <span className="ml-2 text-[var(--shell-group-title)]">{taxStatusText(ev.taxStatusAfter)}{ev.taxNo ? ' · ' + ev.taxNo : ''}</span>
                  {ev.failReason && <span className="ml-2 text-[var(--color-danger)]">{ev.failReason}</span>}
                  <span className="ml-2 text-[var(--shell-group-title)]">{fmtTime(ev.createdAt)}</span>
                </li>
              ))}
            </ul>
          )}
        </Drawer>
      )}
    </Card>
  )
}

// 报告中心页:契约 GET /reports(items)+ GET /reports/latest?period(正文抽屉)+ POST /reports/:id/send。
import { useEffect, useState } from 'react'
import { apiFetch } from '../../../api/client'
import { useT } from '../../../i18n'
import { PageHead, pagerTexts } from '../../org/shared'
import { Pagination } from '../../../components/Pagination'
import { Drawer } from '../../../components/Drawer'
import { fmtTime } from '../../../lib/format'
import { pageSlice, type ReportPayload, type ReportRow } from '../types'
import { useConfirm } from '../../../components/ConfirmDialog'
import { TableStateRow, EmptyState } from '../../../components/business'

const PERIODS = ['daily', 'weekly', 'monthly', 'quarterly'] as const

export default function ReportPage() {
  const t = useT()
  const confirmDialog = useConfirm()
  const r = t.pages.reportPage
  const [rows, setRows] = useState<ReportRow[]>([])
  const [error, setError] = useState('')
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(10)
  const [busy, setBusy] = useState(false)
  const [view, setView] = useState<ReportPayload | null>(null)
  const [viewError, setViewError] = useState('')
  const [notice, setNotice] = useState('')

  const load = () => {
    setError('')
    setBusy(true)
    apiFetch<{ items: ReportRow[] }>('/reports')
      .then((d) => setRows(d?.items ?? []))
      .catch((e) => setError(e instanceof Error ? e.message : r.loadFail))
      .finally(() => setBusy(false))
  }
  useEffect(load, []) // eslint-disable-line react-hooks/exhaustive-deps

  const viewLatest = (period: string) => {
    setView(null)
    setViewError('')
    apiFetch<{ payload: ReportPayload }>('/reports/latest', { query: { period } })
      .then((d) => setView(d?.payload ?? null))
      .catch(() => setViewError(r.viewFail))
  }

  const send = async (row: ReportRow) => {
    if (busy || !(await confirmDialog(r.sendConfirm.replace('{id}', String(row.id))))) return
    setNotice('')
    setBusy(true)
    apiFetch(`/reports/${row.id}/send`, { method: 'POST' })
      .then(() => setNotice(r.sent.replace('{id}', String(row.id))))
      .catch(() => setNotice(r.sendFail))
      .finally(() => setBusy(false))
  }

  const slice = pageSlice(rows, page, pageSize)
  const periodLabel = (p: string) => {
    const i = PERIODS.indexOf(p as typeof PERIODS[number])
    return i >= 0 ? r.periods[i] : p
  }

  return (
    <div>
      <PageHead title={r.title} desc={r.desc} />
      <div className="mb-4 rounded-md border border-[var(--shell-card-border)] bg-[var(--shell-card-bg)] shadow-[var(--shell-card-shadow)]">
        <div className="flex flex-wrap items-center gap-2 p-4">
          <span className="spacer" />
          {PERIODS.map((p, i) => (
            <button key={p} className="h-8 cursor-pointer rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-4 text-[13px] text-[var(--shell-content-text)] hover:border-[var(--color-border-hover)] hover:text-[var(--shell-heading)]" disabled={busy} onClick={() => viewLatest(p)}>
              {r.view} · {r.periods[i]}
            </button>
          ))}
          <button className="h-8 cursor-pointer rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-4 text-[13px] text-[var(--shell-content-text)] hover:border-[var(--color-border-hover)] hover:text-[var(--shell-heading)]" disabled={busy} onClick={load}>{t.pages.audit.refresh}</button>
          {notice && <span style={{ fontSize: 13, color: '#52c41a' }}>{notice}</span>}
        </div>
        {error ? <div className="mx-4 mb-3 rounded-sm border border-[color-mix(in_srgb,var(--color-danger)_25%,transparent)] bg-[color-mix(in_srgb,var(--color-danger)_8%,transparent)] px-3 py-2 text-[13px] text-[var(--color-danger)]">{error}</div> : (
          <div className="overflow-x-auto px-4 pb-4">
            <table className="w-full border-collapse text-[13px] text-[var(--shell-content-text)]">
              <thead className="h-11 px-3 text-left text-xs font-medium whitespace-nowrap border-b border-[var(--shell-side-border)] bg-[var(--shell-menu-hover-bg)] text-[var(--shell-group-title)]"><tr>{r.columns.map((x) => <th key={x} className="h-11 px-3 text-left text-xs font-medium whitespace-nowrap border-b border-[var(--shell-side-border)] bg-[var(--shell-menu-hover-bg)] text-[var(--shell-group-title)]">{x}</th>)}</tr></thead>
              <tbody>
                {slice.map((x) => (
                  <tr key={x.id}>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">#{x.id}</td>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{periodLabel(x.period)}</td>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{fmtTime(x.windowStart)}</td>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{fmtTime(x.windowEnd)}</td>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{fmtTime(x.createdAt)}</td>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">
                      <span className="inline-flex items-center">
                        <button onClick={() => viewLatest(x.period)}>{r.view}</button>
                        <button disabled={busy} onClick={() => send(x)}>{r.send}</button>
                      </span>
                    </td>
                  </tr>
                ))}
                {!slice.length && <TableStateRow colSpan={6} loading={busy} text={r.empty} />}
              </tbody>
            </table>
          </div>
        )}
        <div className="flex justify-end px-4 py-3 text-xs text-[var(--shell-group-title)]">
          <Pagination total={rows.length} page={page} pageSize={pageSize}
            onPage={setPage} onSize={setPageSize} {...pagerTexts(r)} />
        </div>
      </div>
      {(view || viewError) && (
        <Drawer title={r.viewTitle} onClose={() => { setView(null); setViewError('') }}
          footer={<button className="h-8 cursor-pointer rounded-sm border-none bg-[var(--shell-fab-bg)] px-4 text-[13px] text-[var(--shell-fab-icon)] hover:bg-[var(--shell-fab-bg-hover)]" onClick={() => { setView(null); setViewError('') }}>
            {t.pages.company.cancel}
          </button>}>
          {viewError ? <div className="mx-4 mb-3 rounded-sm border border-[color-mix(in_srgb,var(--color-danger)_25%,transparent)] bg-[color-mix(in_srgb,var(--color-danger)_8%,transparent)] px-3 py-2 text-[13px] text-[var(--color-danger)]">{viewError}</div> : !view ? (
            <EmptyState text={r.viewEmpty} />
          ) : (
            <div className="flex flex-col gap-2.5">
              <div className="flex gap-3 text-[13px]"><span className="w-24 flex-none text-[var(--shell-group-title)]">{r.generatedAtLabel}</span>
                <span className="break-all text-[var(--shell-content-text)]">{fmtTime(view.generatedAt)}</span></div>
              {view.indicators.map((x) => (
                <div className="flex gap-3 text-[13px]" key={x.key}>
                  <span className="w-24 flex-none text-[var(--shell-group-title)]">{x.name || x.key}</span><span className="break-all text-[var(--shell-content-text)]">{x.value} · {x.detail}</span>
                </div>
              ))}
              {view.conclusions && view.conclusions.length > 0 && (
                <div className="flex gap-3 text-[13px]"><span className="w-24 flex-none text-[var(--shell-group-title)]">{r.conclusionsLabel}</span>
                  <span className="break-all text-[var(--shell-content-text)]">{view.conclusions.map((c, i) => <div key={i}>{c}</div>)}</span></div>
              )}
            </div>
          )}
        </Drawer>
      )}
    </div>
  )
}

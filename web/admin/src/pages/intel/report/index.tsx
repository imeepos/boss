// 报告中心页:契约 GET /reports + /reports/latest?period(正文抽屉)+ /reports/history?period
// (B1 后端 trend 接口,前端用 LineTrend SVG 折线渲染)+ POST /reports/:id/send。
// 顶部 4 统计卡 + LineTrend 趋势曲线(本期真趋势!)+ 四周期对比柱状保留作概览;
// 下方报告列表+正文 Drawer。
import { useEffect, useMemo, useState } from 'react'
import { apiFetch } from '../../../api/client'
import { ApiError } from '../../../api/envelope'
import { useT } from '../../../i18n'
import { PageHead, pagerTexts } from '../../org/shared'
import { Pagination } from '../../../components/Pagination'
import { Drawer } from '../../../components/Drawer'
import { fmtTime } from '../../../lib/format'
import { pageSlice, type ReportPayload, type ReportRow } from '../types'
import { useConfirm } from '../../../components/ConfirmDialog'
import { TableStateRow, EmptyState } from '../../../components/business'
import { CardShell, StatCard, LineTrend, StackedBars } from '../../../components/business/charts'
import { formatCurrency, formatIndicatorDetail, formatIndicatorValue } from '../../../components/business/charts/format-value'
import { buildTrendSeries, type TrendSnap } from './trend'

const PERIODS = ['daily', 'weekly', 'monthly', 'quarterly'] as const
// 业务 code 40400 = 该周期尚无快照(后端 reportLatestHandler ErrNoSnapshot)。
const CODE_NOT_FOUND = 40400

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
  const [viewOpen, setViewOpen] = useState(false)
  const [notice, setNotice] = useState('')
  // trend 曲线(B1+B6):选周期 + 取 history
  const [trendPeriod, setTrendPeriod] = useState<string>('daily')
  const [trendSnaps, setTrendSnaps] = useState<{ windowStart: string; payload: ReportPayload }[]>([])
  const [trendBusy, setTrendBusy] = useState(false)

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
    setViewOpen(true)
    apiFetch<{ payload: ReportPayload }>('/reports/latest', { query: { period } })
      .then((d) => setView(d?.payload ?? null))
      .catch((e) => {
        // 尚无该周期快照是空态不是故障:展示"暂无报告正文"。
        if (e instanceof ApiError && e.code === CODE_NOT_FOUND) setView(null)
        else setViewError(r.viewFail)
      })
  }

  // trend 曲线数据拉取(后端 /reports/history?period&limit)
  const loadTrend = (period: string) => {
    setTrendBusy(true)
    apiFetch<{ items: { windowStart: string; payload: ReportPayload }[] }>('/reports/history', { query: { period, limit: 12 } })
      .then((d) => setTrendSnaps(d?.items ?? []))
      .catch(() => setTrendSnaps([]))
      .finally(() => setTrendBusy(false))
  }
  useEffect(() => { loadTrend(trendPeriod) }, [trendPeriod]) // eslint-disable-line react-hooks/exhaustive-deps

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

  // 顶部 4 张卡 + 四周期对比柱状:本期无 trend 历史接口,柱内堆叠的 5 段值
  // 取自 view.payload 或全部 0 兜底(loading 状态)。view 来自用户点击的某一期快照,
  // 故本概览为"快照对照",i18n 文案显式标注 compareDesc"非趋势"。
  const v = view
  const groups = PERIODS.map((_, i) => ({
    stacks: v
      ? [
          v.regionROI?.[i]?.revenue ?? 0,
          v.regionROI?.[i]?.investment ?? 0,
          v.regionROI?.[i]?.roi ?? 0,
          0,
          v.maintenance?.length ?? 0,
        ]
      : [0, 0, 0, 0, 0],
  }))

  // 趋势曲线(B6 + 回归修复):buildTrendSeries 纯函数处理单点字段缺失
  // (regression: B6 第一次用 snap?.regionROI.reduce(...) 在 regionROI=null
  // 的老快照上 .reduce 被 null 调用 → "Cannot read properties of null (reading 'reduce')")。
  const trendSeries = useMemo(
    () => buildTrendSeries(trendSnaps as TrendSnap[], r.compareLegend),
    [trendSnaps, r.compareLegend],
  )
  const trendLabels = useMemo(() => [...trendSnaps].reverse().map((s) => fmtTime(s.windowStart).slice(5, 10)), [trendSnaps])

  return (
    <div>
      <PageHead title={r.title} desc={r.desc} />
      <section className="mb-6 grid grid-cols-1 gap-4 md:grid-cols-2 xl:grid-cols-4">
        <StatCard label={r.columns[0]} value={rows.length} />
        <StatCard label={r.periods[3]} value={rows.filter((x) => x.period === 'quarterly').length} />
        <StatCard label={r.generatedAtLabel} value={rows.length > 0 ? fmtTime(rows[0].createdAt).split(' ')[0] : '—'} />
        <StatCard label={r.periods[2]} value={rows.filter((x) => x.period === 'monthly').length} />
      </section>

      <CardShell className="mb-4">
        <div className="mb-1 flex items-center justify-between">
          <h3 className="m-0 text-base font-semibold text-[var(--shell-heading)]">{r.trendTitle}</h3>
          <div className="flex items-center gap-2">
            <span className="text-[12px] text-[var(--shell-group-title)]">{r.trendDesc}</span>
            <select className="h-8 rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-2 text-[13px] text-[var(--shell-content-text)]"
              value={trendPeriod} onChange={(e) => setTrendPeriod(e.target.value)} disabled={trendBusy}>
              {PERIODS.map((p, i) => <option key={p} value={p}>{r.periods[i]}</option>)}
            </select>
          </div>
        </div>
        {trendSnaps.length >= 2 ? (
          <LineTrend labels={trendLabels} series={trendSeries} />
        ) : (
          <div className="flex h-48 items-center justify-center text-[13px] text-[var(--shell-group-title)]">{r.trendEmpty}</div>
        )}
      </CardShell>

      <CardShell className="mb-4">
        <div className="mb-1 flex items-center justify-between">
          <h3 className="m-0 text-base font-semibold text-[var(--shell-heading)]">{r.compareTitle}</h3>
          <span className="text-[12px] text-[var(--shell-group-title)]">{r.compareDesc}</span>
        </div>
        <StackedBars groups={groups} legends={r.compareLegend} formatValue={formatCurrency} />
      </CardShell>

      <CardShell className="mb-4">
        <div className="mb-3 flex flex-wrap items-center gap-2">
          <span className="flex-1" />
          {PERIODS.map((p, i) => (
            <button key={p} className="h-8 cursor-pointer rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-4 text-[13px] text-[var(--shell-content-text)] hover:border-[var(--color-border-hover)] hover:text-[var(--shell-heading)]" disabled={busy} onClick={() => viewLatest(p)}>
              {r.view} · {r.periods[i]}
            </button>
          ))}
          <button className="h-8 cursor-pointer rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-4 text-[13px] text-[var(--shell-content-text)] hover:border-[var(--color-border-hover)] hover:text-[var(--shell-heading)]" disabled={busy} onClick={load}>{t.pages.audit.refresh}</button>
          {notice && <span className="text-[13px] text-[var(--color-success)]">{notice}</span>}
        </div>
        {error ? <div className="mx-2 mb-2 rounded-sm border border-[color-mix(in_srgb,var(--color-danger)_25%,transparent)] bg-[color-mix(in_srgb,var(--color-danger)_8%,transparent)] px-3 py-2 text-[13px] text-[var(--color-danger)]">{error}</div> : (
          <div className="overflow-x-auto">
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
                      <span className="inline-flex items-center gap-2">
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
        <div className="flex justify-end pt-3 text-xs text-[var(--shell-group-title)]">
          <Pagination total={rows.length} page={page} pageSize={pageSize}
            onPage={setPage} onSize={setPageSize} {...pagerTexts(r)} />
        </div>
      </CardShell>

      {viewOpen && (
        <Drawer title={r.viewTitle} onClose={() => { setView(null); setViewError(''); setViewOpen(false) }}
          footer={<button className="h-8 cursor-pointer rounded-sm border-none bg-[var(--shell-fab-bg)] px-4 text-[13px] text-[var(--shell-fab-icon)] hover:bg-[var(--shell-fab-bg-hover)]" onClick={() => { setView(null); setViewError(''); setViewOpen(false) }}>
            {t.pages.company.cancel}
          </button>}>
          {viewError ? <div className="mx-4 mb-3 rounded-sm border border-[color-mix(in_srgb,var(--color-danger)_25%,transparent)] bg-[color-mix(in_srgb,var(--color-danger)_8%,transparent)] px-3 py-2 text-[13px] text-[var(--color-danger)]">{viewError}</div> : !view ? (
            <EmptyState text={r.viewEmpty} />
          ) : (
            <div className="flex flex-col gap-2.5">
              <div className="flex gap-3 text-[13px]"><span className="w-24 flex-none text-[var(--shell-group-title)]">{r.generatedAtLabel}</span>
                <span className="break-all text-[var(--shell-content-text)]">{fmtTime(view.generatedAt)}</span></div>
              {(view.indicators ?? []).map((x) => (
                <div className="flex gap-3 text-[13px]" key={x.key}>
                  <span className="w-24 flex-none text-[var(--shell-group-title)]">{x.name || x.key}</span><span className="break-all text-[var(--shell-content-text)]">{formatIndicatorValue(x.key, x.value)} · {formatIndicatorDetail(x.detail)}</span>
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
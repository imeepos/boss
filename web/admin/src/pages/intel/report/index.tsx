// 报告中心页:契约 GET /reports + /reports/latest?period(正文抽屉)+ /reports/history?period
// (B1 后端 trend 接口,前端用 LineTrend SVG 折线渲染)+ POST /reports/:id/send。
// 打磨:列表周期/关键词客户端筛选、统计卡语义化、推送逐行等待态、
// 成败走 sonner toast(失败带原因可复制)、抽屉加载/错误态、对比卡自动取最新快照。
import { useEffect, useMemo, useState } from 'react'
import { toast } from 'sonner'
import { apiFetch } from '../../../api/client'
import { ApiError } from '../../../api/envelope'
import { useT } from '../../../i18n'
import { PageHead, pagerTexts } from '../../org/shared'
import { ErrorBanner } from '../../../components/business/page-head'
import { Pagination } from '../../../components/Pagination'
import { Dropdown } from '../../../components/Dropdown'
import { Drawer } from '../../../components/Drawer'
import { Table, TableHeader, TableBody, TableRow, TableHead, TableCell } from '../../../components/ui/table'
import { fmtTime } from '../../../lib/format'
import { pageSlice, type ReportPayload, type ReportRow } from '../types'
import { useConfirm } from '../../../components/ConfirmDialog'
import { TableStateRow, EmptyState, LoadingState, ToolbarButton } from '../../../components/business'
import { copyText } from '../../../components/business/feedback'
import { CardShell, StatCard, LineTrend, StackedBars } from '../../../components/business/charts'
import { formatCurrency, formatIndicatorDetail, formatIndicatorValue } from '../../../components/business/charts/format-value'
import { buildTrendSeries, type TrendSnap } from './trend'
import { PERIODS, periodLabel, filterReports } from './filter'
import { Input } from '../../../components/ui/input'

// 业务 code 40400 = 该周期尚无快照(后端 reportLatestHandler ErrNoSnapshot)。
const CODE_NOT_FOUND = 40400
const Q_DEBOUNCE_MS = 300

const errText = (e: unknown, fallback: string): string =>
  e instanceof Error && e.message ? e.message : fallback

/** 对比卡快照:周期键 + 生成时间 + 正文。 */
interface CompareSnap { period: string; generatedAt: string; payload: ReportPayload }

export default function ReportPage() {
  const t = useT()
  const confirmDialog = useConfirm()
  const r = t.pages.reportPage
  const [rows, setRows] = useState<ReportRow[]>([])
  const [error, setError] = useState('')
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(10)
  const [busy, setBusy] = useState(false)
  const [sendingId, setSendingId] = useState<number | null>(null)
  const [view, setView] = useState<ReportPayload | null>(null)
  const [viewError, setViewError] = useState('')
  const [viewLoading, setViewLoading] = useState(false)
  const [viewOpen, setViewOpen] = useState(false)
  // 列表筛选(客户端;/reports 无服务端筛选参数)
  const [keyword, setKeyword] = useState('')
  const [q, setQ] = useState('')
  const [periodFilter, setPeriodFilter] = useState('')
  // trend 曲线(B1+B6):选周期 + 取 history
  const [trendPeriod, setTrendPeriod] = useState<string>('daily')
  const [trendSnaps, setTrendSnaps] = useState<{ windowStart: string; payload: ReportPayload }[]>([])
  const [trendBusy, setTrendBusy] = useState(false)
  // 对比卡快照:挂载取最新日报;用户查看某期正文后随周期联动
  const [compareSnap, setCompareSnap] = useState<CompareSnap | null>(null)

  const load = () => {
    setError('')
    setBusy(true)
    apiFetch<{ items: ReportRow[] }>('/reports')
      .then((d) => setRows(d?.items ?? []))
      .catch((e) => setError(errText(e, r.loadFail)))
      .finally(() => setBusy(false))
  }
  useEffect(load, []) // eslint-disable-line react-hooks/exhaustive-deps

  const fetchLatest = (period: string) =>
    apiFetch<{ payload: ReportPayload }>('/reports/latest', { query: { period } })

  const viewLatest = (period: string) => {
    setView(null)
    setViewError('')
    setViewLoading(true)
    setViewOpen(true)
    fetchLatest(period)
      .then((d) => {
        const payload = d?.payload ?? null
        setView(payload)
        if (payload) setCompareSnap({ period, generatedAt: payload.generatedAt, payload })
      })
      .catch((e) => {
        // 尚无该周期快照是空态不是故障:展示"暂无报告正文"。
        if (e instanceof ApiError && e.code === CODE_NOT_FOUND) setView(null)
        else setViewError(errText(e, r.viewFail))
      })
      .finally(() => setViewLoading(false))
  }

  // 对比卡兜底:挂载取最新日报快照;无快照属空态,卡片保持占位不报错。
  useEffect(() => {
    fetchLatest('daily')
      .then((d) => {
        const p = d?.payload
        if (p) setCompareSnap({ period: 'daily', generatedAt: p.generatedAt, payload: p })
      })
      .catch(() => { /* 无日报快照 = 对比卡空态占位 */ })
  }, []) // eslint-disable-line react-hooks/exhaustive-deps

  // trend 曲线数据拉取(后端 /reports/history?period&limit)
  const loadTrend = (period: string) => {
    setTrendBusy(true)
    apiFetch<{ items: { windowStart: string; payload: ReportPayload }[] }>('/reports/history', { query: { period, limit: 12 } })
      .then((d) => setTrendSnaps(d?.items ?? []))
      .catch(() => setTrendSnaps([]))
      .finally(() => setTrendBusy(false))
  }
  useEffect(() => { loadTrend(trendPeriod) }, [trendPeriod]) // eslint-disable-line react-hooks/exhaustive-deps

  // 关键词防抖下发(客户端过滤),并回第一页
  useEffect(() => {
    const h = setTimeout(() => { setQ(keyword.trim()); setPage(1) }, Q_DEBOUNCE_MS)
    return () => clearTimeout(h)
  }, [keyword])

  const send = async (row: ReportRow) => {
    if (sendingId !== null) return
    if (!(await confirmDialog(r.sendConfirm.replace('{id}', String(row.id))))) return
    setSendingId(row.id)
    apiFetch(`/reports/${row.id}/send`, { method: 'POST' })
      .then(() => toast.success(r.sent.replace('{id}', String(row.id))))
      .catch((e) => {
        const reason = errText(e, r.sendFail)
        toast.error(r.sendFail, {
          description: reason,
          action: { label: t.common.copy, onClick: () => { copyText(reason) } },
        })
      })
      .finally(() => setSendingId(null))
  }

  // 筛选收缩导致当前页空:回缩到最后非空页
  const filtered = useMemo(() => filterReports(rows, periodFilter, q, r), [rows, periodFilter, q, r])
  useEffect(() => {
    const max = Math.max(1, Math.ceil(filtered.length / pageSize))
    if (page > max) setPage(max)
  }, [filtered.length, pageSize, page])
  const slice = pageSlice(filtered, page, pageSize)

  // 趋势曲线(B6 + 回归修复):buildTrendSeries 纯函数处理单点字段缺失。
  const trendSeries = useMemo(
    () => buildTrendSeries(trendSnaps as TrendSnap[], r.compareLegend),
    [trendSnaps, r.compareLegend],
  )
  const trendLabels = useMemo(() => [...trendSnaps].reverse().map((s) => fmtTime(s.windowStart).slice(5, 10)), [trendSnaps])

  // 四周期对比柱状:取对比卡快照(挂载=最新日报;查看正文后联动),无快照全 0 占位。
  const cp = compareSnap?.payload
  const groups = PERIODS.map((_, i) => ({
    stacks: cp
      ? [
          cp.regionROI?.[i]?.revenue ?? 0,
          cp.regionROI?.[i]?.investment ?? 0,
          cp.regionROI?.[i]?.roi ?? 0,
          0,
          cp.maintenance?.length ?? 0,
        ]
      : [0, 0, 0, 0, 0],
  }))

  const periodOptions = [{ value: '', label: r.filterAll }].concat(
    PERIODS.map((p, i) => ({ value: p, label: r.periods[i] })))

  return (
    <div>
      <PageHead title={r.title} desc={r.desc} />
      <section className="mb-6 grid grid-cols-1 gap-4 md:grid-cols-2 xl:grid-cols-4">
        <StatCard label={r.statTotal} value={rows.length} />
        <StatCard label={r.statQuarterly} value={rows.filter((x) => x.period === 'quarterly').length} />
        <StatCard label={r.statMonthly} value={rows.filter((x) => x.period === 'monthly').length} />
        <StatCard label={r.statLatest} value={rows.length > 0 ? fmtTime(rows[0].createdAt).split(' ')[0] : '—'} />
      </section>

      <CardShell className="mb-4">
        <div className="mb-1 flex items-center justify-between">
          <h3 className="m-0 text-base font-semibold text-[var(--shell-heading)]">{r.trendTitle}</h3>
          <div className="flex items-center gap-2">
            <span className="text-[12px] text-[var(--shell-group-title)]">{r.trendDesc}</span>
            <Dropdown value={trendPeriod} ariaLabel={r.trendPeriodLabel} disabled={trendBusy}
              options={periodOptions}
              onChange={(v) => setTrendPeriod(v)} />
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
          {compareSnap && (
            <span className="text-[12px] text-[var(--shell-group-title)]">
              {r.compareCurrent.replace('{period}', periodLabel(compareSnap.period, r)).replace('{time}', fmtTime(compareSnap.generatedAt))}
            </span>
          )}
        </div>
        <StackedBars groups={groups} legends={r.compareLegend} formatValue={formatCurrency} />
      </CardShell>

      <CardShell className="mb-4">
        <div className="mb-3 flex flex-wrap items-center gap-2">
          <Input className="w-52" placeholder={r.searchPlaceholder} aria-label={r.searchPlaceholder}
            value={keyword} onChange={(e) => setKeyword(e.target.value)} />
          <Dropdown value={periodFilter} ariaLabel={r.filterPeriod} disabled={busy}
            options={periodOptions}
            onChange={(v) => { setPeriodFilter(v); setPage(1) }} />
          <span className="flex-1" />
          {PERIODS.map((p, i) => (
            <ToolbarButton key={p} disabled={busy} onClick={() => viewLatest(p)}>
              {r.view} · {r.periods[i]}
            </ToolbarButton>
          ))}
          <ToolbarButton disabled={busy} onClick={load}>{t.pages.audit.refresh}</ToolbarButton>
        </div>
        {error ? <ErrorBanner message={error} className="mx-2 mb-2" /> : (
          <Table>
            <TableHeader>
              <TableRow>{r.columns.map((x) => <TableHead key={x}>{x}</TableHead>)}</TableRow>
            </TableHeader>
            <TableBody>
              {slice.map((x) => (
                <TableRow key={x.id}>
                  <TableCell>#{x.id}</TableCell>
                  <TableCell>{periodLabel(x.period, r)}</TableCell>
                  <TableCell>{fmtTime(x.windowStart)}</TableCell>
                  <TableCell>{fmtTime(x.windowEnd)}</TableCell>
                  <TableCell>{fmtTime(x.createdAt)}</TableCell>
                  <TableCell>
                    <span className="inline-flex items-center gap-2">
                      <button className="h-7 cursor-pointer rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-2 text-[12px] text-[var(--shell-content-text)] hover:border-[var(--color-border-hover)] hover:text-[var(--shell-heading)] disabled:cursor-not-allowed disabled:opacity-50" disabled={sendingId !== null} onClick={() => viewLatest(x.period)}>{r.view}</button>
                      <button className="h-7 cursor-pointer rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-2 text-[12px] text-[var(--shell-content-text)] hover:border-[var(--color-border-hover)] hover:text-[var(--shell-heading)] disabled:cursor-not-allowed disabled:opacity-50" disabled={sendingId !== null} onClick={() => send(x)}>
                        {sendingId === x.id ? r.sending : r.send}
                      </button>
                    </span>
                  </TableCell>
                </TableRow>
              ))}
              {!slice.length && <TableStateRow colSpan={6} loading={busy} text={r.empty} />}
            </TableBody>
          </Table>
        )}
        <div className="flex justify-end pt-3 text-xs text-[var(--shell-group-title)]">
          <Pagination total={filtered.length} page={page} pageSize={pageSize}
            onPage={setPage} onSize={setPageSize} {...pagerTexts(r)} />
        </div>
      </CardShell>

      {viewOpen && (
        <Drawer title={r.viewTitle} onClose={() => { setView(null); setViewError(''); setViewOpen(false) }}
          footer={<button className="h-8 cursor-pointer rounded-sm border-none bg-[var(--shell-fab-bg)] px-4 text-[13px] text-[var(--shell-fab-icon)] hover:bg-[var(--shell-fab-bg-hover)]" onClick={() => { setView(null); setViewError(''); setViewOpen(false) }}>
            {t.pages.company.cancel}
          </button>}>
          {viewError ? (
            <ErrorBanner message={viewError} className="mx-4 mb-3" />
          ) : viewLoading ? (
            <LoadingState />
          ) : !view ? (
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
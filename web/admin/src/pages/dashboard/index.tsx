// 工作台:GET /dashboard 聚合真实数据(统计卡/订单状态分布/待办/趋势)。
// 体验打磨(2026-09-07):首屏加载态、失败可复制可重试、刷新等待与更新时间、
// 待办编号与正文拆分、分布条按可见最大值归一、趋势标题随周期联动、欢迎语用角色名。
import { useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import type { Profile } from '../../api/auth'
import { apiFetch } from '../../api/client'
import { useT } from '../../i18n'
import { PageHead } from '../../components/business/page-head'
import { CopyButton, EmptyState, LoadingState, Spinner } from '../../components/business/feedback'
import { CardShell, OrderTrend, StatCard, type Trend } from '../../components/business/charts'
import { Dropdown } from '../../components/Dropdown'
import { Button } from '../../components/ui/button'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '../../components/ui/table'
import { StatusTag } from '../../components/StatusTag'
import { fmtTime } from '../../lib/format'
import { todoTarget } from './todoTarget'
import { splitSubject } from './todoSubject'

interface StatCardDto { key: string; label: string; value: string; delta: string; trend: string }
interface StatusDist { status: string; statusLabel: string; count: number; percent: string }
interface TodoItem { todoId: number; subject: string; source: string; time: string }
interface TrendSeriesDto { status: string; statusLabel: string; values: number[] }
interface DashboardData {
  stats: StatCardDto[]
  orderStatusDist: StatusDist[]
  todos: { items: TodoItem[] }
  trend: { days: string[]; series: TrendSeriesDto[] }
}

const TODO_PAGE_SIZE = 5

const STAT_CARD_TARGETS: Record<string, string> = {
  todayOrders: '/boss/order?created=today',
  activeTickets: '/boss/dispatch?status=DOING',
  pendingAlarms: '/alarm?status=OPEN',
  assetConsistency: '/quad/check?status=CONFLICT',
}

const VALID_TREND: ReadonlySet<Trend> = new Set<Trend>(['up', 'down', 'flat'])
const TREND_COLORS = ['#D5A63A', '#1677FF', '#722ED1', '#52C41A', '#8C8C8C']
const TREND_PERIODS = ['week', 'month', 'quarter', 'year', 'all'] as const

// 下钻箭头:有跳转目标的统计卡右上角常驻,提示可点击。
function DrillIcon() {
  return (
    <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" aria-hidden>
      <path d="m9 18 6-6-6-6" />
    </svg>
  )
}

export default function DashboardPage({ profile }: { profile: Profile }) {
  const t = useT()
  const navigate = useNavigate()
  const d = t.pages.dashboard
  const [data, setData] = useState<DashboardData | null>(null)
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)
  const [loadedAt, setLoadedAt] = useState('')
  const [todoPage, setTodoPage] = useState(1)
  const [trendPeriod, setTrendPeriod] = useState<(typeof TREND_PERIODS)[number]>('week')

  const load = (period = trendPeriod) => {
    setError('')
    setBusy(true)
    apiFetch<DashboardData>('/dashboard', { query: { trendPeriod: period } })
      .then((v) => { setData(v); setLoadedAt(new Date().toISOString()) })
      .catch((e) => setError(e instanceof Error ? e.message : d.loadFail))
      .finally(() => setBusy(false))
  }
  useEffect(() => { load() }, []) // eslint-disable-line react-hooks/exhaustive-deps

  const changeTrendPeriod = (value: string) => {
    if (!TREND_PERIODS.includes(value as (typeof TREND_PERIODS)[number])) return
    const period = value as (typeof TREND_PERIODS)[number]
    setTrendPeriod(period)
    void load(period)
  }

  const welcome = d.welcome.replace('{name}', profile.realName).replace('{role}', profile.roleName || profile.roleCode)
  const todoItems = data?.todos.items ?? []
  const totalTodoPages = Math.ceil(todoItems.length / TODO_PAGE_SIZE)
  const pagedTodos = todoItems.slice((todoPage - 1) * TODO_PAGE_SIZE, todoPage * TODO_PAGE_SIZE)
  const trendLabels = data?.trend.days ?? []
  const trendSeries = (data?.trend.series ?? []).map((item, index) => ({
    key: item.status,
    label: item.statusLabel,
    color: TREND_COLORS[index % TREND_COLORS.length],
    values: item.values,
  }))
  // 分布条按可见集合内最大数量归一(原样渲染会让小占比全部贴 0 不可辨),真实占比以文本列为准。
  const maxDistCount = Math.max(0, ...(data?.orderStatusDist ?? []).map((r) => r.count))

  const loadButton = (label: string) => (
    <Button size="sm" disabled={busy} onClick={() => { void load() }}>
      {busy ? (<span className="inline-flex items-center gap-2"><Spinner size={14} />{d.refreshing}</span>) : label}
    </Button>
  )

  return (
    <div>
      <div className="mb-4 flex flex-wrap items-start justify-between gap-3">
        <PageHead title={d.title} desc={welcome} />
        <div className="flex items-center gap-3 pt-1.5">
          {loadedAt && !busy && (
            <span aria-live="polite" className="text-xs text-[var(--shell-crumb-text)]">
              {d.updatedAt.replace('{time}', fmtTime(loadedAt).split(' ')[1] || '')}
            </span>
          )}
          {loadButton(d.refresh)}
        </div>
      </div>
      {error ? (
        <div role="alert" className="mb-4 flex flex-wrap items-center gap-3 rounded-md border border-destructive/25 bg-destructive/10 px-4 py-3 text-sm text-destructive">
          <span className="min-w-0 flex-1 break-all">{error}</span>
          <CopyButton text={error} />
          {loadButton(d.retry)}
        </div>
      ) : !data ? (
        <LoadingState />
      ) : (
        <>
          <section className="mb-6 grid grid-cols-1 gap-4 md:grid-cols-2 xl:grid-cols-4">
            {data.stats.map((s) => {
              const target = STAT_CARD_TARGETS[s.key]
              return (
                <StatCard
                  key={s.key}
                  label={s.label}
                  value={s.value}
                  delta={s.delta}
                  trend={VALID_TREND.has(s.trend as Trend) ? (s.trend as Trend) : 'flat'}
                  icon={target ? <DrillIcon /> : undefined}
                  title={target ? d.statHint : undefined}
                  onClick={target ? () => navigate(target) : undefined}
                />
              )
            })}
          </section>

          <section className="mb-6 grid grid-cols-1 gap-4 xl:grid-cols-2">
            <CardShell title={d.distTitle}>
              <div className="overflow-x-auto">
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead>{d.colStatus}</TableHead>
                      <TableHead>{d.colCount}</TableHead>
                      <TableHead title={d.percentHint}>{d.colPercent}</TableHead>
                      <TableHead>{d.colProgress}</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {data.orderStatusDist.map((r) => (
                      <TableRow
                        key={r.status}
                        className="cursor-pointer"
                        title={d.distRowHint}
                        onClick={() => navigate('/boss/order?status=' + encodeURIComponent(r.status))}
                        onKeyDown={(e) => {
                          if (e.key === 'Enter' || e.key === ' ') {
                            e.preventDefault()
                            navigate('/boss/order?status=' + encodeURIComponent(r.status))
                          }
                        }}
                        tabIndex={0}
                        role="link"
                      >
                        <TableCell>
                          <StatusTag domain="order" value={r.status} />
                        </TableCell>
                        <TableCell>{r.count}</TableCell>
                        <TableCell>{r.percent}</TableCell>
                        <TableCell>
                          <div className="h-1.5 min-w-[100px] overflow-hidden rounded-sm bg-muted">
                            <div
                              className="h-full rounded-sm bg-primary transition-[width] duration-300"
                              style={{ width: maxDistCount > 0 ? Math.round((r.count / maxDistCount) * 100) + '%' : '0%' }}
                            />
                          </div>
                        </TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              </div>
            </CardShell>

            <CardShell>
              <div className="-mt-5 mb-4 flex items-center justify-between">
                <div className="flex items-baseline">
                  <h3 className="m-0 text-base font-semibold text-[var(--shell-heading)]">{d.todoTitle}</h3>
                  {todoItems.length > 0 && (
                    <span className="ml-2 text-xs text-muted-foreground">
                      {d.todoCount.replace('{n}', String(todoItems.length))}
                    </span>
                  )}
                </div>
                {totalTodoPages > 1 && (
                  <div className="flex items-center gap-2">
                    <Button variant="outline" size="sm" disabled={todoPage === 1} onClick={() => setTodoPage((p) => Math.max(1, p - 1))}>
                      {t.pages.company.prev}
                    </Button>
                    <span className="text-xs text-muted-foreground">{todoPage} / {totalTodoPages}</span>
                    <Button variant="outline" size="sm" disabled={todoPage === totalTodoPages} onClick={() => setTodoPage((p) => Math.min(totalTodoPages, p + 1))}>
                      {t.pages.company.next}
                    </Button>
                  </div>
                )}
              </div>
              <div className="max-h-[320px] overflow-y-auto">
                {todoItems.length === 0 && <EmptyState text={d.todoEmpty} />}
                {pagedTodos.map((it) => {
                  const time = it.time ? fmtTime(it.time).split(' ')[1] || fmtTime(it.time) : '—'
                  // 待办处理入口:source 经 todoTarget 解析为已注册菜单路由,未注册不跳转。
                  const target = todoTarget(it.source)
                  const parts = splitSubject(it.subject)
                  return (
                    <button
                      key={it.todoId}
                      className="flex w-full cursor-pointer items-start gap-3 border-b border-border bg-transparent border-0 p-0 py-2.5 text-left last:border-b-0 hover:bg-muted/50"
                      onClick={() => { if (target) navigate(target) }}
                      title={target ? d.todoGo : it.subject}
                    >
                      <div className="w-14 flex-shrink-0 pt-px text-xs text-muted-foreground">{time}</div>
                      <div className="min-w-0 flex-1">
                        <div className="line-clamp-2 text-[13px] leading-relaxed text-[var(--shell-content-text)]" title={it.subject}>
                          {parts.id && <span className="mr-1.5 font-mono text-[11px] text-muted-foreground">{parts.id}</span>}
                          {parts.text}
                        </div>
                        <span className="mt-0.5 inline-block rounded-sm bg-primary/10 px-1.5 py-px text-[11px] font-medium text-primary">{it.source}</span>
                      </div>
                      {target && <span className="flex-shrink-0 self-center text-xs text-primary hover:underline">{d.todoGo}</span>}
                    </button>
                  )
                })}
              </div>
            </CardShell>
          </section>

          <CardShell>
            <div className="mb-4 flex flex-wrap items-center justify-between gap-3">
              <h3 className="m-0 text-base font-semibold text-[var(--shell-heading)]">{d.trendTitles[trendPeriod]}</h3>
              <div className="flex items-center gap-2">
                <span className="text-xs text-[var(--shell-group-title)]">{d.trendPeriod}</span>
                <Dropdown
                  value={trendPeriod}
                  options={TREND_PERIODS.map((value) => ({ value, label: d.trendPeriods[value] }))}
                  onChange={changeTrendPeriod}
                  ariaLabel={d.trendPeriod}
                  disabled={busy}
                  triggerStyle={{ minWidth: 116 }}
                />
              </div>
            </div>
            <OrderTrend
              labels={trendLabels}
              series={trendSeries}
              valueUnit={d.trendUnit}
              tooltipLabel={d.trendTooltip}
              emptyText={d.empty}
              statusToggleLabel={d.trendStatusToggle}
              interactionLabels={{ previous: d.trendPrevious, next: d.trendNext, zoomOut: d.trendZoomOut, zoomIn: d.trendZoomIn, reset: d.trendReset }}
            />
          </CardShell>
        </>
      )}
    </div>
  )
}
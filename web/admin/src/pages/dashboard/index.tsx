// 工作台:GET /dashboard 聚合真实数据(统计卡/订单状态分布/待办/近7日趋势)
// 样式:tailwind 原子类 + ui/button + business/charts 共享组件。
import { useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import type { Profile } from '../../api/auth'
import { apiFetch } from '../../api/client'
import { useT } from '../../i18n'
import { PageHead } from '../../components/business/page-head'
import { CardShell, OrderTrend, StatCard, type Trend } from '../../components/business/charts'
import { Dropdown } from '../../components/Dropdown'
import { Button } from '../../components/ui/button'
import { StatusTag } from '../../components/StatusTag'
import { fmtTime } from '../../lib/format'
import { todoTarget } from './todoTarget'

interface StatCardDto { key: string; label: string; value: string; delta: string; trend: string }
interface StatusDist { status: string; statusLabel: string; count: number; percent: string }
interface TodoItem { todoId: number; subject: string; source: string; time: string }
interface DashboardData {
  stats: StatCardDto[]
  orderStatusDist: StatusDist[]
  todos: { items: TodoItem[] }
  trend: { days: string[]; values: number[] }
}

const TODO_PAGE_SIZE = 5

const VALID_TREND: ReadonlySet<Trend> = new Set<Trend>(['up', 'down', 'flat'])
const TREND_PERIODS = ['week', 'month', 'quarter', 'year', 'all'] as const

export default function DashboardPage({ profile }: { profile: Profile }) {
  const t = useT()
  const navigate = useNavigate()
  const d = t.pages.dashboard
  const [data, setData] = useState<DashboardData | null>(null)
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)
  const [todoPage, setTodoPage] = useState(1)
  const [trendPeriod, setTrendPeriod] = useState<(typeof TREND_PERIODS)[number]>('week')

  const load = (period = trendPeriod) => {
    setError('')
    setBusy(true)
    apiFetch<DashboardData>('/dashboard', { query: { trendPeriod: period } })
      .then((v) => setData(v))
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

  const welcome = d.welcome.replace('{name}', profile.realName).replace('{role}', profile.roleCode)
  const todoItems = data?.todos.items ?? []
  const totalTodoPages = Math.ceil(todoItems.length / TODO_PAGE_SIZE)
  const pagedTodos = todoItems.slice((todoPage - 1) * TODO_PAGE_SIZE, todoPage * TODO_PAGE_SIZE)
  const trendLabels = data?.trend.days ?? []
  const trendValues = data?.trend.values ?? []

  return (
    <div>
      <PageHead title={d.title} desc={welcome} />
      <div className="mb-6 flex items-center gap-2">
        <Button size="sm" disabled={busy} onClick={() => { void load() }}>{t.pages.audit.refresh}</Button>
      </div>
      {error ? (
        <div className="mb-4 rounded-md border border-destructive/25 bg-destructive/10 px-4 py-3 text-sm text-destructive">
          {error}
        </div>
      ) : data ? (
        <>
          <section className="mb-6 grid grid-cols-1 gap-4 md:grid-cols-2 xl:grid-cols-4">
            {data.stats.map((s) => (
              <StatCard
                key={s.key}
                label={s.label}
                value={s.value}
                delta={s.delta}
                trend={VALID_TREND.has(s.trend as Trend) ? (s.trend as Trend) : 'flat'}
              />
            ))}
          </section>

          <section className="mb-6 grid grid-cols-1 gap-4 xl:grid-cols-2">
            <CardShell title={d.distTitle}>
              <div className="overflow-x-auto">
                <table className="w-full border-collapse text-[13px]">
                  <thead>
                    <tr>
                      <th className="border-b border-border px-3 py-2 text-left font-medium text-muted-foreground">{d.colStatus}</th>
                      <th className="border-b border-border px-3 py-2 text-left font-medium text-muted-foreground">{d.colCount}</th>
                      <th className="border-b border-border px-3 py-2 text-left font-medium text-muted-foreground">{d.colPercent}</th>
                      <th className="border-b border-border px-3 py-2 text-left font-medium text-muted-foreground">{d.colProgress}</th>
                    </tr>
                  </thead>
                  <tbody>
                    {data.orderStatusDist.map((r) => (
                      <tr key={r.status}>
                        <td className="border-b border-border px-3 py-2.5 text-[var(--shell-content-text)]">
                          <StatusTag domain="order" value={r.status} />
                        </td>
                        <td className="border-b border-border px-3 py-2.5 text-[var(--shell-content-text)]">{r.count}</td>
                        <td className="border-b border-border px-3 py-2.5 text-[var(--shell-content-text)]">{r.percent}</td>
                        <td className="border-b border-border px-3 py-2.5">
                          <div className="h-1.5 min-w-[100px] overflow-hidden rounded-sm bg-muted">
                            <div className="h-full rounded-sm bg-primary transition-[width] duration-300" style={{ width: r.percent }} />
                          </div>
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
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
                {todoItems.length === 0 && (
                  <p className="m-0 py-8 text-center text-sm text-muted-foreground">{d.empty}</p>
                )}
                {pagedTodos.map((it) => {
                  const time = it.time ? fmtTime(it.time).split(' ')[1] || fmtTime(it.time) : '--'
                  // 待办处理入口:source 经 todoTarget 解析为已注册菜单路由,未注册不跳转。
                  const target = todoTarget(it.source)
                  return (
                    <button
                      key={it.todoId}
                      className="flex w-full cursor-pointer items-start gap-3 border-b border-border bg-transparent border-0 p-0 py-2.5 text-left last:border-b-0 hover:bg-muted/50"
                      onClick={() => { if (target) navigate(target) }}
                      title={d.todoGo}
                    >
                      <div className="w-14 flex-shrink-0 text-xs text-muted-foreground">{time}</div>
                      <div className="flex-1">
                        <span className="text-[13px] leading-relaxed text-[var(--shell-content-text)]">{it.subject}</span>
                        <span className="ml-1.5 inline-block rounded-sm bg-primary/10 px-1.5 py-px text-[11px] font-medium align-middle text-primary">
                          {it.source}
                        </span>
                      </div>
                      <span className="flex-shrink-0 self-center text-xs text-primary hover:underline">{d.todoGo}</span>
                    </button>
                  )
                })}
              </div>
            </CardShell>
          </section>

          <CardShell>
            <div className="mb-4 flex flex-wrap items-center justify-between gap-3">
              <h3 className="m-0 text-base font-semibold text-[var(--shell-heading)]">{d.trendTitle}</h3>
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
              values={trendValues}
              valueUnit={d.trendUnit}
              tooltipLabel={d.trendTooltip}
              emptyText={d.empty}
            />
          </CardShell>
        </>
      ) : null}
    </div>
  )
}

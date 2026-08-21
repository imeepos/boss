// 工作台:GET /dashboard 聚合真实数据(统计卡/订单状态分布/待办/近7日趋势)
// 样式:tailwind 原子类 + ui/button + business/page-head,已移除 dashboard.css。
import { useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import type { Profile } from '../../api/auth'
import { apiFetch } from '../../api/client'
import { useT } from '../../i18n'
import { PageHead } from '../../components/business/page-head'
import { Button } from '../../components/ui/button'
import { StatusTag } from '../../components/StatusTag'
import { fmtTime } from '../../lib/format'

interface StatCard { key: string; label: string; value: string; delta: string; trend: string }
interface StatusDist { status: string; statusLabel: string; count: number; percent: string }
interface TodoItem { todoId: number; subject: string; source: string; time: string }
interface DashboardData {
  stats: StatCard[]
  orderStatusDist: StatusDist[]
  todos: { items: TodoItem[] }
  trend: { days: string[]; values: number[] }
}

const TODO_PAGE_SIZE = 5

const TREND_CLASS = {
  up: 'text-[var(--color-danger)]',
  down: 'text-[var(--color-success)]',
  flat: 'text-muted-foreground',
} as const

export default function DashboardPage({ profile }: { profile: Profile }) {
  const t = useT()
  const navigate = useNavigate()
  const d = t.pages.dashboard
  const [data, setData] = useState<DashboardData | null>(null)
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)
  const [todoPage, setTodoPage] = useState(1)

  const load = () => {
    setError('')
    setBusy(true)
    apiFetch<DashboardData>('/dashboard')
      .then((v) => setData(v))
      .catch((e) => setError(e instanceof Error ? e.message : d.loadFail))
      .finally(() => setBusy(false))
  }
  useEffect(load, []) // eslint-disable-line react-hooks/exhaustive-deps

  const welcome = d.welcome.replace('{name}', profile.realName).replace('{role}', profile.roleCode)
  const maxTrend = Math.max(1, ...(data?.trend.values ?? [1]))
  const todoItems = data?.todos.items ?? []
  const totalTodoPages = Math.ceil(todoItems.length / TODO_PAGE_SIZE)
  const pagedTodos = todoItems.slice((todoPage - 1) * TODO_PAGE_SIZE, todoPage * TODO_PAGE_SIZE)

  return (
    <div>
      <PageHead title={d.title} desc={welcome} />
      <div className="mb-6 flex items-center gap-2">
        <Button size="sm" disabled={busy} onClick={load}>{t.pages.audit.refresh}</Button>
      </div>
      {error ? (
        <div className="mb-4 rounded-md border border-destructive/25 bg-destructive/10 px-4 py-3 text-sm text-destructive">
          {error}
        </div>
      ) : data ? (
        <>
          <section className="mb-6 grid grid-cols-1 gap-4 md:grid-cols-2 xl:grid-cols-4">
            {data.stats.map((s) => (
              <div
                key={s.key}
                className="rounded-md border border-[var(--shell-card-border)] bg-[var(--shell-card-bg)] p-5 shadow-[var(--shell-card-shadow)]"
              >
                <div className="mb-2 text-sm text-[var(--shell-content-text)]">{s.label}</div>
                <div className="text-[28px] font-semibold leading-tight text-[var(--shell-heading)]">{s.value}</div>
                {s.delta ? <div className={'mt-1 text-[13px] ' + (TREND_CLASS[s.trend as keyof typeof TREND_CLASS] ?? TREND_CLASS.flat)}>{s.delta}</div> : null}
              </div>
            ))}
          </section>

          <section className="mb-6 grid grid-cols-1 gap-4 xl:grid-cols-2">
            <article className="rounded-md border border-[var(--shell-card-border)] bg-[var(--shell-card-bg)] p-5 shadow-[var(--shell-card-shadow)]">
              <h3 className="mb-4 text-base font-semibold text-[var(--shell-heading)]">{d.distTitle}</h3>
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
            </article>

            <article className="rounded-md border border-[var(--shell-card-border)] bg-[var(--shell-card-bg)] p-5 shadow-[var(--shell-card-shadow)]">
              <div className="mb-4 flex items-center justify-between">
                <h3 className="m-0 text-base font-semibold text-[var(--shell-heading)]">{d.todoTitle}</h3>
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
                {pagedTodos.map((it) => {
                  const time = it.time ? fmtTime(it.time).split(' ')[1] || fmtTime(it.time) : '--'
                  // 待办处理入口:派单池待指派跳派单管理(默认即派单池页签),告警跳告警中心。
                  const target = it.source === '派单池' ? '/boss/dispatch' : '/alarm'
                  return (
                    <button
                      key={it.todoId}
                      className="flex w-full cursor-pointer items-start gap-3 border-b border-border bg-transparent border-0 p-0 py-2.5 text-left last:border-b-0 hover:bg-muted/50"
                      onClick={() => navigate(target)}
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
            </article>
          </section>

          <section className="rounded-md border border-[var(--shell-card-border)] bg-[var(--shell-card-bg)] p-5 shadow-[var(--shell-card-shadow)]">
            <h3 className="mb-4 text-base font-semibold text-[var(--shell-heading)]">{d.trendTitle}</h3>
            <div className="flex h-40 items-end gap-3 overflow-x-auto py-5">
              {(data.trend.days ?? []).map((day, i) => (
                <div key={day + i} className="flex h-full min-w-12 flex-col items-center justify-end gap-2" title={`${day}: ${data.trend.values[i]}`}>
                  <div className="text-xs font-medium text-muted-foreground">{data.trend.values[i]}</div>
                  <div
                    className="min-h-1 w-3/5 max-w-8 rounded-t-sm bg-primary transition-[height] duration-300"
                    style={{ height: `${(data.trend.values[i] / maxTrend) * 100}%` }}
                  />
                  <span className="text-xs text-muted-foreground">{day.slice(5)}</span>
                </div>
              ))}
            </div>
          </section>
        </>
      ) : null}
    </div>
  )
}

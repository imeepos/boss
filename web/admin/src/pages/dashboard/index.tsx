// 工作台:GET /dashboard 聚合真实数据(统计卡/订单状态分布/待办/近7日趋势)
import { useEffect, useState } from 'react'
import type { Profile } from '../../api/auth'
import { apiFetch } from '../../api/client'
import { useT } from '../../i18n'
import { PageHead } from '../org/shared'
import { fmtTime } from '../../lib/format'
import { Button } from 'antd'
import './dashboard.css'

interface StatCard { key: string; label: string; value: string; delta: string; trend: string }
interface StatusDist { status: string; statusLabel: string; count: number; percent: string }
interface TodoItem { todoId: number; subject: string; source: string; time: string }
interface DashboardData {
  stats: StatCard[]
  orderStatusDist: StatusDist[]
  todos: { items: TodoItem[] }
  trend: { days: string[]; values: number[] }
}

const STATUS_COLORS: Record<string, string> = {
  PENDING: '#1677ff',
  RESERVED: '#fa8c16',
  INSTALLING: '#fa8c16',
  DONE: '#52c41a',
  CANCELLED: '#ff4d4f',
}

const TODO_BADGE_COLORS: Record<string, string> = {
  '派单池': '#1677ff',
  '告警中心': '#ff4d4f',
}

const TODO_PAGE_SIZE = 5

export default function DashboardPage({ profile }: { profile: Profile }) {
  const t = useT()
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
    <div className="dash-page">
      <PageHead title={d.title} desc={welcome} />
      <div className="dash-toolbar">
        <Button type="primary" onClick={load} loading={busy}>刷新</Button>
      </div>
      {error ? (
        <div className="dash-error-msg">{error}</div>
      ) : data ? (
        <>
          <section className="dash-section dash-stats">
            {data.stats.map((s) => (
              <div key={s.key} className="dash-stat-card">
                <div className="dash-stat-label">{s.label}</div>
                <div className="dash-stat-value">{s.value}</div>
                {s.delta ? <div className={'dash-delta ' + s.trend}>{s.delta}</div> : null}
              </div>
            ))}
          </section>

          <section className="dash-section dash-grid-2">
            <article className="dash-card">
              <h3 className="dash-card-title">{d.distTitle}</h3>
              <div className="dash-table-wrap">
                <table className="dash-table">
                  <thead>
                    <tr>
                      <th>{d.colStatus}</th>
                      <th>{d.colCount}</th>
                      <th>{d.colPercent}</th>
                      <th>{d.colProgress}</th>
                    </tr>
                  </thead>
                  <tbody>
                    {data.orderStatusDist.map((r) => (
                      <tr key={r.status}>
                        <td>
                          <span
                            className="dash-status-tag"
                            style={{
                              background: getStatusBg(r.status),
                              color: getStatusColor(r.status),
                              borderColor: getStatusBorder(r.status),
                            }}
                          >
                            {r.statusLabel}
                          </span>
                        </td>
                        <td>{r.count}</td>
                        <td>{r.percent}</td>
                        <td>
                          <div className="dash-progress">
                            <div
                              className="dash-progress-fill"
                              style={{
                                width: r.percent,
                                background: STATUS_COLORS[r.status] || '#1677ff',
                              }}
                            />
                          </div>
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            </article>

            <article className="dash-card">
              <div className="dash-card-header">
                <h3 className="dash-card-title">{d.todoTitle}</h3>
                {totalTodoPages > 1 && (
                  <div className="dash-pager">
                    <button
                      type="button"
                      className="dash-pager-btn"
                      disabled={todoPage === 1}
                      onClick={() => setTodoPage((p) => Math.max(1, p - 1))}
                    >
                      上一页
                    </button>
                    <span className="dash-pager-info">
                      {todoPage} / {totalTodoPages}
                    </span>
                    <button
                      type="button"
                      className="dash-pager-btn"
                      disabled={todoPage === totalTodoPages}
                      onClick={() => setTodoPage((p) => Math.min(totalTodoPages, p + 1))}
                    >
                      下一页
                    </button>
                  </div>
                )}
              </div>
              <div className="dash-todo-list">
                {pagedTodos.map((it) => {
                  const time = it.time
                    ? fmtTime(it.time).split(' ')[1] || fmtTime(it.time)
                    : '--'
                  const badgeColor = TODO_BADGE_COLORS[it.source] || '#1677ff'
                  return (
                    <div key={it.todoId} className="dash-todo-item">
                      <div className="dash-todo-time">{time}</div>
                      <div className="dash-todo-content">
                        <span className="dash-todo-text">{it.subject}</span>
                        <span
                          className="dash-todo-badge"
                          style={{ background: badgeColor + '15', color: badgeColor }}
                        >
                          {it.source}
                        </span>
                      </div>
                    </div>
                  )
                })}
              </div>
            </article>
          </section>

          <section className="dash-section dash-card">
            <h3 className="dash-card-title">{d.trendTitle}</h3>
            <div className="dash-trend">
              {(data.trend.days ?? []).map((day, i) => (
                <div key={day + i} className="dash-trend-col" title={`${day}: ${data.trend.values[i]}`}>
                  <div className="dash-trend-value">{data.trend.values[i]}</div>
                  <div
                    className="dash-trend-bar"
                    style={{ height: `${(data.trend.values[i] / maxTrend) * 100}%` }}
                  />
                  <span>{day.slice(5)}</span>
                </div>
              ))}
            </div>
          </section>
        </>
      ) : null}
    </div>
  )
}

function getStatusBg(status: string): string {
  const map: Record<string, string> = {
    PENDING: 'var(--token-color-primary-bg, #e6f7ff)',
    RESERVED: 'var(--token-color-warning-bg, #fff7e6)',
    INSTALLING: 'var(--token-color-warning-bg, #fff7e6)',
    DONE: 'var(--token-color-success-bg, #f6ffed)',
    CANCELLED: 'var(--token-color-error-bg, #fff1f0)',
  }
  return map[status] || 'var(--token-color-fill-tertiary, #f5f5f5)'
}

function getStatusColor(status: string): string {
  const map: Record<string, string> = {
    PENDING: 'var(--token-color-primary, #1677ff)',
    RESERVED: 'var(--token-color-warning, #fa8c16)',
    INSTALLING: 'var(--token-color-warning, #fa8c16)',
    DONE: 'var(--token-color-success, #52c41a)',
    CANCELLED: 'var(--token-color-error, #ff4d4f)',
  }
  return map[status] || 'var(--token-color-text, #666)'
}

function getStatusBorder(status: string): string {
  const map: Record<string, string> = {
    PENDING: 'var(--token-color-primary-border, #91caff)',
    RESERVED: 'var(--token-color-warning-border, #ffd591)',
    INSTALLING: 'var(--token-color-warning-border, #ffd591)',
    DONE: 'var(--token-color-success-border, #b7eb8f)',
    CANCELLED: 'var(--token-color-error-border, #ffa39e)',
  }
  return map[status] || 'var(--token-color-border, #d9d9d9)'
}
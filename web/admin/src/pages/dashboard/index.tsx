// 工作台:GET /dashboard 聚合真实数据(统计卡/订单状态分布/待办/近7日趋势),纯 CSS 无图表库。
import { useEffect, useState } from 'react'
import type { Profile } from '../../api/auth'
import { apiFetch } from '../../api/client'
import { useT } from '../../i18n'
import { PageHead } from '../org/shared'
import { fmtTime } from '../../lib/format'
import '../org/org.css'

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
    <div>
      <PageHead title={d.title} desc={welcome} />
      <div className="org-toolbar">
        <button className="org-btn" disabled={busy} onClick={load}>{t.pages.audit.refresh}</button>
      </div>
      {error ? <div className="org-card"><div className="org-error">{error}</div></div> : data && (
        <>
          {/* 统计卡片 */}
          <div className="dash-stats">
            {data.stats.map((s, i) => (
              <div key={s.key} className="dash-stat">
                <div className="dash-stat-icon" style={{ background: getStatIconBg(i), color: getStatIconColor(i) }}>
                  {getStatIcon(i)}
                </div>
                <div className="dash-stat-label">{s.label}</div>
                <div className="dash-stat-value">{s.value}</div>
                {s.delta ? <div className={'dash-delta ' + s.trend}>{s.delta}</div> : null}
              </div>
            ))}
          </div>

          {/* 订单状态分布 + 待办事项 */}
          <div className="dash-grid-2">
            <div className="org-card">
              <h3>订单状态分布</h3>
              <div className="org-table-wrap">
                <table className="org-table">
                  <thead><tr><th>状态</th><th>数量</th><th>占比</th><th>进度</th></tr></thead>
                  <tbody>
                    {data.orderStatusDist.map((r) => (
                      <tr key={r.status}>
                        <td>
                          <span className="dash-status-tag" style={{ 
                            background: getStatusBg(r.status), 
                            color: getStatusColor(r.status),
                            borderColor: getStatusBorder(r.status)
                          }}>
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
                                background: STATUS_COLORS[r.status] || '#1677ff'
                              }} 
                            />
                          </div>
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            </div>

            <div className="org-card">
              <h3>我的待办 <span className="dash-todo-count">共 {todoItems.length} 条</span></h3>
              <div className="dash-todo-list">
                {pagedTodos.map((it) => {
                  const time = it.time ? (fmtTime(it.time).split(' ')[1] || fmtTime(it.time)) : '--'
                  const badgeColor = TODO_BADGE_COLORS[it.source] || '#1677ff'
                  return (
                    <div key={it.todoId} className="dash-todo-item">
                      <div className="dash-todo-time">{time}</div>
                      <div className="dash-todo-content">
                        {it.subject}
                        <span className="dash-todo-badge" style={{ background: badgeColor + '15', color: badgeColor }}>
                          {it.source}
                        </span>
                      </div>
                    </div>
                  )
                })}
              </div>
              {totalTodoPages > 1 && (
                <div className="dash-todo-pager">
                  <button 
                    className="dash-todo-page-btn" 
                    disabled={todoPage === 1}
                    onClick={() => setTodoPage((p) => Math.max(1, p - 1))}
                  >
                    上一页
                  </button>
                  <span className="dash-todo-page-info">{todoPage} / {totalTodoPages}</span>
                  <button 
                    className="dash-todo-page-btn" 
                    disabled={todoPage === totalTodoPages}
                    onClick={() => setTodoPage((p) => Math.min(totalTodoPages, p + 1))}
                  >
                    下一页
                  </button>
                </div>
              )}
            </div>
          </div>

          {/* 近7日趋势 */}
          <div className="org-card">
            <h3>近 7 日下单趋势</h3>
            <div className="dash-trend">
              {(data.trend.days ?? []).map((day, i) => (
                <div key={day + i} className="dash-trend-col" title={`${day}: ${data.trend.values[i]}`}>
                  <div className="dash-trend-value">{data.trend.values[i]}</div>
                  <div className="dash-trend-bar" style={{ height: `${(data.trend.values[i] / maxTrend) * 100}%` }} />
                  <span>{day.slice(5)}</span>
                </div>
              ))}
            </div>
          </div>
        </>
      )}
    </div>
  )
}

function getStatDotColor(key: string): string {
  const map: Record<string, string> = {
    todayOrders: '#1890ff',
    activeTickets: '#fa8c16',
    pendingAlarms: '#ff4d4f',
    assetConsistency: '#52c41a',
  }
  return map[key] || '#1890ff'
}

function getStatusBg(status: string): string {
  const map: Record<string, string> = {
    PENDING: '#e6f7ff',
    RESERVED: '#fff7e6',
    INSTALLING: '#fff7e6',
    DONE: '#f6ffed',
    CANCELLED: '#fff1f0',
  }
  return map[status] || '#f5f5f5'
}

function getStatusColor(status: string): string {
  const map: Record<string, string> = {
    PENDING: '#1890ff',
    RESERVED: '#fa8c16',
    INSTALLING: '#fa8c16',
    DONE: '#52c41a',
    CANCELLED: '#ff4d4f',
  }
  return map[status] || '#666'
}

function getStatusBorder(status: string): string {
  const map: Record<string, string> = {
    PENDING: '#91caff',
    RESERVED: '#ffd591',
    INSTALLING: '#ffd591',
    DONE: '#b7eb8f',
    CANCELLED: '#ffa39e',
  }
  return map[status] || '#d9d9d9'
}

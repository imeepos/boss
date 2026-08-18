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

export default function DashboardPage({ profile }: { profile: Profile }) {
  const t = useT()
  const d = t.pages.dashboard
  const [data, setData] = useState<DashboardData | null>(null)
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)

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

  return (
    <div>
      <PageHead title={d.title} desc={welcome} />
      <div className="org-toolbar">
        <button className="org-btn" disabled={busy} onClick={load}>{t.pages.audit.refresh}</button>
      </div>
      {error ? <div className="org-card"><div className="org-error">{error}</div></div> : data && (
        <div className="org-card">
          <h3>{d.statTitle}</h3>
          <div className="dash-stats">
            {data.stats.map((s) => (
              <div key={s.key} className="dash-stat">
                <div className="dash-stat-label">{s.label}</div>
                <div className="dash-stat-value">{s.value}</div>
                {s.delta ? <div className={'dash-delta ' + s.trend}>{s.delta}</div> : null}
              </div>
            ))}
          </div>

          <h3>{d.distTitle}</h3>
          <div className="org-table-wrap">
            <table className="org-table">
              <thead><tr><th>{t.pages.dashboard.distTitle}</th><th>%</th></tr></thead>
              <tbody>
                {data.orderStatusDist.map((r) => (
                  <tr key={r.status}>
                    <td>{r.statusLabel}({r.count})</td>
                    <td>
                      <span className="dash-bar"><span style={{ width: r.percent }} /></span> {r.percent}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>

          <h3>{d.todoTitle}</h3>
          {data.todos.items.length === 0 ? <p>{d.empty}</p> : (
            <ul className="dash-todos">
              {data.todos.items.map((it) => (
                <li key={it.todoId}>{fmtTime(it.time)} · {it.subject}</li>
              ))}
            </ul>
          )}

          <h3>{d.trendTitle}</h3>
          <div className="dash-trend">
            {(data.trend.days ?? []).map((day, i) => (
              <div key={day + i} className="dash-trend-col" title={`${day}: ${data.trend.values[i]}`}>
                <div className="dash-trend-bar" style={{ height: `${(data.trend.values[i] / maxTrend) * 100}%` }} />
                <span>{day.slice(5)}</span>
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  )
}

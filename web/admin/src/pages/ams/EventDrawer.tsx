// 标签/资产事件时间轴抽屉(P2-T4 消费面,仿资产轨迹抽屉交互):
// 一行一事件(时间/操作人/动作徽标/对象/变更),changed JSONB 只渲染实际变化的键
// (键: 旧值 → 新值,等宽字体),不 dump 全量 JSON、不引 diff 库。
// 契约 GET /tags/{tagId}/events、/assets/{assetId}/events(id 倒序,limit 上限 100)。
import { useEffect, useState } from 'react'
import { apiFetch } from '../../api/client'
import { Drawer } from '../../components/Drawer'
import { useT } from '../../i18n'
import { fmtTime } from '../../lib/format'
import { EmptyState } from '../../components/business'
import type { TagEventRow } from './types'
import { changedPairs } from './opsRules'

const ACTION_COLOR: Record<string, string> = {
  BIND: 'var(--color-success)',
  UNBIND: 'var(--color-warning)',
  RECYCLE: 'var(--color-danger)',
}

export function EventDrawer({ kind, id, code, onClose }: { kind: 'tag' | 'asset'; id: number; code: string; onClose: () => void }) {
  const t = useT()
  const g = t.pages.eventOps
  const [rows, setRows] = useState<TagEventRow[] | null>(null)
  const [error, setError] = useState('')

  useEffect(() => {
    const base = kind === 'tag' ? '/tags/' + id + '/events' : '/assets/' + id + '/events'
    apiFetch<{ items: TagEventRow[] }>(base + '?limit=100')
      .then((d) => setRows(d?.items ?? []))
      .catch(() => setError(g.loadFail))
  }, [kind, id]) // eslint-disable-line react-hooks/exhaustive-deps

  const title = (kind === 'tag' ? g.tagEventsTitle : g.assetEventsTitle) + ' · ' + code
  const cols = [g.evTime, g.evAction, g.evActor, g.evObject, g.evChanged]
  const badge = (action: string) => {
    const color = ACTION_COLOR[action] ?? 'var(--shell-group-title)'
    const label = action === 'BIND' ? g.actBIND : action === 'UNBIND' ? g.actUNBIND : action === 'RECYCLE' ? g.actRECYCLE : action
    return (
      <span className="inline-flex items-center rounded-full border px-2 py-0.5 text-xs whitespace-nowrap"
        style={{ color, borderColor: 'color-mix(in srgb, ' + color + ' 45%, transparent)', background: 'color-mix(in srgb, ' + color + ' 10%, transparent)' }}>{label}</span>
    )
  }
  return (
    <Drawer title={title} onClose={onClose} width={760}
      footer={<button className="h-8 cursor-pointer rounded-sm border-none bg-[var(--shell-fab-bg)] px-4 text-[13px] text-[var(--shell-fab-icon)] hover:bg-[var(--shell-fab-bg-hover)]" onClick={onClose}>{t.pages.company.cancel}</button>}>
      {error ? (
        <div className="mx-4 mb-3 rounded-sm border border-[color-mix(in_srgb,var(--color-danger)_25%,transparent)] bg-[color-mix(in_srgb,var(--color-danger)_8%,transparent)] px-3 py-2 text-[13px] text-[var(--color-danger)]">{error}</div>
      ) : (
        <div className="overflow-x-auto px-4 pb-4">
          <table className="w-full border-collapse text-[13px] text-[var(--shell-content-text)]">
            <thead><tr>{cols.map((x) => <th key={x} className="h-11 px-3 text-left text-xs font-medium whitespace-nowrap border-b border-[var(--shell-side-border)] bg-[var(--shell-menu-hover-bg)] text-[var(--shell-group-title)]">{x}</th>)}</tr></thead>
            <tbody>
              {(rows ?? []).map((r) => {
                const obj = r.action === 'RECYCLE' ? (r.tagId ? g.evTag + ' #' + r.tagId : '-') : r.assetId ? g.evAsset + ' #' + r.assetId : '-'
                return (
                  <tr key={r.id}>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{fmtTime(r.createdAt)}</td>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] hover:bg-[var(--shell-menu-hover-bg)]">{badge(r.action)}</td>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{r.actorAccountId ? '#' + r.actorAccountId : '-'}</td>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{obj}</td>
                    <td className="h-11 px-3 border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">
                      {changedPairs(r.changed).map((p) => (
                        <div key={p.key} className="font-mono text-xs whitespace-nowrap">{p.key}: {p.from} → {p.to}</div>
                      ))}
                      {r.detail ? <div className="text-xs text-[var(--shell-group-title)]">{r.detail}</div> : null}
                    </td>
                  </tr>
                )
              })}
              {rows !== null && !rows.length && (
                <tr><td colSpan={5} className="h-11 px-3 border-b border-[var(--shell-side-border)]"><EmptyState text={g.empty} /></td></tr>
              )}
            </tbody>
          </table>
        </div>
      )}
    </Drawer>
  )
}

// 资产状态轨迹 + 持有台账双页签抽屉:GET /assets/:assetId/lifecycle、/assets/:assetId/assignments。
import { useEffect, useState } from 'react'
import { apiFetch } from '../../../api/client'
import { Drawer } from '../../../components/Drawer'
import { StatusTag } from '../../../components/StatusTag'
import { useT } from '../../../i18n'
import { fmtTime } from '../../../lib/format'
import type { AssetRow, AssignmentRow, LifecycleRow } from '../types'
import { EmptyState } from '../../../components/business'

export function AssetTrailDrawer({
  asset, onClose,
}: { asset: AssetRow; onClose: () => void }) {
  const t = useT()
  const a = t.pages.assetPage
  const [tab, setTab] = useState<'lifecycle' | 'assignment'>('lifecycle')
  const [lifecycle, setLifecycle] = useState<LifecycleRow[] | null>(null)
  const [assignments, setAssignments] = useState<AssignmentRow[] | null>(null)
  const [error, setError] = useState('')

  useEffect(() => {
    apiFetch<{ items: LifecycleRow[] }>(`/assets/${asset.assetId}/lifecycle`)
      .then((d) => setLifecycle(d?.items ?? []))
      .catch(() => setError(a.loadFail))
    apiFetch<{ items: AssignmentRow[] }>(`/assets/${asset.assetId}/assignments`)
      .then((d) => setAssignments(d?.items ?? []))
      .catch(() => setAssignments([]))
  }, [asset.assetId]) // eslint-disable-line react-hooks/exhaustive-deps

  const head = (key: string, label: string) => (
    <button key={key} onClick={() => setTab(key as 'lifecycle' | 'assignment')}
      style={{
        padding: '8px 16px', fontSize: 14, cursor: 'pointer', background: 'none', border: 'none',
        borderBottom: tab === key ? '2px solid #1677ff' : '2px solid transparent',
        color: tab === key ? '#1677ff' : '#666', fontWeight: tab === key ? 600 : 400,
      }}>
      {label}
    </button>
  )

  return (
    <Drawer title={`${a.lifecycleTitle} · ${asset.assetCode}`} onClose={onClose}
      footer={<button className="h-8 cursor-pointer rounded-sm border-none bg-[var(--shell-fab-bg)] px-4 text-[13px] text-[var(--shell-fab-icon)] hover:bg-[var(--shell-fab-bg-hover)]" onClick={onClose}>{t.pages.company.cancel}</button>}>
      <div style={{ display: 'flex', gap: 4, marginBottom: 12, borderBottom: '1px solid #f0f0f0' }}>
        {head('lifecycle', a.lifecycle)}
        {head('assignment', a.assignment)}
      </div>
      {error ? <div className="mx-4 mb-3 rounded-sm border border-[color-mix(in_srgb,var(--color-danger)_25%,transparent)] bg-[color-mix(in_srgb,var(--color-danger)_8%,transparent)] px-3 py-2 text-[13px] text-[var(--color-danger)]">{error}</div> : tab === 'lifecycle' ? (
        <div className="overflow-x-auto px-4 pb-4">
          <table className="w-full border-collapse text-[13px] text-[var(--shell-content-text)]">
            <thead className="h-11 px-3 text-left text-xs font-medium whitespace-nowrap border-b border-[var(--shell-side-border)] bg-[var(--shell-menu-hover-bg)] text-[var(--shell-group-title)]"><tr>{a.lifecycleColumns.map((x) => <th key={x} className="h-11 px-3 text-left text-xs font-medium whitespace-nowrap border-b border-[var(--shell-side-border)] bg-[var(--shell-menu-hover-bg)] text-[var(--shell-group-title)]">{x}</th>)}</tr></thead>
            <tbody>
              {(lifecycle ?? []).map((r) => (
                <tr key={r.id}>
                  <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{fmtTime(r.changedAt)}</td>
                  <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]"><StatusTag domain="asset" value={r.status} /></td>
                  <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{r.addressName || (r.addressId ? `#${r.addressId}` : '—')}</td>
                  <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{r.workerName || (r.workerId ? `#${r.workerId}` : '—')}</td>
                  <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{r.workerName || '—'}</td>
                </tr>
              ))}
              {lifecycle !== null && !lifecycle.length && (
                <tr><td colSpan={5} className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]"><EmptyState text={a.empty} /></td></tr>
              )}
            </tbody>
          </table>
        </div>
      ) : (
        <div className="overflow-x-auto px-4 pb-4">
          <table className="w-full border-collapse text-[13px] text-[var(--shell-content-text)]">
            <thead className="h-11 px-3 text-left text-xs font-medium whitespace-nowrap border-b border-[var(--shell-side-border)] bg-[var(--shell-menu-hover-bg)] text-[var(--shell-group-title)]"><tr>{a.assignmentColumns.map((x) => <th key={x} className="h-11 px-3 text-left text-xs font-medium whitespace-nowrap border-b border-[var(--shell-side-border)] bg-[var(--shell-menu-hover-bg)] text-[var(--shell-group-title)]">{x}</th>)}</tr></thead>
            <tbody>
              {(assignments ?? []).map((r) => (
                <tr key={r.id}>
                  <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{r.workerName || (r.workerId ? `#${r.workerId}` : '—')}</td>
                  <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{r.addressName || (r.addressId ? `#${r.addressId}` : '—')}</td>
                  <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{r.reason || '—'}</td>
                  <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{fmtTime(r.effectiveFrom)}</td>
                  <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{r.effectiveTo ? fmtTime(r.effectiveTo) : '至今'}</td>
                </tr>
              ))}
              {assignments !== null && !assignments.length && (
                <tr><td colSpan={5} className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]"><EmptyState text={a.empty} /></td></tr>
              )}
            </tbody>
          </table>
        </div>
      )}
    </Drawer>
  )
}

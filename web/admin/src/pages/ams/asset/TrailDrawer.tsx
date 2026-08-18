// 资产状态轨迹 + 持有台账双页签抽屉:GET /assets/:assetId/lifecycle、/assets/:assetId/assignments。
import { useEffect, useState } from 'react'
import { apiFetch } from '../../../api/client'
import { Drawer } from '../../../components/Drawer'
import { StatusTag } from '../../../components/StatusTag'
import { useT } from '../../../i18n'
import { fmtTime } from '../../../lib/format'
import type { AssetRow, AssignmentRow, LifecycleRow } from '../types'

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
      footer={<button className="org-btn org-btn-primary" onClick={onClose}>{t.pages.company.cancel}</button>}>
      <div style={{ display: 'flex', gap: 4, marginBottom: 12, borderBottom: '1px solid #f0f0f0' }}>
        {head('lifecycle', a.lifecycle)}
        {head('assignment', a.assignment)}
      </div>
      {error ? <div className="org-error">{error}</div> : tab === 'lifecycle' ? (
        <div className="org-table-wrap">
          <table className="org-table">
            <thead><tr>{a.lifecycleColumns.map((x) => <th key={x}>{x}</th>)}</tr></thead>
            <tbody>
              {(lifecycle ?? []).map((r) => (
                <tr key={r.id}>
                  <td>{fmtTime(r.changedAt)}</td>
                  <td><StatusTag domain="asset" value={r.status} /></td>
                  <td>{r.addressName || (r.addressId ? `#${r.addressId}` : '—')}</td>
                  <td>{r.workerName || (r.workerId ? `#${r.workerId}` : '—')}</td>
                  <td>{r.workerName || '—'}</td>
                </tr>
              ))}
              {lifecycle !== null && !lifecycle.length && (
                <tr><td colSpan={5}><div className="org-empty">{a.empty}</div></td></tr>
              )}
            </tbody>
          </table>
        </div>
      ) : (
        <div className="org-table-wrap">
          <table className="org-table">
            <thead><tr>{a.assignmentColumns.map((x) => <th key={x}>{x}</th>)}</tr></thead>
            <tbody>
              {(assignments ?? []).map((r) => (
                <tr key={r.id}>
                  <td>{r.workerName || (r.workerId ? `#${r.workerId}` : '—')}</td>
                  <td>{r.addressName || (r.addressId ? `#${r.addressId}` : '—')}</td>
                  <td>{r.reason || '—'}</td>
                  <td>{fmtTime(r.effectiveFrom)}</td>
                  <td>{r.effectiveTo ? fmtTime(r.effectiveTo) : '至今'}</td>
                </tr>
              ))}
              {assignments !== null && !assignments.length && (
                <tr><td colSpan={5}><div className="org-empty">{a.empty}</div></td></tr>
              )}
            </tbody>
          </table>
        </div>
      )}
    </Drawer>
  )
}

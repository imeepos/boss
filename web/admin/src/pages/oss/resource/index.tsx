// 端口台账页:列名以 fields.md §4.2 为准;契约 GET /resources + GET /ports?resourceId。
import { useEffect, useState } from 'react'
import { apiFetch } from '../../../api/client'
import { useT } from '../../../i18n'
import { PageHead, pagerTexts } from '../../org/shared'
import { StatusTag } from '../../../components/StatusTag'
import { Pagination } from '../../../components/Pagination'
import { Drawer } from '../../../components/Drawer'
import { fmtTime } from '../../../lib/format'
import { pageSlice, type PortHistoryRow, type PortRow, type ResourceRow } from '../types'
import '../../org/org.css'

export default function ResourcePage() {
  const t = useT()
  const r = t.pages.resourcePage
  const [devices, setDevices] = useState<ResourceRow[]>([])
  const [rows, setRows] = useState<PortRow[]>([])
  const [error, setError] = useState('')
  const [resourceId, setResourceId] = useState(0)
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(10)
  const [busy, setBusy] = useState(false)
  const [history, setHistory] = useState<PortRow | null>(null)
  const [historyRows, setHistoryRows] = useState<PortHistoryRow[] | null>(null)

  const loadPorts = (rid: number) => {
    setError('')
    setBusy(true)
    apiFetch<{ items: PortRow[] }>('/ports', { query: { resourceId: rid || undefined } })
      .then((d) => setRows(d?.items ?? []))
      .catch((e) => setError(e instanceof Error ? e.message : r.loadFail))
      .finally(() => setBusy(false))
  }
  useEffect(() => {
    apiFetch<{ items: ResourceRow[] }>('/resources')
      .then((d) => setDevices(d?.items ?? []))
      .catch(() => setDevices([]))
    loadPorts(0)
  }, []) // eslint-disable-line react-hooks/exhaustive-deps

  const deviceName = (rid: number) => devices.find((d) => d.id === rid)?.name ?? `#${rid}`
  const openHistory = (p: PortRow) => {
    setHistory(p)
    setHistoryRows(null)
    apiFetch<{ items: PortHistoryRow[] }>(`/ports/${p.portId}/change-history`)
      .then((d) => setHistoryRows(d?.items ?? []))
      .catch(() => setHistoryRows([]))
  }

  const slice = pageSlice(rows, page, pageSize)

  return (
    <div>
      <PageHead title={r.title} desc={r.desc} />
      <div className="org-card">
        <div className="org-toolbar">
          <select className="org-select" value={resourceId ? String(resourceId) : ''}
            onChange={(e) => { const v = Number(e.target.value) || 0; setResourceId(v); setPage(1); loadPorts(v) }}>
            <option value="">{r.allDevice}</option>
            {devices.map((d) => <option key={d.id} value={d.id}>{d.name} ({d.code})</option>)}
          </select>
          <span className="spacer" />
          <button className="org-btn" disabled={busy} onClick={() => loadPorts(resourceId)}>
            {t.pages.audit.refresh}
          </button>
        </div>
        {error ? <div className="org-error">{error}</div> : (
          <div className="org-table-wrap">
            <table className="org-table">
              <thead><tr>{r.columns.map((x) => <th key={x}>{x}</th>)}</tr></thead>
              <tbody>
                {slice.map((p) => (
                  <tr key={p.portId}>
                    <td>{p.portCode}</td>
                    <td>{p.quadCode || '—'}</td>
                    <td>{deviceName(p.resourceId)}</td>
                    <td>{p.addressId ? `#${p.addressId}` : '—'}</td>
                    <td><StatusTag domain="port" value={p.status} /></td>
                    <td>{p.orderId ? `#${p.orderId}` : '—'}</td>
                    <td>
                      <span className="org-act">
                        <button onClick={() => openHistory(p)}>{r.history}</button>
                      </span>
                    </td>
                  </tr>
                ))}
                {!slice.length && <tr><td colSpan={7}><div className="org-empty">{r.empty}</div></td></tr>}
              </tbody>
            </table>
          </div>
        )}
        <div className="org-footer">
          <Pagination total={rows.length} page={page} pageSize={pageSize}
            onPage={setPage} onSize={setPageSize} {...pagerTexts(r)} />
        </div>
      </div>
      {history && (
        <Drawer title={`${r.historyTitle} · ${history.portCode}`} onClose={() => setHistory(null)}
          footer={<button className="org-btn org-btn-primary" onClick={() => setHistory(null)}>
            {t.pages.company.cancel}
          </button>}>
          <div className="org-table-wrap">
            <table className="org-table">
              <thead><tr>{r.historyColumns.map((x) => <th key={x}>{x}</th>)}</tr></thead>
              <tbody>
                {(historyRows ?? []).map((h) => (
                  <tr key={h.id}>
                    <td>{fmtTime(h.changedAt)}</td>
                    <td><StatusTag domain="port" value={h.status} /></td>
                    <td>{h.orderId ? `#${h.orderId}` : '—'}</td>
                  </tr>
                ))}
                {historyRows !== null && !historyRows.length && (
                  <tr><td colSpan={3}><div className="org-empty">{r.empty}</div></td></tr>
                )}
              </tbody>
            </table>
          </div>
        </Drawer>
      )}
    </div>
  )
}

// GIS 地图页:契约 GET /gis/levels + GET /gis/drill?level&parentId + GET /gis/resources/:id/detail。
import { useEffect, useState } from 'react'
import { apiFetch } from '../../../api/client'
import { useT } from '../../../i18n'
import { PageHead, pagerTexts } from '../../org/shared'
import { Pagination } from '../../../components/Pagination'
import { DetailDrawer } from '../../org/shared'
import { fmtTime } from '../../../lib/format'
import { pageSlice, type GisNode, type GisResourceDetail } from '../types'
import '../../org/org.css'

const LEVELS = [1, 2, 3, 4, 5, 6, 7, 8] as const

export default function GisPage() {
  const t = useT()
  const g = t.pages.gisPage
  const [level, setLevel] = useState(1)
  const [parentId, setParentId] = useState(0)
  const [nodes, setNodes] = useState<GisNode[]>([])
  const [error, setError] = useState('')
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(10)
  const [busy, setBusy] = useState(false)
  const [detail, setDetail] = useState<GisResourceDetail | null>(null)
  const [detailError, setDetailError] = useState('')

  const load = (lv: number, pid: number) => {
    setError('')
    setBusy(true)
    apiFetch<{ items: GisNode[] }>('/gis/drill', { query: { level: lv, parentId: pid || undefined } })
      .then((d) => setNodes(d?.items ?? []))
      .catch((e) => setError(e instanceof Error ? e.message : g.loadFail))
      .finally(() => setBusy(false))
  }
  useEffect(() => { load(1, 0) }, []) // eslint-disable-line react-hooks/exhaustive-deps

  const openDetail = (resourceId: number) => {
    setDetail(null)
    setDetailError('')
    apiFetch<GisResourceDetail>(`/gis/resources/${resourceId}/detail`)
      .then((d) => setDetail(d))
      .catch(() => setDetailError(g.detailFail))
  }

  const slice = pageSlice(nodes, page, pageSize)

  return (
    <div>
      <PageHead title={g.title} desc={g.desc} />
      <div className="org-card">
        <div className="org-toolbar">
          <select className="org-select" value={level}
            onChange={(e) => { const v = Number(e.target.value); setLevel(v); setParentId(0); setPage(1); load(v, 0) }}>
            {LEVELS.map((lv, i) => <option key={lv} value={lv}>{lv}. {g.levels[i]}</option>)}
          </select>
          <input className="org-input" type="number" placeholder="parentId"
            value={parentId || ''} onChange={(e) => { setParentId(Number(e.target.value) || 0); setPage(1) }} />
          <span className="spacer" />
          <button className="org-btn" disabled={busy} onClick={() => load(level, parentId)}>{t.pages.audit.refresh}</button>
        </div>
        {error ? <div className="org-error">{error}</div> : (
          <div className="org-table-wrap">
            <table className="org-table">
              <thead><tr>{g.drillColumns.map((x) => <th key={x}>{x}</th>)}</tr></thead>
              <tbody>
                {slice.map((n) => (
                  <tr key={n.id}>
                    <td>#{n.id}</td>
                    <td>{n.name}</td>
                    <td>{n.level}. {g.levels[n.level - 1] ?? n.level}</td>
                    <td>{n.count}</td>
                  </tr>
                ))}
                {!slice.length && <tr><td colSpan={4}><div className="org-empty">{g.empty}</div></td></tr>}
              </tbody>
            </table>
          </div>
        )}
        <div className="org-footer">
          <Pagination total={nodes.length} page={page} pageSize={pageSize}
            onPage={setPage} onSize={setPageSize} {...pagerTexts(g)} />
        </div>
        <div style={{ padding: '8px 12px', fontSize: 13, color: '#888' }}>
          <input className="org-input" type="number" style={{ width: 160 }} placeholder="resourceId (6/7 级)"
            onChange={(e) => { const v = Number(e.target.value); if (v > 0) openDetail(v) }} />
        </div>
      </div>
      {(detail || detailError) && (
        <DetailDrawer
          title={g.detailTitle}
          closeText={t.pages.company.cancel}
          onClose={() => { setDetail(null); setDetailError('') }}
          items={detailError ? [{ k: 'Error', v: detailError }] : [
            { k: g.detailFields[0], v: detail?.code ?? '' },
            { k: g.detailFields[1], v: detail?.name ?? '' },
            { k: g.detailFields[2], v: detail?.type ?? '' },
            { k: g.detailFields[3], v: detail?.status ?? '' },
            { k: g.detailFields[4], v: detail?.opticalPower != null ? String(detail.opticalPower) : '—' },
            { k: g.detailFields[5], v: detail?.packetLoss != null ? String(detail.packetLoss) : '—' },
            { k: g.detailFields[6], v: detail?.customerName ?? '' },
            { k: g.detailFields[7], v: detail ? `${detail.portsUsed}/${detail.portsTotal}` : '' },
            { k: g.detailFields[8], v: detail?.collectedAt ? fmtTime(detail.collectedAt) : '—' },
          ]}
        />
      )}
    </div>
  )
}

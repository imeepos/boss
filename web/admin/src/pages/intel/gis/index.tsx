// GIS 地图页:契约 GET /gis/drill + /gis/points + /gis/resources/:id/detail。
// 顶部 4 张统计卡 + PGIS 真地图(主题可切换) + 八级 drill 明细表。
import { useEffect, useMemo, useRef, useState } from 'react'
import { apiFetch } from '../../../api/client'
import { useT } from '../../../i18n'
import { PageHead, pagerTexts } from '../../org/shared'
import { Pagination } from '../../../components/Pagination'
import { Dropdown } from '../../../components/Dropdown'
import { SimplePicker } from '../../../components/pickers/SimplePicker'
import { DetailDrawer } from '../../org/shared'
import { fmtTime } from '../../../lib/format'
import { useLocalStorage } from '../../../lib/useLocalStorage'
import { pageSlice, type GisNode, type GisPointRow, type GisResourceDetail } from '../types'
import { TableStateRow, IdRef } from '../../../components/business'
import { CardShell, StatCard } from '../../../components/business/charts'
import { PgisMap, type GisPoint, type Theme } from '../../../components/business/maps'

const LEVELS = [1, 2, 3, 4, 5, 6, 7, 8] as const

export default function GisPage() {
  const t = useT()
  const g = t.pages.gisPage
  const [level, setLevel] = useState(1)
  const [parentId, setParentId] = useState(0)
  const [nodes, setNodes] = useState<GisNode[]>([])
  const [points, setPoints] = useState<GisPoint[]>([])
  const [error, setError] = useState('')
  const [pointsError, setPointsError] = useState('')
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(10)
  const [busy, setBusy] = useState(false)
  const [detail, setDetail] = useState<GisResourceDetail | null>(null)
  const [detailError, setDetailError] = useState('')
  const [theme, setTheme] = useLocalStorage<Theme>('intel.gis.theme', 'light')
  // ODN 图层(odn-points):'off'|'facility'|'site'|'device'|'survey'|'progress'(W7 勘测/进度)。
  const [odnLayer, setOdnLayer] = useState<'off' | 'facility' | 'site' | 'device' | 'survey' | 'progress'>('off')
  const [odnPoints, setOdnPoints] = useState<GisPoint[]>([])

  const load = (lv: number, pid: number) => {
    setError('')
    setBusy(true)
    apiFetch<{ items: GisNode[] }>('/gis/drill', { query: { level: lv, parentId: pid || undefined } })
      .then((d) => setNodes(d?.items ?? []))
      .catch((e) => setError(e instanceof Error ? e.message : g.loadFail))
      .finally(() => setBusy(false))
  }
  // 地图点位与 drill 同步(level/parentId 改变即重拉)。
  const loadPoints = (lv: number, pid: number) => {
    setPointsError('')
    apiFetch<{ items: GisPointRow[] }>('/gis/points', { query: { level: lv, parentId: pid || undefined } })
      .then((d) => setPoints((d?.items ?? []).map((r) => ({
        id: r.id, level: r.level, name: r.name,
        lng: r.lng, lat: r.lat, status: r.status,
        count: r.count, parentId: r.parentId,
      }))))
      .catch((e) => setPointsError(e instanceof Error ? e.message : g.mapLoadFail))
  }
  useEffect(() => { load(1, 0); loadPoints(1, 0) }, []) // eslint-disable-line react-hooks/exhaustive-deps

  const refreshAll = () => { load(level, parentId); loadPoints(level, parentId); loadOdnPoints(odnLayer === 'off' ? 'facility' : odnLayer) }
  const changeLevel = (lv: number) => {
    setLevel(lv); setParentId(0); setPage(1)
    load(lv, 0); loadPoints(lv, 0)
  }

  // 上级节点筛选数据源:同页 /gis/drill 取上级层(level-1)节点;level=1 无上级,选择器置灰。
  const searchParentNodes = async (keyword: string): Promise<{ value: string; label: string }[] | null> => {
    if (level <= 1) return []
    const d = await apiFetch<{ items: GisNode[] }>('/gis/drill', { query: { level: level - 1 } })
    const kw = keyword.trim().toLowerCase()
    return (d?.items ?? [])
      .filter((n) => !kw || n.name.toLowerCase().includes(kw) || String(n.id).includes(kw))
      .map((n) => ({ value: String(n.id), label: n.name + ' (#' + n.id + ')' }))
  }
  const changeParent = (v: string) => {
    const pid = Number(v) || 0
    setParentId(pid); setPage(1)
    load(level, pid); loadPoints(level, pid)
  }

  // ODN 图层点位:entity 显式传入(不用闭包 odnLayer,避免 state 未刷新误判 off);
  // bbox 空=全量(图层为概览层,不随视域收缩)。
  const loadOdnPoints = (entity: 'facility' | 'site' | 'device' | 'survey' | 'progress') => {
    setPointsError('')
    apiFetch<{ items: GisPointRow[] }>('/gis/odn-points', { query: { entity } })
      .then((d) => setOdnPoints((d?.items ?? []).map((r) => ({
        id: r.id, level: r.level, name: r.name,
        lng: r.lng, lat: r.lat, status: r.status,
        count: r.count, parentId: r.parentId,
      }))))
      .catch((e) => setPointsError(e instanceof Error ? e.message : g.mapLoadFail))
  }
  const switchOdnLayer = (v: 'off' | 'facility' | 'site' | 'device' | 'survey' | 'progress') => {
    setOdnLayer(v)
    if (v === 'off') setOdnPoints([])
    else loadOdnPoints(v)
  }

  const openDetail = (resourceId: number) => {
    setDetail(null)
    setDetailError('')
    apiFetch<GisResourceDetail>(`/gis/resources/${resourceId}/detail`)
      .then((d) => setDetail(d))
      .catch(() => setDetailError(g.detailFail))
  }

  const slice = pageSlice(nodes, page, pageSize)
  // 地图点位 = 当前层级点位 + ODN 图层点位(开关非 off 时叠加)。
  const mapPoints = odnLayer === 'off' ? points : [...points, ...odnPoints]

  // 顶部 4 张统计卡:当前层级点位/视域内点位/在线点位/平均子级数。
  // 视域内点位由 map.onViewportChange(B2)持续更新,inBbox 在 mapPoints 变化时重置。
  const onlineCount = useMemo(() => mapPoints.filter((p) => p.status === 'ONLINE').length, [mapPoints])
  const avgCount = useMemo(() => {
    if (mapPoints.length === 0) return 0
    return Math.round(mapPoints.reduce((s, p) => s + p.count, 0) / mapPoints.length)
  }, [mapPoints])
  // 视域内点位:bbox 变化 + mapPoints 变化时重算。
  const [inBbox, setInBbox] = useState(0)
  const bboxRef = useRef<{ minLng: number; minLat: number; maxLng: number; maxLat: number } | null>(null)
  const onViewportChange = (b: { minLng: number; minLat: number; maxLng: number; maxLat: number }) => {
    bboxRef.current = b
    setInBbox(mapPoints.filter((p) => p.lng >= b.minLng && p.lng <= b.maxLng && p.lat >= b.minLat && p.lat <= b.maxLat).length)
  }
  // mapPoints 变化时用上次 bbox 重算
  useEffect(() => {
    const b = bboxRef.current
    if (!b) return
    setInBbox(mapPoints.filter((p) => p.lng >= b.minLng && p.lng <= b.maxLng && p.lat >= b.minLat && p.lat <= b.maxLat).length)
  }, [mapPoints])

  return (
    <div>
      <PageHead title={g.title} desc={g.desc} />
      <div className="mb-4 flex flex-wrap items-center gap-2">
        <Dropdown
          value={String(level)}
          options={LEVELS.map((lv, i) => ({ value: String(lv), label: `${lv}. ${g.levels[i]}` }))}
          onChange={(v) => changeLevel(Number(v))}
          ariaLabel={g.title}
        />
        <SimplePicker
          key={level}
          value={parentId ? String(parentId) : ''}
          search={searchParentNodes}
          onChange={changeParent}
          ariaLabel={g.parentNode}
          placeholder={g.parentNode}
          searchPlaceholder={g.parentNode}
          errorText={g.loadFail}
          emptyLabel={t.pages.pickers.common.all}
          clearable
          clearLabel={t.pages.pickers.common.clear}
          disabled={level <= 1}
          minWidth={200}
        />
        <button className="h-8 cursor-pointer rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-4 text-[13px] text-[var(--shell-content-text)] hover:border-[var(--color-border-hover)] hover:text-[var(--shell-heading)]" disabled={busy} onClick={refreshAll}>{t.pages.audit.refresh}</button>
        <span className="flex-1" />
        <Dropdown
          value={odnLayer}
          options={[
            { value: 'off', label: g.odnLayerOff },
            { value: 'facility', label: g.odnLayerFacility },
            { value: 'site', label: g.odnLayerSite },
            { value: 'device', label: g.odnLayerDevice },
            { value: 'survey', label: g.odnLayerSurvey },
            { value: 'progress', label: g.odnLayerProgress },
          ]}
          onChange={(v) => switchOdnLayer(v as 'off' | 'facility' | 'site' | 'device' | 'survey' | 'progress')}
          ariaLabel={g.odnLayerTitle}
        />
        <button className="h-8 cursor-pointer rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-4 text-[13px] text-[var(--shell-content-text)] hover:border-[var(--color-border-hover)] hover:text-[var(--shell-heading)]" title={g.themeSwitchHint} onClick={() => setTheme(theme === 'light' ? 'dark' : 'light')}>
          {theme === 'light' ? g.themeDark : g.themeLight}
        </button>
      </div>

      <section className="mb-6 grid grid-cols-1 gap-4 md:grid-cols-2 xl:grid-cols-4">
        <StatCard label={g.statLevelNodes} value={mapPoints.length} />
        <StatCard label={g.statInBbox} value={inBbox} />
        <StatCard label={g.statOnline} value={onlineCount} />
        <StatCard label={g.statAvgCount} value={avgCount} />
      </section>

      <CardShell className="mb-4">
        <div className="mb-3 flex items-center justify-between">
          <h3 className="m-0 text-base font-semibold text-[var(--shell-heading)]">{g.mapTitle}</h3>
          <button className="h-8 cursor-pointer rounded-sm border-none bg-[var(--shell-fab-bg)] px-4 text-[13px] text-[var(--shell-fab-icon)] hover:bg-[var(--shell-fab-bg-hover)]" disabled={busy} onClick={refreshAll}>{g.mapRefresh}</button>
        </div>
        {pointsError ? (
          <div className="mx-2 mb-2 rounded-sm border border-[color-mix(in_srgb,var(--color-danger)_25%,transparent)] bg-[color-mix(in_srgb,var(--color-danger)_8%,transparent)] px-3 py-2 text-[13px] text-[var(--color-danger)]">{pointsError}</div>
        ) : null}
        <div className="relative h-[480px]">
          <PgisMap points={mapPoints} onSelect={(p) => p.level >= 6 && p.level <= 8 && openDetail(p.id)} theme={theme} onViewportChange={onViewportChange} />
          {!mapPoints.length && !pointsError ? (
            <div className="pointer-events-none absolute inset-0 flex items-center justify-center text-[13px] text-[var(--shell-group-title)]">
              {g.mapEmpty}
            </div>
          ) : null}
        </div>
      </CardShell>

      <CardShell>
        {error ? <div className="mx-2 mb-2 rounded-sm border border-[color-mix(in_srgb,var(--color-danger)_25%,transparent)] bg-[color-mix(in_srgb,var(--color-danger)_8%,transparent)] px-3 py-2 text-[13px] text-[var(--color-danger)]">{error}</div> : (
          <div className="overflow-x-auto">
            <table className="w-full border-collapse text-[13px] text-[var(--shell-content-text)]">
              <thead className="h-11 px-3 text-left text-xs font-medium whitespace-nowrap border-b border-[var(--shell-side-border)] bg-[var(--shell-menu-hover-bg)] text-[var(--shell-group-title)]"><tr>{g.drillColumns.map((x) => <th key={x} className="h-11 px-3 text-left text-xs font-medium whitespace-nowrap border-b border-[var(--shell-side-border)] bg-[var(--shell-menu-hover-bg)] text-[var(--shell-group-title)]">{x}</th>)}</tr></thead>
              <tbody>
                {slice.map((n) => (
                  <tr key={n.id}>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]"><IdRef value={n.id} /></td>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{n.name}</td>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{n.level}. {g.levels[n.level - 1] ?? n.level}</td>
                    <td className="h-11 px-3 whitespace-nowrap border-b border-[var(--shell-side-border)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]">{n.count}</td>
                  </tr>
                ))}
                {!slice.length && <TableStateRow colSpan={4} loading={busy} text={g.empty} />}
              </tbody>
            </table>
          </div>
        )}
        <div className="flex justify-end pt-3 text-xs text-[var(--shell-group-title)]">
          <Pagination total={nodes.length} page={page} pageSize={pageSize}
            onPage={setPage} onSize={setPageSize} {...pagerTexts(g)} />
        </div>
      </CardShell>

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
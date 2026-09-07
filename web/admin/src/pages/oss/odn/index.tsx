// ODN 无源网络页 v2(designs/oss-odn-v2.spec.md §1):左树(280px)+右主区布局。
// 树选择经 ResourceTree;prv/city/tab 同步路由 query,刷新可还原(spec §2)。
import { useCallback, useEffect, useMemo, useState } from 'react'
import { useNavigate, useSearchParams } from 'react-router-dom'
import { ChevronRight, X } from 'lucide-react'
import { toast } from 'sonner'
import { apiFetch } from '../../../api/client'
import { useT } from '../../../i18n'
import { Badge } from '../../../components/ui/badge'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '../../../components/ui/table'
import { EmptyState, ErrorBanner, ToolbarButton } from '../../../components/business/page-head'
import { Pagination } from '../../../components/Pagination'
import { pagerTexts } from '../../org/shared'
import { BatchImportEntry } from '../../base/importer/BatchImportEntry'
import { useConfirm } from '../../../components/ConfirmDialog'
import { AssetsPanel } from './AssetsPanel'
import { CoveragePanel } from './coverage'
import ConstructionsPanel from './constructions'
import { ResourceTree, type GroupKey, type LeafFocus, type ConstructionLite } from './ResourceTree'
import { KpiCards } from './KpiCards'
import { RelationChain, type ChainCoverage } from './RelationChain'
import { CARD, LABEL, ODNForm, type Tab, type Grid, type Facility, type Site, type Device, type Region, type City } from './forms'
import type { ResourceRow } from '../types'

// constructions tab (P-INFRA-1 W1): panel has own literals (i18n central files frozen for W1).
/** tab → 批量导入实体 kind(与 base/importer/entities.ts 对齐;coverage 无批量导入)。 */
const TAB_KIND: Partial<Record<Tab, string>> = { grids: 'odn_grid', facilities: 'odn_facility', sites: 'odn_site', devices: 'odn_device' }
const TABS: Tab[] = ['grids', 'facilities', 'sites', 'devices', 'coverage', 'constructions', 'assets']

export default function ODNPage() {
  const t = useT()
  const g = t.pages.odn
  const confirmDialog = useConfirm()
  const navigate = useNavigate()
  const [sp, setSp] = useSearchParams()
  // 路由 query 为唯一事实源:prv/city/tab 刷新可还原(spec §2)。tab 非法值回落 grids。
  const prv = sp.get('prv') ?? 'PHL001'
  const city = sp.get('city') ?? ''
  const tabRaw = sp.get('tab') as Tab | null
  const tab: Tab = tabRaw !== null && TABS.includes(tabRaw) ? tabRaw : 'grids'
  const patch = (d: Record<string, string>) => {
    const next = new URLSearchParams(sp)
    for (const [k, v] of Object.entries(d)) { if (v) next.set(k, v); else next.delete(k) }
    setSp(next, { replace: true })
  }

  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)
  const [grids, setGrids] = useState<Grid[]>([])
  const [facilities, setFacilities] = useState<Facility[]>([])
  const [sites, setSites] = useState<Site[]>([])
  const [devices, setDevices] = useState<Device[]>([])
  const [regions, setRegions] = useState<Region[]>([])
  const [cities, setCities] = useState<City[]>([])
  const [olts, setOlts] = useState<ResourceRow[]>([])
  const [constructions, setConstructions] = useState<ConstructionLite[]>([])
  const [coverages, setCoverages] = useState<ChainCoverage[]>([])
  const [showForm, setShowForm] = useState(false)
  const [focus, setFocus] = useState<LeafFocus>(null)
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(10)

  // 省级字典一次性加载;城市字典随省联动,当前城市不在新省列表时锚定首个。
  useEffect(() => {
    apiFetch<Region[]>('/odn/regions').then((xs) => setRegions(xs ?? [])).catch((e) => { console.error('[odn] REGIONS LOAD FAILED', e); setRegions([]) })
  }, [])
  useEffect(() => {
    if (!prv) return
    apiFetch<City[]>('/odn/cities', { query: { prvCode: prv } }).then((xs) => {
      const list = xs ?? []
      setCities(list)
      // 当前城市不在新省列表:经 URL 锚定首个(query 为唯一事实源)。
      if (!list.some((ct) => ct.cityPrefix === city)) {
        const first = list[0]?.cityPrefix ?? ''
        setSp((p) => { const n = new URLSearchParams(p); if (first) n.set('city', first); else n.delete('city'); return n }, { replace: true })
      }
    }).catch((e) => { console.error('[odn] CITIES LOAD FAILED', e); setCities([]) })
  }, [prv]) // eslint-disable-line react-hooks/exhaustive-deps

  // 当前城市数据整卡加载(树计数与四 Tab 共用);失败留 [odn] FAILED 日志并上屏。
  const loadCity = useCallback(async () => {
    if (!prv || !city) return
    const q = { prvCode: prv, cityPrefix: city }
    setError('')
    const rs = await Promise.allSettled([
      apiFetch<Grid[]>('/odn/grids', { query: q }),
      apiFetch<Facility[]>('/odn/facilities', { query: q }),
      apiFetch<Site[]>('/odn/sites', { query: q }),
      apiFetch<Device[]>('/odn/devices', { query: q }),
    ])
    const kinds = ['GRIDS', 'FACILITIES', 'SITES', 'DEVICES']
    rs.forEach((r, i) => { if (r.status === 'rejected') console.error('[odn] ' + kinds[i] + ' LOAD FAILED', r.reason) })
    const [r0, r1, r2, r3] = rs
    setGrids(r0.status === 'fulfilled' ? r0.value ?? [] : [])
    setFacilities(r1.status === 'fulfilled' ? r1.value ?? [] : [])
    setSites(r2.status === 'fulfilled' ? r2.value ?? [] : [])
    setDevices(r3.status === 'fulfilled' ? r3.value ?? [] : [])
    if (rs.some((r) => r.status === 'rejected')) setError(g.loadFail)
  }, [city, g.loadFail, prv])
  useEffect(() => { void loadCity() }, [loadCity])

  // OLT(资源域 /resources,信封 {items})与施工项目不随市变:进页各加载一次。
  useEffect(() => {
    apiFetch<{ items: ResourceRow[] }>('/resources').then((x) => setOlts((x?.items ?? []).filter((r) => r.type === 'OLT')))
      .catch((e) => console.error('[odn] OLT RESOURCES LOAD FAILED', e))
    apiFetch<ConstructionLite[]>('/odn/constructions').then((xs) => setConstructions(xs ?? []))
      .catch((e) => console.error('[odn] CONSTRUCTIONS LOAD FAILED', e))
    apiFetch<ChainCoverage[]>('/odn/coverage/list', { query: { limit: 200 } }).then((xs) => setCoverages(xs ?? []))
      .catch((e) => console.error('[odn] COVERAGE LOAD FAILED', e))
  }, [])

  const treeData = useMemo(() => ({ grids, facilities, sites, olts, constructions }), [grids, facilities, sites, olts, constructions])

  const onPrv = (v: string) => { setFocus(null); setPage(1); patch({ prv: v, city: '' }) }
  const onCity = (v: string) => { setFocus(null); setPage(1); patch({ city: v }) }
  // 点 L4:主区切到对应 Tab 并过滤该行;OLT 属资源域,跳设备页过滤该 OLT(spec §3)。
  const onLeaf = (group: GroupKey, key: string) => {
    if (group === 'olts') { void navigate('/oss/device?resourceId=' + key.slice(2)); return }
    if (group === 'grids') patch({ tab: key.startsWith('f:') ? 'facilities' : 'grids' })
    else if (group === 'sites') patch({ tab: 'sites' })
    else patch({ tab: 'constructions' })
    setFocus({ group, key })
    setPage(1)
  }

  // L4 聚焦 → 表格过滤联动;清空恢复全量。
  const visGrids = focus !== null && focus.group === 'grids' && focus.key.startsWith('g:') ? grids.filter((x) => 'g:' + x.gridCode === focus.key) : grids
  const visFacilities = focus !== null && focus.group === 'grids' && focus.key.startsWith('f:') ? facilities.filter((x) => 'f:' + x.code === focus.key) : facilities
  const visSites = focus !== null && focus.group === 'sites' ? sites.filter((x) => 's:' + x.siteNo === focus.key) : sites
  const countOf = tab === 'grids' ? visGrids.length : tab === 'facilities' ? visFacilities.length : tab === 'sites' ? visSites.length : tab === 'devices' ? devices.length : 0
  const pages = Math.max(1, Math.ceil(countOf / pageSize))
  useEffect(() => { if (page > pages) setPage(pages) }, [pages, page])
  const slice = <T,>(rows: T[]): T[] => rows.slice((page - 1) * pageSize, page * pageSize)

  const load = useCallback(async () => { await loadCity() }, [loadCity])

  const submit = async (body: Record<string, unknown>, path: string) => {
    setBusy(true); setError('')
    try {
      await apiFetch(path, { method: 'POST', query: { prvCode: prv, cityPrefix: city }, body })
      setShowForm(false)
      toast.success(g.saveOk)
      await load()
    } catch (e) { setError(e instanceof Error ? e.message : g.saveFail) } finally { setBusy(false) }
  }

  const retire = async (path: string) => {
    if (!(await confirmDialog(g.retireConfirm, { danger: true }))) return
    setBusy(true); setError('')
    try { await apiFetch(path, { method: 'DELETE', query: { prvCode: prv, cityPrefix: city } }); await load() }
    catch (e) { setError(e instanceof Error ? e.message : g.saveFail) } finally { setBusy(false) }
  }

  const region = regions.find((r) => r.prvCode === prv)
  const cityRow = cities.find((ct) => ct.cityPrefix === city)
  // KPI 容量条:网格占用 facilities/999 取最高(spec §1 卡内进度)。
  const capacityPct = grids.length ? Math.min(100, Math.round(Math.max(...grids.map((x) => (x.facilities / 999) * 100)))) : 0
  // 关联 OLT chip:资源域 OLT 与 ODN 市按编码段 OLT-<CITY>-NN 逻辑关联(spec §3 桥接,无 FK)。
  const onOlt = (o: ResourceRow) => { void navigate('/oss/device?resourceId=' + o.id) }
  const cityOlt = olts.find((o) => o.code.split('-').includes(city)) ?? null
  const paged = tab === 'grids'
    ? <GridTable rows={slice(visGrids)} onRetire={(n) => retire('/odn/grids/' + n)} g={g} />
    : tab === 'facilities'
      ? <FacilityTable rows={slice(visFacilities)} onRetire={(code) => retire('/odn/facilities/' + encodeURIComponent(code))} g={g} olt={cityOlt} onOlt={onOlt} />
      : tab === 'sites'
        ? <SiteTable rows={slice(visSites)} onRetire={(n) => retire('/odn/sites/' + n)} g={g} olt={cityOlt} onOlt={onOlt} />
        : <DeviceTable rows={slice(devices)} onRetire={(id) => retire('/odn/devices/' + id)} g={g} olt={cityOlt} onOlt={onOlt} />

  return <div className="flex items-start gap-4">
    <aside className="w-[280px] shrink-0">
      <ResourceTree regions={regions} cities={cities} prv={prv} city={city} data={treeData} focus={focus} g={g.tree} onPrv={onPrv} onCity={onCity} onLeaf={onLeaf} />
    </aside>
    <main className="min-w-0 flex-1">
      <nav aria-label="breadcrumb" className="mb-3 flex items-center gap-1 text-xs text-[var(--shell-crumb-text)]">
        <span>{prv}{region ? ' ' + region.name : ''}</span>
        {cityRow && <><ChevronRight className="h-3 w-3" /><span>{cityRow.cityPrefix} {cityRow.name}</span></>}
        {focus !== null && <><ChevronRight className="h-3 w-3" /><span className="font-medium text-[var(--shell-heading)]">{focus.key.slice(2)}</span>
          <button type="button" aria-label={g.tree.clearFilter} onClick={() => setFocus(null)} className="ml-1 cursor-pointer border-none bg-transparent p-0.5 text-[var(--color-text-tertiary)] hover:text-[var(--shell-heading)]"><X className="h-3 w-3" /></button></>}
      </nav>
      <div className="mb-4 flex items-center justify-between">
        <div><h2 className="m-0 text-xl font-bold text-[var(--shell-heading)]">{g.title}</h2><p className="mt-1 text-xs text-[var(--shell-crumb-text)]">{g.subtitle}</p></div>
        <div className="flex items-center gap-2">{TAB_KIND[tab] && <BatchImportEntry kind={TAB_KIND[tab]} onImported={load} />}
          {tab !== 'coverage' && tab !== 'constructions' && tab !== 'assets' && <ToolbarButton primary onClick={() => setShowForm(!showForm)}>{showForm ? g.cancel : g.add}</ToolbarButton>}</div>
      </div>
      <div className="mb-4">
        <KpiCards grids={grids.length} facilities={facilities.length} sites={sites.length} olts={olts.length} capacityPct={capacityPct} g={g.kpi} />
      </div>
      <div className="mb-4 flex gap-6 border-b border-[var(--shell-side-border)]">
        {TABS.map((key) => <button key={key} className={'cursor-pointer border-b-2 px-1 py-3 text-sm ' + (tab === key ? 'border-[var(--color-brand-gold-500)] font-semibold text-[var(--shell-heading)]' : 'border-transparent text-[var(--shell-content-text)]')} onClick={() => { patch({ tab: key }); setShowForm(false); setPage(1) }}>{g.tabs[key]}</button>)}
      </div>
      <DependencyHint show={tab === 'facilities' && grids.length === 0} message={g.hintNeedGrid} action={g.hintGotoGrids} onAction={() => patch({ tab: 'grids' })} />
      <DependencyHint show={tab === 'devices' && sites.length === 0} variant="info" message={g.hintNeedSite} action={g.tabs.sites} onAction={() => patch({ tab: 'sites' })} />
      {error && <ErrorBanner message={error} className="mt-3" />}
      {showForm && tab !== 'coverage' && tab !== 'constructions' && tab !== 'assets' && <ODNForm tab={tab} busy={busy} prv={prv} city={city} submit={submit} g={g} grids={grids} sites={sites} devices={devices} />}
      {tab === 'devices' && <div className="mt-4"><RelationChain olts={olts} sites={sites} devices={devices} facilities={facilities} coverages={coverages} city={city} g={g.chain} /></div>}
      <section className={CARD + ' mt-4 overflow-hidden'}>
        {tab === 'grids' && paged}
        {tab === 'facilities' && paged}
        {tab === 'sites' && paged}
        {tab === 'devices' && paged}
        {tab === 'coverage' && <CoveragePanel g={g} prv={prv} city={city} />}
        {tab === 'constructions' && <ConstructionsPanel />}
        {tab === 'assets' && <AssetsPanel />}
      </section>
      {countOf > 0 && tab !== 'coverage' && tab !== 'constructions' && tab !== 'assets' && (
        <Pagination page={page} pageSize={pageSize} total={countOf} onPage={setPage} onSize={(n) => { setPageSize(n); setPage(1) }} {...pagerTexts(g)} />
      )}
    </main>
  </div>
}

// DependencyHint 前置依赖提示:资源按 网格→设施→局点→设备 顺序建,缺前置时空态引导跳转。
function DependencyHint({ show, message, action, onAction, variant = 'warning' }: { show: boolean; message: string; action: string; onAction: () => void; variant?: 'warning' | 'info' }) {
  if (!show) return null
  return <div className="mt-3 flex items-center justify-between gap-3 rounded-md border border-[var(--shell-side-border)] bg-[var(--shell-card-bg)] px-3 py-2">
    <div className="flex items-center gap-2"><Badge variant={variant}>{variant === 'warning' ? '!' : 'i'}</Badge><span className={LABEL}>{message}</span></div>
    <ToolbarButton onClick={onAction}>{action}</ToolbarButton>
  </div>
}

function GridTable({ rows, onRetire, g }: { rows: Grid[]; onRetire: (n: number) => void; g: any }) { return <DataTable headers={[g.gridCode, g.name, g.coverage, g.usage, g.status, g.actions]} rows={rows} empty={g.empty} render={(r) => <TableRow key={r.gridCode}><TableCell>{String(r.gridCode).padStart(2, '0')}</TableCell><TableCell>{r.name}</TableCell><TableCell>{r.coverage}</TableCell><TableCell>{r.facilities}/999 {r.warn && <Badge variant="warning">{g.warn}</Badge>}</TableCell><TableCell>{r.status}</TableCell><TableCell><button className="text-[var(--color-text-link)]" onClick={() => onRetire(r.gridCode)}>{g.retire}</button></TableCell></TableRow>} /> }
// OltChip 关联 OLT:蓝 tinted 底(14% alpha)+ mono 12px,非状态色(spec §6);点击跳设备页过滤。
function OltChip({ olt, onOlt, label }: { olt: ResourceRow; onOlt: (o: ResourceRow) => void; label: string }) {
  return <button type="button" aria-label={label + ' ' + olt.code} onClick={() => onOlt(olt)} className="cursor-pointer rounded-full px-2 py-0.5 font-mono text-[12px] leading-5 text-[var(--color-info)]" style={{ background: 'color-mix(in srgb, var(--color-info) 14%, transparent)' }}>{olt.code}</button>
}
function FacilityTable({ rows, onRetire, g, olt, onOlt }: { rows: Facility[]; onRetire: (c: string) => void; g: any; olt: ResourceRow | null; onOlt: (o: ResourceRow) => void }) { return <DataTable headers={[g.code, g.kind, g.relGrid, g.name, g.status, g.kpi.linkedOlt, g.actions]} rows={rows} empty={g.empty} render={(r) => <TableRow key={r.code}><TableCell className="font-mono">{r.code}</TableCell><TableCell>{r.kind}</TableCell><TableCell>{r.gridCode ? String(r.gridCode).padStart(2, '0') : '-'}</TableCell><TableCell>{r.name}</TableCell><TableCell>{r.status}</TableCell><TableCell>{olt && <OltChip olt={olt} onOlt={onOlt} label={g.kpi.linkedOlt} />}</TableCell><TableCell><button className="text-[var(--color-text-link)]" onClick={() => onRetire(r.code)}>{g.retire}</button></TableCell></TableRow>} /> }
function SiteTable({ rows, onRetire, g, olt, onOlt }: { rows: Site[]; onRetire: (n: number) => void; g: any; olt: ResourceRow | null; onOlt: (o: ResourceRow) => void }) { return <DataTable headers={[g.nodeCode, g.name, g.status, g.kpi.linkedOlt, g.actions]} rows={rows} empty={g.empty} render={(r) => <TableRow key={r.siteNo}><TableCell className="font-mono">{r.cityPrefix}{String(r.siteNo).padStart(3, '0')}</TableCell><TableCell>{r.name}</TableCell><TableCell>{r.status}</TableCell><TableCell>{olt && <OltChip olt={olt} onOlt={onOlt} label={g.kpi.linkedOlt} />}</TableCell><TableCell><button className="text-[var(--color-text-link)]" onClick={() => onRetire(r.siteNo)}>{g.retire}</button></TableCell></TableRow>} /> }
function DeviceTable({ rows, onRetire, g, olt, onOlt }: { rows: Device[]; onRetire: (id: number) => void; g: any; olt: ResourceRow | null; onOlt: (o: ResourceRow) => void }) {
  return <DataTable headers={[g.code, g.kind, g.relSite, g.name, g.status, g.kpi.linkedOlt, g.actions]} rows={rows} empty={g.empty} render={(r) => <TableRow key={r.id}><TableCell className="font-mono">{r.code}</TableCell><TableCell>{r.kind}</TableCell><TableCell className="font-mono">{r.siteNo ? r.cityPrefix + String(r.siteNo).padStart(3, '0') : '-'}</TableCell><TableCell>{r.name}</TableCell><TableCell>{r.status}</TableCell><TableCell>{olt && <OltChip olt={olt} onOlt={onOlt} label={g.kpi.linkedOlt} />}</TableCell><TableCell><button className="text-[var(--color-text-link)]" onClick={() => onRetire(r.id)}>{g.retire}</button></TableCell></TableRow>} /> }
function DataTable<T>({ headers, rows, empty, render }: { headers: string[]; rows: T[]; empty: string; render: (row: T) => React.ReactNode }) { return rows.length === 0 ? <EmptyState text={empty} /> : <div className="overflow-x-auto"><Table><TableHeader><TableRow>{headers.map((h) => <TableHead key={h}>{h}</TableHead>)}</TableRow></TableHeader><TableBody>{rows.map(render)}</TableBody></Table></div> }
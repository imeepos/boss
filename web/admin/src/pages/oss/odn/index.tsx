import { useCallback, useEffect, useState } from 'react'
import { toast } from 'sonner'
import { apiFetch } from '../../../api/client'
import { useT } from '../../../i18n'
import { Badge } from '../../../components/ui/badge'
import { Dropdown } from '../../../components/Dropdown'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '../../../components/ui/table'
import { EmptyState, ErrorBanner, ToolbarButton } from '../../../components/business/page-head'
import { BatchImportEntry } from '../../base/importer/BatchImportEntry'
import { useConfirm } from '../../../components/ConfirmDialog'
import { AssetsPanel } from './AssetsPanel'
import { CoveragePanel } from './coverage'
import ConstructionsPanel from './constructions'
import { CARD, FIELD, LABEL, ODNForm, type Tab, type Grid, type Facility, type Site, type Device } from './forms'

// constructions tab (P-INFRA-1 W1): construction/contractor/settlement, panel has own literals (i18n central files frozen for W1).
// assets tab (P-INFRA-1 W8): 资产化凭证/材料出库,panel own literals 同 W1 先例;types from forms.tsx。
/** tab → 批量导入实体 kind(与 base/importer/entities.ts 对齐;coverage 无批量导入)。 */
const TAB_KIND: Partial<Record<Tab, string>> = { grids: 'odn_grid', facilities: 'odn_facility', sites: 'odn_site', devices: 'odn_device' }

// fmtCoord 坐标展示:空值显示 '-'。
function fmtCoord(v: number | null | undefined): string {
  return v == null ? '-' : String(v)
}

type Region = { prvCode: string; name: string }
type City = { cityPrefix: string; name: string }
type FilterProps = { prv: string; city: string; setPrv: (v: string) => void; setCity: (v: string) => void; regions: Region[]; cities: City[]; g: { prvLabel: string; cityLabel: string } }
// 省市级联下拉:数据源 /odn/regions 与 /odn/cities(000075 字典),替代手填编码文本框。
function CityFilter({ prv, city, setPrv, setCity, regions, cities, g }: FilterProps) {
  return <div className="flex flex-wrap items-end gap-2">
    <label className={FIELD}><span className={LABEL}>{g.prvLabel}</span><Dropdown value={prv} options={regions.map((r) => ({ value: r.prvCode, label: r.prvCode + ' ' + r.name }))} onChange={setPrv} ariaLabel={g.prvLabel} /></label>
    <label className={FIELD}><span className={LABEL}>{g.cityLabel}</span><Dropdown value={city} options={cities.map((ct) => ({ value: ct.cityPrefix, label: ct.cityPrefix + ' ' + ct.name }))} onChange={setCity} ariaLabel={g.cityLabel} /></label>
  </div>
}

export default function ODNPage() {
  const t = useT()
  const g = t.pages.odn
  const confirmDialog = useConfirm()
  const [tab, setTab] = useState<Tab>('grids')
  const [prv, setPrv] = useState('PHL001')
  const [city, setCity] = useState('MNL')
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)
  const [grids, setGrids] = useState<Grid[]>([])
  const [facilities, setFacilities] = useState<Facility[]>([])
  const [sites, setSites] = useState<Site[]>([])
  const [devices, setDevices] = useState<Device[]>([])
  const [regions, setRegions] = useState<Region[]>([])
  const [cities, setCities] = useState<City[]>([])
  const [showForm, setShowForm] = useState(false)

  // 省级字典一次性加载;城市字典随省联动,当前城市不在新省列表时锚定首个。
  useEffect(() => { void apiFetch<Region[]>('/odn/regions').then((xs) => setRegions(xs ?? [])).catch(() => setRegions([])) }, [])
  useEffect(() => {
    if (!prv) return
    void apiFetch<City[]>('/odn/cities', { query: { prvCode: prv } }).then((xs) => {
      const list = xs ?? []
      setCities(list)
      setCity((cur) => (list.some((ct) => ct.cityPrefix === cur) ? cur : (list[0]?.cityPrefix ?? '')))
    }).catch(() => setCities([]))
  }, [prv])

  const load = useCallback(async () => {
    if (!prv || !city) return
    setError('')
    try {
      if (tab === 'grids') setGrids(await apiFetch<Grid[]>('/odn/grids', { query: { prvCode: prv, cityPrefix: city } }) ?? [])
      if (tab === 'facilities') setFacilities(await apiFetch<Facility[]>('/odn/facilities', { query: { prvCode: prv, cityPrefix: city } }) ?? [])
      if (tab === 'sites') setSites(await apiFetch<Site[]>('/odn/sites', { query: { prvCode: prv, cityPrefix: city } }) ?? [])
      if (tab === 'devices') setDevices(await apiFetch<Device[]>('/odn/devices', { query: { prvCode: prv, cityPrefix: city } }) ?? [])
    } catch (e) { setError(e instanceof Error ? e.message : g.loadFail) }
  }, [city, g.loadFail, prv, tab])

  useEffect(() => { void load() }, [load])

  const submit = async (body: Record<string, unknown>, path: string) => {
    setBusy(true); setError('')
    try { await apiFetch(path, { method: 'POST', query: { prvCode: prv, cityPrefix: city }, body }); setShowForm(false); toast.success(g.saveOk); await load() } catch (e) { setError(e instanceof Error ? e.message : g.saveFail) } finally { setBusy(false) }
  }

  const retire = async (path: string) => {
    if (!(await confirmDialog(g.retireConfirm, { danger: true }))) return
    setBusy(true); setError('')
    try { await apiFetch(path, { method: 'DELETE', query: { prvCode: prv, cityPrefix: city } }); await load() } catch (e) { setError(e instanceof Error ? e.message : g.saveFail) } finally { setBusy(false) }
  }

  return <div>
    <div className="mb-4 flex items-center justify-between"><div><h2 className="m-0 text-xl font-bold text-[var(--shell-heading)]">{g.title}</h2><p className="mt-1 text-xs text-[var(--shell-crumb-text)]">{g.subtitle}</p></div><div className="flex items-center gap-2">{TAB_KIND[tab] && <BatchImportEntry kind={TAB_KIND[tab]} onImported={load} />}{tab !== 'coverage' && tab !== 'constructions' && tab !== 'assets' && <ToolbarButton primary onClick={() => setShowForm(!showForm)}>{showForm ? g.cancel : g.add}</ToolbarButton>}</div></div>
    <div className="mb-4 flex gap-6 border-b border-[var(--shell-side-border)]">{(['grids', 'facilities', 'sites', 'devices', 'coverage', 'constructions', 'assets'] as Tab[]).map((key) => <button key={key} className={`cursor-pointer border-b-2 px-1 py-3 text-sm ${tab === key ? 'border-[var(--color-brand-gold-500)] font-semibold text-[var(--shell-heading)]' : 'border-transparent text-[var(--shell-content-text)]'}`} onClick={() => { setTab(key); setShowForm(false) }}>{key === 'constructions' ? '施工项目' : key === 'assets' ? '资产化' : g.tabs[key]}</button>)}</div>
    <CityFilter prv={prv} city={city} setPrv={setPrv} setCity={setCity} regions={regions} cities={cities} g={g} />
    <DependencyHint show={tab === 'facilities' && grids.length === 0} message={g.hintNeedGrid} action={g.hintGotoGrids} onAction={() => { setTab('grids'); setShowForm(false) }} />
    <DependencyHint show={tab === 'devices' && sites.length === 0} variant="info" message={g.hintNeedSite} action={g.tabs.sites} onAction={() => { setTab('sites'); setShowForm(false) }} />
    {error && <ErrorBanner message={error} className="mt-3" />}
    {showForm && tab !== 'coverage' && tab !== 'constructions' && tab !== 'assets' && <ODNForm tab={tab} busy={busy} prv={prv} city={city} submit={submit} g={g} grids={grids} sites={sites} devices={devices} />}
    <section className={`${CARD} mt-4 overflow-hidden`}>
      {tab === 'grids' && <GridTable rows={grids} onRetire={(n) => retire(`/odn/grids/${n}`)} g={g} />}
      {tab === 'facilities' && <FacilityTable rows={facilities} onRetire={(code) => retire(`/odn/facilities/${encodeURIComponent(code)}`)} g={g} />}
      {tab === 'sites' && <SiteTable rows={sites} onRetire={(n) => retire(`/odn/sites/${n}`)} g={g} />}
      {tab === 'devices' && <DeviceTable rows={devices} onRetire={(id) => retire(`/odn/devices/${id}`)} g={g} />}
      {tab === 'coverage' && <CoveragePanel g={g} prv={prv} city={city} />}
      {tab === 'constructions' && <ConstructionsPanel />}
      {tab === 'assets' && <AssetsPanel />}
    </section>
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
function FacilityTable({ rows, onRetire, g }: { rows: Facility[]; onRetire: (c: string) => void; g: any }) { return <DataTable headers={[g.code, g.kind, g.gridCode, g.name, g.lat, g.lng, g.status, '资产', g.actions]} rows={rows} empty={g.empty} render={(r) => <TableRow key={r.code}><TableCell className="font-mono">{r.code}</TableCell><TableCell>{r.kind}</TableCell><TableCell>{r.gridCode || '-'}</TableCell><TableCell>{r.name}</TableCell><TableCell>{fmtCoord(r.lat)}</TableCell><TableCell>{fmtCoord(r.lng)}</TableCell><TableCell>{r.status}</TableCell><TableCell>{r.assetReg ? <Badge variant="success">{r.assetReg.assetCode} · {r.assetReg.registrationNo}</Badge> : <span className="text-xs opacity-40">未登记</span>}</TableCell><TableCell><button className="text-[var(--color-text-link)]" onClick={() => onRetire(r.code)}>{g.retire}</button></TableCell></TableRow>} /> }
function SiteTable({ rows, onRetire, g }: { rows: Site[]; onRetire: (n: number) => void; g: any }) { return <DataTable headers={[g.nodeCode, g.name, g.lat, g.lng, g.status, g.actions]} rows={rows} empty={g.empty} render={(r) => <TableRow key={r.siteNo}><TableCell className="font-mono">{r.cityPrefix}{String(r.siteNo).padStart(3, '0')}</TableCell><TableCell>{r.name}</TableCell><TableCell>{fmtCoord(r.lat)}</TableCell><TableCell>{fmtCoord(r.lng)}</TableCell><TableCell>{r.status}</TableCell><TableCell><button className="text-[var(--color-text-link)]" onClick={() => onRetire(r.siteNo)}>{g.retire}</button></TableCell></TableRow>} /> }
function DeviceTable({ rows, onRetire, g }: { rows: Device[]; onRetire: (n: number) => void; g: any }) {
  const codeById = new Map(rows.map((d) => [d.id, d.code]))
  return <DataTable headers={[g.code, g.kind, g.parentDevice, g.name, g.lat, g.lng, g.status, '资产', g.actions]} rows={rows} empty={g.empty} render={(r) => <TableRow key={r.id}><TableCell className="font-mono">{r.code}</TableCell><TableCell>{r.kind}</TableCell><TableCell className="font-mono">{r.parentId ? (codeById.get(r.parentId) ?? '#' + r.parentId) : '-'}</TableCell><TableCell>{r.name}</TableCell><TableCell>{fmtCoord(r.lat)}</TableCell><TableCell>{fmtCoord(r.lng)}</TableCell><TableCell>{r.status}</TableCell><TableCell>{r.assetReg ? <Badge variant="success">{r.assetReg.assetCode} · {r.assetReg.registrationNo}</Badge> : <span className="text-xs opacity-40">未登记</span>}</TableCell><TableCell><button className="text-[var(--color-text-link)]" onClick={() => onRetire(r.id)}>{g.retire}</button></TableCell></TableRow>} /> }
function DataTable<T>({ headers, rows, empty, render }: { headers: string[]; rows: T[]; empty: string; render: (row: T) => React.ReactNode }) { return rows.length === 0 ? <EmptyState text={empty} /> : <div className="overflow-x-auto"><Table><TableHeader><TableRow>{headers.map((h) => <TableHead key={h}>{h}</TableHead>)}</TableRow></TableHeader><TableBody>{rows.map(render)}</TableBody></Table></div> }

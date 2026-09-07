// ODN 新增/编辑抽屉(spec oss-odn-v2 §0/§4):一切枚举/关联字段走 Dropdown,
// 禁原生 select 与自由文本填编码;编码自动顺延可覆盖,类型变化即时预取。
// 编辑模式按契约分两档:网格可改字段(PUT grids/{code});设施/局点/设备仅
// lifecycle 可转移(PUT */lifecycle,身份字段只读)。
import { useEffect, useState, type ReactNode } from 'react'
import { toast } from 'sonner'
import { apiFetch } from '../../../api/client'
import { Drawer } from '../../../components/Drawer'
import { Dropdown } from '../../../components/Dropdown'
import { Input } from '../../../components/ui/input'
import { ErrorBanner, ToolbarButton } from '../../../components/business/page-head'
import { legalParentKind, nextDeviceCode } from './nextcode'
import type { City, Device, Grid, Region, Facility, Site, Tab } from './forms'

export type DrawerTarget = { tab: Tab; editing: Grid | Facility | Site | Device | null }

interface DrawerProps {
  target: DrawerTarget
  prv: string
  city: string
  regions: Region[]
  grids: Grid[]
  sites: Site[]
  devices: Device[]
  g: Record<string, any>
  onClose: () => void
  onSaved: () => void
}

const FIELD = 'flex flex-col gap-1'
const LABEL = 'text-[13px] text-[var(--color-text-secondary)]'
const HINT = 'text-[11px] text-[var(--color-text-tertiary)]'
const num = (v: string) => (v === '' || !/^-?\d+(\.\d+)?$/.test(v) ? undefined : Number(v))
const KIND_FAC = ['P', 'MH', 'TW', 'CLS', 'TBX']
const KIND_DEV = ['SNW', 'OLT', 'ODF', 'OCC', 'ODB', 'OBD', 'SDB', 'SBD', 'PRT', 'TBP']
const LIFECYCLE = ['PLANNED', 'IN_BUILD', 'IN_SERVICE', 'RETIRED']

export function ResourceDrawer({ target, prv, city, regions, grids, sites, devices, g, onClose, onSaved }: DrawerProps) {
  const { tab, editing } = target
  const isEdit = editing !== null
  const [f, setF] = useState<Record<string, string>>({})
  const [err, setErr] = useState('')
  const [busy, setBusy] = useState(false)
  const [cities, setCities] = useState<City[]>([])
  const set = (k: string, v: string) => setF((s) => ({ ...s, [k]: v }))

  // 初始化:create 锚当前省/市;edit 回填行值(lifecycle 缺省映射自 status)。
  useEffect(() => {
    const e = editing
    setF(e ? {
      prv: e.prvCode, city: e.cityPrefix, kind: 'kind' in e ? e.kind : '',
      gridCode: 'gridCode' in e && e.gridCode ? String(e.gridCode) : '',
      siteNo: 'siteNo' in e && e.siteNo ? String(e.siteNo) : '',
      parentId: 'parentId' in e && e.parentId ? String(e.parentId) : '',
      code: 'code' in e ? e.code : '', name: e.name ?? '',
      lat: 'lat' in e && e.lat != null ? String(e.lat) : '', lng: 'lng' in e && e.lng != null ? String(e.lng) : '',
      coverage: 'coverage' in e ? e.coverage ?? '' : '', status: e.status,
      lifecycle: ('lifecycleStatus' in e ? e.lifecycleStatus : '') || (e.status === 'RETIRED' ? 'RETIRED' : 'IN_SERVICE'),
    } : { prv, city, kind: '', gridCode: '', siteNo: '', parentId: '', code: '', name: '', lat: '', lng: '', coverage: '', status: 'ACTIVE', lifecycle: '' })
    setErr('')
  }, [target]) // eslint-disable-line react-hooks/exhaustive-deps

  // 城市字典随抽屉内省份联动(独立于页面,选省即拉)。
  useEffect(() => {
    if (!f.prv) return
    apiFetch<City[]>('/odn/cities', { query: { prvCode: f.prv } }).then((xs) => setCities(xs ?? [])).catch(() => setCities([]))
  }, [f.prv])

  const kind = f.kind ?? ''

  // 编码自动顺延:设施=服务端 next-code(类型/网格变化即时预取);局点=site-next-no;
  // 设备=前端同型最大序号+1(服务端唯一索引兜底)。
  useEffect(() => {
    if (isEdit || tab !== 'facilities' || !kind) return
    const gridKind = kind === 'P' || kind === 'MH'
    if (gridKind && !f.gridCode) { set('code', ''); return }
    const q: Record<string, string> = { kind }
    if (gridKind) q.gridCode = f.gridCode
    apiFetch<{ code: string }>('/odn/facility-next-code', { query: q })
      .then((r) => { if (r?.code) set('code', r.code) })
      .catch((e) => { console.error('[odn] NEXT-CODE FETCH FAILED', e); set('code', '') })
  }, [isEdit, tab, kind, f.gridCode]) // eslint-disable-line react-hooks/exhaustive-deps
  useEffect(() => {
    if (isEdit || tab !== 'sites' || !f.prv || !f.city) return
    apiFetch<{ siteNo: number }>('/odn/site-next-no', { query: { prvCode: f.prv, cityPrefix: f.city } })
      .then((r) => { if (r?.siteNo) set('siteNo', String(r.siteNo)) })
      .catch((e) => { console.error('[odn] NEXT-NO FETCH FAILED', e); set('siteNo', '') })
  }, [isEdit, tab, f.prv, f.city]) // eslint-disable-line react-hooks/exhaustive-deps
  useEffect(() => {
    if (isEdit || tab !== 'devices' || !kind) return
    const s = nextDeviceCode(kind, devices.map((x) => x.code))
    if (s) set('code', s)
  }, [isEdit, tab, kind]) // eslint-disable-line react-hooks/exhaustive-deps

  const cityGrids = grids.filter((x) => x.cityPrefix === f.city)
  const citySites = sites.filter((x) => x.cityPrefix === f.city)
  const wantParent = kind ? legalParentKind(kind) : ''
  const kindOpts = (tab === 'facilities' ? KIND_FAC : KIND_DEV).map((k) => ({ value: k, label: g.drawer['kind' + k] ?? k }))

  const fld = (label: string, node: ReactNode, hint?: string) => <label className={FIELD}><span className={LABEL}>{label}</span>{node}{hint && <span className={HINT}>{hint}</span>}</label>
  const ro = (label: string, value: string) => fld(label, <Input value={value} disabled />)
  const sel = (label: string, value: string, options: { value: string; label: string }[], onChange: (v: string) => void, searchable = false) => fld(label, <Dropdown value={value} options={options} onChange={onChange} ariaLabel={label} searchable={searchable} emptyText={g.empty} />)

  const save = async () => {
    const q = { prvCode: f.prv, cityPrefix: f.city }
    setBusy(true)
    setErr('')
    try {
      if (!isEdit) {
        if (tab === 'grids') {
          if (!f.gridCode) throw new Error(g.drawer.missingRequired)
          await apiFetch('/odn/grids', { method: 'POST', query: q, body: { gridCode: Number(f.gridCode), name: f.name, coverage: f.coverage, status: f.status, prvCode: f.prv, cityPrefix: f.city } })
        } else if (tab === 'facilities') {
          if (!f.code || !kind || ((kind === 'P' || kind === 'MH') && !f.gridCode)) throw new Error(g.drawer.missingRequired)
          await apiFetch('/odn/facilities', { method: 'POST', query: q, body: { code: f.code, kind, gridCode: f.gridCode ? Number(f.gridCode) : undefined, name: f.name, lat: num(f.lat), lng: num(f.lng), prvCode: f.prv, cityPrefix: f.city } })
        } else if (tab === 'sites') {
          if (!f.siteNo) throw new Error(g.drawer.missingRequired)
          await apiFetch('/odn/sites', { method: 'POST', query: q, body: { siteNo: Number(f.siteNo), name: f.name, lat: num(f.lat), lng: num(f.lng), prvCode: f.prv, cityPrefix: f.city } })
        } else {
          if (!f.code || !kind || !f.siteNo) throw new Error(g.drawer.missingRequired)
          await apiFetch('/odn/devices', { method: 'POST', query: q, body: { code: f.code, kind, siteNo: Number(f.siteNo), parentId: f.parentId ? Number(f.parentId) : undefined, name: f.name, lat: num(f.lat), lng: num(f.lng), prvCode: f.prv, cityPrefix: f.city } })
        }
      } else if (tab === 'grids' && editing) {
        await apiFetch('/odn/grids/' + (editing as Grid).gridCode, { method: 'PUT', query: q, body: { name: f.name, coverage: f.coverage, status: f.status } })
      } else if (editing) {
        const path = tab === 'facilities'
          ? '/odn/facilities/' + encodeURIComponent((editing as Facility).code) + '/lifecycle'
          : tab === 'sites' ? '/odn/sites/' + (editing as Site).siteNo + '/lifecycle' : '/odn/devices/' + (editing as Device).id + '/lifecycle'
        await apiFetch(path, { method: 'PUT', body: { lifecycleStatus: f.lifecycle } })
      }
      toast.success(g.saveOk)
      onSaved()
      onClose()
    } catch (e) {
      console.error('[odn] DRAWER SAVE FAILED', e)
      setErr(e instanceof Error ? e.message : g.saveFail)
    } finally { setBusy(false) }
  }

  const identity = <>{ro(g.prvLabel, (f.prv ?? '') + ' ' + (regions.find((r) => r.prvCode === f.prv)?.name ?? ''))}{ro(g.cityLabel, f.city ?? '')}
    {tab === 'grids' && editing && ro(g.gridCode, String((editing as Grid).gridCode).padStart(2, '0'))}
    {tab === 'facilities' && ro(g.code, (editing as Facility)?.code ?? '')}
    {tab === 'sites' && ro(g.nodeCode, (editing as Site) ? (editing as Site).cityPrefix + String((editing as Site).siteNo).padStart(3, '0') : '')}
    {tab === 'devices' && ro(g.code, (editing as Device)?.code ?? '')}</>

  const createChain = <>{sel(g.prvLabel, f.prv ?? '', regions.map((r) => ({ value: r.prvCode, label: r.prvCode + ' ' + r.name })), (v) => { set('prv', v); set('city', '') })}
    {sel(g.cityLabel, f.city ?? '', cities.map((ct) => ({ value: ct.cityPrefix, label: ct.cityPrefix + ' ' + ct.name })), (v) => set('city', v))}</>

  return <Drawer title={isEdit ? g.drawer.editTitle : g.drawer.createTitle} onClose={onClose} width={440}
    footer={<><ToolbarButton onClick={onClose}>{g.cancel}</ToolbarButton><ToolbarButton primary disabled={busy} onClick={save}>{busy ? g.saving : g.save}</ToolbarButton></>}>
    <div className="flex flex-col gap-4">
      {err && <ErrorBanner message={err} />}
      {isEdit ? identity : createChain}
      {!isEdit && tab === 'grids' && sel(g.gridCode, f.gridCode ?? '', Array.from({ length: 99 }, (_, i) => ({ value: String(i + 1), label: String(i + 1).padStart(2, '0') })), (v) => set('gridCode', v), true)}
      {!isEdit && tab !== 'grids' && tab !== 'sites' && sel(g.kind, kind, kindOpts, (v) => set('kind', v))}
      {!isEdit && tab === 'facilities' && (kind === 'P' || kind === 'MH') && sel(g.relGrid, f.gridCode ?? '', cityGrids.map((x) => ({ value: String(x.gridCode), label: String(x.gridCode).padStart(2, '0') + ' ' + x.name })), (v) => set('gridCode', v), true)}
      {!isEdit && tab === 'devices' && sel(g.relSite, f.siteNo ?? '', citySites.map((x) => ({ value: String(x.siteNo), label: x.cityPrefix + String(x.siteNo).padStart(3, '0') + (x.name ? ' ' + x.name : '') })), (v) => set('siteNo', v), true)}
      {!isEdit && tab === 'devices' && wantParent && sel(g.parentDevice + '(' + wantParent + ')', f.parentId ?? '', devices.filter((x) => x.kind === wantParent).map((x) => ({ value: String(x.id), label: x.code + (x.name ? ' ' + x.name : '') })), (v) => set('parentId', v), true)}
      {!isEdit && tab !== 'grids' && fld(tab === 'sites' ? g.autoSiteNo : g.autoCode, <Input value={tab === 'sites' ? (f.siteNo ?? '') : (f.code ?? '')} onChange={(e) => set(tab === 'sites' ? 'siteNo' : 'code', e.target.value)} className="bg-[var(--shell-input-disabled-bg)]" />, g.drawer.hintOverride)}
      {tab === 'grids' && <>{fld(g.name, <Input value={f.name ?? ''} onChange={(e) => set('name', e.target.value)} />)}{fld(g.coverage, <Input value={f.coverage ?? ''} onChange={(e) => set('coverage', e.target.value)} />)}</>}
      {tab !== 'grids' && fld(g.name, <Input value={f.name ?? ''} disabled={isEdit} onChange={(e) => set('name', e.target.value)} />)}
      {tab === 'grids' && (isEdit
        ? fld(g.status, <Dropdown value={f.status ?? ''} options={['ACTIVE', 'RESERVED', 'RETIRED'].map((s) => ({ value: s, label: s }))} onChange={(v) => set('status', v)} ariaLabel={g.status} />)
        : sel(g.status, f.status ?? '', [{ value: 'ACTIVE', label: 'ACTIVE' }, { value: 'RESERVED', label: 'RESERVED' }], (v) => set('status', v)))}
      {isEdit && tab !== 'grids' && fld(g.drawer.lifecycle, <Dropdown value={f.lifecycle ?? ''} options={LIFECYCLE.map((s) => ({ value: s, label: s }))} onChange={(v) => set('lifecycle', v)} ariaLabel={g.drawer.lifecycle} />)}
      {tab !== 'grids' && <div className="grid grid-cols-2 gap-3">
        {fld(g.lat, <Input value={f.lat ?? ''} disabled={isEdit} onChange={(e) => set('lat', e.target.value)} placeholder="14.55" />)}
        {fld(g.lng, <Input value={f.lng ?? ''} disabled={isEdit} onChange={(e) => set('lng', e.target.value)} placeholder="120.98" />)}
      </div>}
    </div>
  </Drawer>
}
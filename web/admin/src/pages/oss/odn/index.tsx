import { useCallback, useEffect, useState } from 'react'
import { apiFetch } from '../../../api/client'
import { useT } from '../../../i18n'
import { Input } from '../../../components/ui/input'
import { Badge } from '../../../components/ui/badge'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '../../../components/ui/table'
import { EmptyState, ErrorBanner, ToolbarButton } from '../../../components/business/page-head'
import { useConfirm } from '../../../components/ConfirmDialog'

const CARD = 'rounded-md border border-[var(--shell-card-border)] bg-[var(--shell-card-bg)] shadow-[var(--shell-card-shadow)]'
const FIELD = 'flex flex-col gap-1'
const LABEL = 'text-xs text-[var(--shell-content-text)]'

type Tab = 'grids' | 'facilities' | 'sites' | 'devices'
type Grid = { prvCode: string; cityPrefix: string; gridCode: number; name: string; coverage: string; status: string; facilities: number; warn: boolean }
type Facility = { code: string; kind: string; prvCode: string; cityPrefix: string; gridCode: number; name: string; lat: number | null; lng: number | null; status: string }
type Site = { prvCode: string; cityPrefix: string; siteNo: number; name: string; lat: number | null; lng: number | null; status: string }
type Device = { id: number; code: string; kind: string; prvCode: string; cityPrefix: string; siteNo: number; parentId: number; name: string; lat: number | null; lng: number | null; status: string }

// fmtCoord 坐标展示:空值显示 '-'。
function fmtCoord(v: number | null | undefined): string {
  return v == null ? '-' : String(v)
}

type FilterProps = { prv: string; city: string; setPrv: (v: string) => void; setCity: (v: string) => void }
function CityFilter({ prv, city, setPrv, setCity }: FilterProps) {
  return <div className="flex flex-wrap items-end gap-2">
    <label className={FIELD}><span className={LABEL}>PRV</span><Input value={prv} onChange={(e) => setPrv(e.target.value.toUpperCase())} placeholder="PHL001" /></label>
    <label className={FIELD}><span className={LABEL}>城市前缀</span><Input value={city} onChange={(e) => setCity(e.target.value.toUpperCase())} placeholder="MNL" /></label>
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
  const [showForm, setShowForm] = useState(false)

  const load = useCallback(async () => {
    if (!prv || !city) return
    setError('')
    try {
      if (tab === 'grids') setGrids(await apiFetch<Grid[]>('/odn/grids', { query: { prvCode: prv, cityPrefix: city } }) ?? [])
      if (tab === 'facilities') setFacilities(await apiFetch<Facility[]>('/odn/facilities', { query: { prvCode: prv, cityPrefix: city } }) ?? [])
      if (tab === 'sites') setSites(await apiFetch<Site[]>('/odn/sites', { query: { prvCode: prv, cityPrefix: city } }) ?? [])
      if (tab === 'devices') setDevices(await apiFetch<Device[]>('/odn/devices', { query: { prvCode: prv, cityPrefix: city } }) ?? [])
    } catch { setError(g.loadFail) }
  }, [city, g.loadFail, prv, tab])

  useEffect(() => { void load() }, [load])

  const submit = async (body: Record<string, unknown>, path: string) => {
    setBusy(true); setError('')
    try { await apiFetch(path, { method: 'POST', query: { prvCode: prv, cityPrefix: city }, body }); setShowForm(false); await load() } catch { setError(g.saveFail) } finally { setBusy(false) }
  }

  const retire = async (path: string) => {
    if (!(await confirmDialog(g.retireConfirm, { danger: true }))) return
    setBusy(true); setError('')
    try { await apiFetch(path, { method: 'DELETE', query: { prvCode: prv, cityPrefix: city } }); await load() } catch { setError(g.saveFail) } finally { setBusy(false) }
  }

  return <div>
    <div className="mb-4 flex items-center justify-between"><div><h2 className="m-0 text-xl font-bold text-[var(--shell-heading)]">{g.title}</h2><p className="mt-1 text-xs text-[var(--shell-crumb-text)]">{g.subtitle}</p></div><ToolbarButton primary onClick={() => setShowForm(!showForm)}>{showForm ? g.cancel : g.add}</ToolbarButton></div>
    <div className="mb-4 flex gap-6 border-b border-[var(--shell-side-border)]">{(['grids', 'facilities', 'sites', 'devices'] as Tab[]).map((key) => <button key={key} className={`cursor-pointer border-b-2 px-1 py-3 text-sm ${tab === key ? 'border-[var(--color-brand-gold-500)] font-semibold text-[var(--shell-heading)]' : 'border-transparent text-[var(--shell-content-text)]'}`} onClick={() => { setTab(key); setShowForm(false) }}>{g.tabs[key]}</button>)}</div>
    <CityFilter prv={prv} city={city} setPrv={setPrv} setCity={setCity} />
    {error && <ErrorBanner message={error} className="mt-3" />}
    {showForm && <Form tab={tab} busy={busy} submit={submit} prv={prv} city={city} g={g} />}
    <section className={`${CARD} mt-4 overflow-hidden`}>
      {tab === 'grids' && <GridTable rows={grids} onRetire={(n) => retire(`/odn/grids/${n}`)} g={g} />}
      {tab === 'facilities' && <FacilityTable rows={facilities} onRetire={(code) => retire(`/odn/facilities/${encodeURIComponent(code)}`)} g={g} />}
      {tab === 'sites' && <SiteTable rows={sites} onRetire={(n) => retire(`/odn/sites/${n}`)} g={g} />}
      {tab === 'devices' && <DeviceTable rows={devices} onRetire={(id) => retire(`/odn/devices/${id}`)} g={g} />}
    </section>
  </div>
}

type FormProps = { tab: Tab; busy: boolean; submit: (body: Record<string, unknown>, path: string) => Promise<void>; prv: string; city: string; g: any }
function Form({ tab, busy, submit, g }: FormProps) {
  const [values, setValues] = useState<Record<string, string>>({})
  const set = (key: string, value: string) => setValues((v) => ({ ...v, [key]: value }))
  const field = (key: string, label: string, placeholder = '') => <label className={FIELD}><span className={LABEL}>{label}</span><Input value={values[key] ?? ''} placeholder={placeholder} onChange={(e) => set(key, e.target.value)} /></label>
  const save = () => {
    const paths: Record<Tab, string> = { grids: '/odn/grids', facilities: '/odn/facilities', sites: '/odn/sites', devices: '/odn/devices' }
    const body = Object.fromEntries(Object.entries(values).map(([k, v]) => {
      if (v === '') return [k, undefined]
      if (/^-?\d+(\.\d+)?$/.test(v)) return [k, Number(v)]
      return [k, v]
    }))
    void submit(body, paths[tab])
  }
  return <div className={`${CARD} mt-4 p-4`}><div className="grid grid-cols-2 gap-3 md:grid-cols-4">
    {tab === 'grids' && <>{field('gridCode', g.gridCode, '01~99')}{field('name', g.name)}{field('coverage', g.coverage)}{field('status', g.status, 'ACTIVE')}</>}
    {tab === 'facilities' && <>{field('code', g.code, 'P01001')}{field('kind', g.kind, 'P / MH / TW / CLS / TBX')}{field('gridCode', g.gridCode, 'P/MH 必填')}{field('name', g.name)}{field('lat', g.lat, '14.55')}{field('lng', g.lng, '120.98')}</>}
    {tab === 'sites' && <>{field('siteNo', g.siteNo, '001~999')}{field('name', g.name)}{field('lat', g.lat, '14.55')}{field('lng', g.lng, '120.98')}</>}
    {tab === 'devices' && <>{field('code', g.deviceCode, 'OLT001 / ODB001-2')}{field('kind', g.deviceKind, 'OLT / ODB / SDB ...')}{field('siteNo', g.siteNo)}{field('parentId', g.parentId)}{field('lat', g.lat, '14.55')}{field('lng', g.lng, '120.98')}</>}
  </div><div className="mt-3 flex justify-end"><ToolbarButton primary disabled={busy} onClick={save}>{busy ? g.saving : g.save}</ToolbarButton></div></div>
}

function GridTable({ rows, onRetire, g }: { rows: Grid[]; onRetire: (n: number) => void; g: any }) { return <DataTable headers={[g.gridCode, g.name, g.coverage, g.usage, g.status, g.actions]} rows={rows} empty={g.empty} render={(r) => <TableRow key={r.gridCode}><TableCell>{String(r.gridCode).padStart(2, '0')}</TableCell><TableCell>{r.name}</TableCell><TableCell>{r.coverage}</TableCell><TableCell>{r.facilities}/999 {r.warn && <Badge variant="warning">{g.warn}</Badge>}</TableCell><TableCell>{r.status}</TableCell><TableCell><button className="text-[var(--color-text-link)]" onClick={() => onRetire(r.gridCode)}>{g.retire}</button></TableCell></TableRow>} /> }
function FacilityTable({ rows, onRetire, g }: { rows: Facility[]; onRetire: (c: string) => void; g: any }) { return <DataTable headers={[g.code, g.kind, g.gridCode, g.name, g.lat, g.lng, g.status, g.actions]} rows={rows} empty={g.empty} render={(r) => <TableRow key={r.code}><TableCell className="font-mono">{r.code}</TableCell><TableCell>{r.kind}</TableCell><TableCell>{r.gridCode || '-'}</TableCell><TableCell>{r.name}</TableCell><TableCell>{fmtCoord(r.lat)}</TableCell><TableCell>{fmtCoord(r.lng)}</TableCell><TableCell>{r.status}</TableCell><TableCell><button className="text-[var(--color-text-link)]" onClick={() => onRetire(r.code)}>{g.retire}</button></TableCell></TableRow>} /> }
function SiteTable({ rows, onRetire, g }: { rows: Site[]; onRetire: (n: number) => void; g: any }) { return <DataTable headers={[g.nodeCode, g.name, g.lat, g.lng, g.status, g.actions]} rows={rows} empty={g.empty} render={(r) => <TableRow key={r.siteNo}><TableCell className="font-mono">{r.cityPrefix}{String(r.siteNo).padStart(3, '0')}</TableCell><TableCell>{r.name}</TableCell><TableCell>{fmtCoord(r.lat)}</TableCell><TableCell>{fmtCoord(r.lng)}</TableCell><TableCell>{r.status}</TableCell><TableCell><button className="text-[var(--color-text-link)]" onClick={() => onRetire(r.siteNo)}>{g.retire}</button></TableCell></TableRow>} /> }
function DeviceTable({ rows, onRetire, g }: { rows: Device[]; onRetire: (n: number) => void; g: any }) { return <DataTable headers={[g.code, g.kind, g.parentId, g.name, g.lat, g.lng, g.status, g.actions]} rows={rows} empty={g.empty} render={(r) => <TableRow key={r.id}><TableCell className="font-mono">{r.code}</TableCell><TableCell>{r.kind}</TableCell><TableCell>{r.parentId || '-'}</TableCell><TableCell>{r.name}</TableCell><TableCell>{fmtCoord(r.lat)}</TableCell><TableCell>{fmtCoord(r.lng)}</TableCell><TableCell>{r.status}</TableCell><TableCell><button className="text-[var(--color-text-link)]" onClick={() => onRetire(r.id)}>{g.retire}</button></TableCell></TableRow>} /> }
function DataTable<T>({ headers, rows, empty, render }: { headers: string[]; rows: T[]; empty: string; render: (row: T) => React.ReactNode }) { return rows.length === 0 ? <EmptyState text={empty} /> : <div className="overflow-x-auto"><Table><TableHeader><TableRow>{headers.map((h) => <TableHead key={h}>{h}</TableHead>)}</TableRow></TableHeader><TableBody>{rows.map(render)}</TableBody></Table></div> }

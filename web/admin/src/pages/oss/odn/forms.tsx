import { useEffect, useState } from 'react'
import { apiFetch } from '../../../api/client'
import { Input } from '../../../components/ui/input'
import { Dropdown } from '../../../components/Dropdown'
import { ToolbarButton } from '../../../components/business/page-head'
import { legalParentKind, nextDeviceCode } from './nextcode'

// ODN 新增表单(自 index.tsx 抽取;各 tab 字段见 ODNForm)。
// 可操作性三原则:引用一律选择器、编码自动顺延可覆盖、归属链只列合法上级。

export const CARD = 'rounded-md border border-[var(--shell-card-border)] bg-[var(--shell-card-bg)] shadow-[var(--shell-card-shadow)]'
export const FIELD = 'flex flex-col gap-1'
export const LABEL = 'text-xs text-[var(--shell-content-text)]'
const CARD_PADDED = CARD + ' mt-4 p-4'

export type Tab = 'grids' | 'facilities' | 'sites' | 'devices' | 'coverage' | 'constructions' | 'assets'
export type Region = { prvCode: string; name: string }
export type City = { cityPrefix: string; name: string }
export type Grid = { prvCode: string; cityPrefix: string; gridCode: number; name: string; coverage: string; status: string; facilities: number; warn: boolean }
export type AssetReg = { registrationNo: string; assetId: number; assetCode: string; assetStatus: string }
export type Facility = { code: string; kind: string; prvCode: string; cityPrefix: string; gridCode: number; name: string; lat: number | null; lng: number | null; status: string; assetReg?: AssetReg | null }
export type Site = { prvCode: string; cityPrefix: string; siteNo: number; name: string; lat: number | null; lng: number | null; status: string }
export type Device = { id: number; code: string; kind: string; prvCode: string; cityPrefix: string; siteNo: number; parentId: number; name: string; lat: number | null; lng: number | null; status: string; assetReg?: AssetReg | null }

type FormProps = { tab: Tab; busy: boolean; prv: string; city: string; submit: (body: Record<string, unknown>, path: string) => Promise<void>; g: any; grids: Grid[]; sites: Site[]; devices: Device[] }
export function ODNForm({ tab, busy, prv, city, submit, g, grids, sites, devices }: FormProps) {
  const [values, setValues] = useState<Record<string, string>>({})
  const set = (key: string, value: string) => setValues((v) => ({ ...v, [key]: value }))
  const kind = values.kind ?? ''
  const gridSel = values.gridCode ?? ''
  const field = (key: string, label: string, placeholder = '') => <label className={FIELD}><span className={LABEL}>{label}</span><Input value={values[key] ?? ''} placeholder={placeholder} onChange={(e) => set(key, e.target.value)} /></label>
  // 枚举字段统一 Dropdown(路线图规则 3:可枚举输入禁自由文本);选项值为协议原值。
  const select = (key: string, label: string, options: { value: string; label: string }[]) => <label className={FIELD}><span className={LABEL}>{label}</span><Dropdown value={values[key] ?? ''} options={[{ value: '', label: '—' }, ...options]} onChange={(v) => set(key, v)} ariaLabel={label} /></label>
  const enumOpts = (xs: string[]) => xs.map((o) => ({ value: o, label: o }))

  // 切 tab 清空上次输入,避免跨实体键串扰(code/siteNo 等共用键)。
  useEffect(() => { setValues({}) }, [tab])

  // 设施编码自动顺延:P/MH 选网格后取号,TW/CLS/TBX 选类型即取号(服务端 MAX+1,退役不复用)。
  useEffect(() => {
    if (tab !== 'facilities' || !kind) return
    const isGrid = kind === 'P' || kind === 'MH'
    if (isGrid && !gridSel) { set('code', ''); return }
    const query: Record<string, string> = { kind }
    if (isGrid) query.gridCode = gridSel
    void apiFetch<{ code: string }>('/odn/facility-next-code', { query })
      .then((r) => { if (r?.code) set('code', r.code) })
      .catch(() => set('code', ''))
  }, [tab, kind, gridSel])

  // 局点序号自动顺延(NodeCode=城市前缀+3 位序号,预览随输随显)。
  useEffect(() => {
    if (tab !== 'sites' || !prv || !city) return
    void apiFetch<{ siteNo: number }>('/odn/site-next-no', { query: { prvCode: prv, cityPrefix: city } })
      .then((r) => { if (r?.siteNo) set('siteNo', String(r.siteNo)) })
      .catch(() => set('siteNo', ''))
  }, [tab, prv, city])

  // 设备编码建议:同 kind 现存最大序号+1(纯前端预览,后端唯一索引兜底)。
  useEffect(() => {
    if (tab !== 'devices' || !kind) return
    const suggestion = nextDeviceCode(kind, devices.map((d) => d.code))
    if (suggestion) set('code', suggestion)
  }, [tab, kind]) // devices 不入依赖:建议锚定类型切换,列表刷新不覆盖手输

  const save = () => {
    const paths: Record<Tab, string> = { grids: '/odn/grids', facilities: '/odn/facilities', sites: '/odn/sites', devices: '/odn/devices', coverage: '', constructions: '', assets: '' }
    const body = Object.fromEntries(Object.entries(values).map(([k, v]) => {
      if (v === '') return [k, undefined]
      if (/^-?\d+(\.\d+)?$/.test(v)) return [k, Number(v)]
      return [k, v]
    }))
    void submit(body, paths[tab])
  }
  const isGridKind = kind === 'P' || kind === 'MH'
  const wantParent = kind ? legalParentKind(kind) : ''
  const parentOptions = wantParent ? devices.filter((d) => d.kind === wantParent).map((d) => ({ value: String(d.id), label: d.code + (d.name ? ' ' + d.name : '') })) : []
  const siteOptions = sites.map((s) => ({ value: String(s.siteNo), label: s.cityPrefix + String(s.siteNo).padStart(3, '0') + (s.name ? ' ' + s.name : '') }))
  return <div className={CARD_PADDED}><div className="grid grid-cols-2 gap-3 md:grid-cols-4">
    {tab === 'grids' && <>{field('gridCode', g.gridCode, '01~99')}{field('name', g.name)}{field('coverage', g.coverage)}{select('status', g.status, enumOpts(['ACTIVE', 'RESERVED']))}</>}
    {tab === 'facilities' && <>{select('kind', g.kind, enumOpts(['P', 'MH', 'TW', 'CLS', 'TBX']))}{isGridKind && select('gridCode', g.gridCode, grids.map((x) => ({ value: String(x.gridCode), label: String(x.gridCode).padStart(2, '0') + ' ' + x.name })))}{field('code', g.autoCode, 'P01001')}{field('name', g.name)}{field('lat', g.lat, '14.55')}{field('lng', g.lng, '120.98')}</>}
    {tab === 'sites' && <>{field('siteNo', g.autoSiteNo, '001~999')}{values.siteNo && <span className={LABEL}>{g.nodeCode}: {city}{String(values.siteNo).padStart(3, '0')}</span>}{field('name', g.name)}{field('lat', g.lat, '14.55')}{field('lng', g.lng, '120.98')}</>}
    {tab === 'devices' && <>{select('kind', g.deviceKind, enumOpts(['SNW', 'OLT', 'ODF', 'OCC', 'ODB', 'OBD', 'SDB', 'SBD', 'PRT', 'TBP']))}{field('code', g.deviceCode + '(' + g.autoCode + ')', 'OLT001 / ODB001-2')}{select('siteNo', g.parentSite, [{ value: '', label: g.noSite }, ...siteOptions])}{wantParent && select('parentId', g.parentDevice + '(' + wantParent + ')', parentOptions)}{field('lat', g.lat, '14.55')}{field('lng', g.lng, '120.98')}</>}
  </div><div className="mt-3 flex justify-end"><ToolbarButton primary disabled={busy} onClick={save}>{busy ? g.saving : g.save}</ToolbarButton></div></div>
}

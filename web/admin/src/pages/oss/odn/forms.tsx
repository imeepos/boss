import { useState } from 'react'
import { Input } from '../../../components/ui/input'
import { Dropdown } from '../../../components/Dropdown'
import { ToolbarButton } from '../../../components/business/page-head'

// ODN 新增表单(自 index.tsx 抽取;各 tab 字段见 ODNForm)。

export const CARD = 'rounded-md border border-[var(--shell-card-border)] bg-[var(--shell-card-bg)] shadow-[var(--shell-card-shadow)]'
export const FIELD = 'flex flex-col gap-1'
export const LABEL = 'text-xs text-[var(--shell-content-text)]'

export type Tab = 'grids' | 'facilities' | 'sites' | 'devices' | 'coverage' | 'constructions'
export type Grid = { prvCode: string; cityPrefix: string; gridCode: number; name: string; coverage: string; status: string; facilities: number; warn: boolean }
export type Facility = { code: string; kind: string; prvCode: string; cityPrefix: string; gridCode: number; name: string; lat: number | null; lng: number | null; status: string }
export type Site = { prvCode: string; cityPrefix: string; siteNo: number; name: string; lat: number | null; lng: number | null; status: string }
export type Device = { id: number; code: string; kind: string; prvCode: string; cityPrefix: string; siteNo: number; parentId: number; name: string; lat: number | null; lng: number | null; status: string }

type FormProps = { tab: Tab; busy: boolean; submit: (body: Record<string, unknown>, path: string) => Promise<void>; g: any; grids: Grid[]; devices: Device[] }
export function ODNForm({ tab, busy, submit, g, grids, devices }: FormProps) {
  const [values, setValues] = useState<Record<string, string>>({})
  const set = (key: string, value: string) => setValues((v) => ({ ...v, [key]: value }))
  const field = (key: string, label: string, placeholder = '') => <label className={FIELD}><span className={LABEL}>{label}</span><Input value={values[key] ?? ''} placeholder={placeholder} onChange={(e) => set(key, e.target.value)} /></label>
  // 枚举字段统一 Dropdown(路线图规则 3:可枚举输入禁自由文本);选项值为协议原值。
  const select = (key: string, label: string, options: { value: string; label: string }[]) => <label className={FIELD}><span className={LABEL}>{label}</span><Dropdown value={values[key] ?? ''} options={[{ value: '', label: '—' }, ...options]} onChange={(v) => set(key, v)} ariaLabel={label} /></label>
  const enumOpts = (xs: string[]) => xs.map((o) => ({ value: o, label: o }))
  const save = () => {
    const paths: Record<Tab, string> = { grids: '/odn/grids', facilities: '/odn/facilities', sites: '/odn/sites', devices: '/odn/devices', coverage: '', constructions: '' }
    const body = Object.fromEntries(Object.entries(values).map(([k, v]) => {
      if (v === '') return [k, undefined]
      if (/^-?\d+(\.\d+)?$/.test(v)) return [k, Number(v)]
      return [k, v]
    }))
    void submit(body, paths[tab])
  }
  return <div className={`${CARD} mt-4 p-4`}><div className="grid grid-cols-2 gap-3 md:grid-cols-4">
    {tab === 'grids' && <>{field('gridCode', g.gridCode, '01~99')}{field('name', g.name)}{field('coverage', g.coverage)}{select('status', g.status, enumOpts(['ACTIVE', 'RESERVED']))}</>}
    {tab === 'facilities' && <>{field('code', g.code, 'P01001')}{select('kind', g.kind, enumOpts(['P', 'MH', 'TW', 'CLS', 'TBX']))}{select('gridCode', g.gridCode, grids.map((x) => ({ value: String(x.gridCode), label: String(x.gridCode).padStart(2, '0') + ' ' + x.name })))}{field('name', g.name)}{field('lat', g.lat, '14.55')}{field('lng', g.lng, '120.98')}</>}
    {tab === 'sites' && <>{field('siteNo', g.siteNo, '001~999')}{field('name', g.name)}{field('lat', g.lat, '14.55')}{field('lng', g.lng, '120.98')}</>}
    {tab === 'devices' && <>{field('code', g.deviceCode, 'OLT001 / ODB001-2')}{select('kind', g.deviceKind, enumOpts(['SNW', 'OLT', 'ODF', 'OCC', 'ODB', 'OBD', 'SDB', 'SBD', 'PRT', 'TBP']))}{field('siteNo', g.siteNo)}{select('parentId', g.parentId, devices.map((x) => ({ value: String(x.id), label: x.code + (x.name ? ' ' + x.name : '') })))}{field('lat', g.lat, '14.55')}{field('lng', g.lng, '120.98')}</>}
  </div><div className="mt-3 flex justify-end"><ToolbarButton primary disabled={busy} onClick={save}>{busy ? g.saving : g.save}</ToolbarButton></div></div>
}

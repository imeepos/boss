// ODN 资源树(designs/oss-odn-v2.spec.md §2):省 L1 → 市 L2 → 分组 L3 → 资源节点 L4。
// L3 行 = 折叠开关+图标+名称+计数 badge;L4 行 = 类型图标+编码/名称+状态点。
// 计数缺接口时显示已加载行数(spec §2:不阻塞渲染,不为计数新增后端契约)。
import { useMemo, useState, type ReactNode } from 'react'
import { Building2, ChevronRight, Folder, HardHat, LayoutGrid, MapPin, Search, Server } from 'lucide-react'
import { Input } from '../../../components/ui/input'
import { Card } from '../../../components/ui/card'
import type { Grid, Facility, Site, Region, City } from './forms'
import type { ResourceRow } from '../types'

export type GroupKey = 'olts' | 'grids' | 'sites' | 'constructions'

/** 施工项目树内最小投影(constructions.tsx Project 的子集)。 */
export type ConstructionLite = { id: number; projNo: string; name: string; status: string }

export interface TreeData {
  grids: Grid[]
  facilities: Facility[]
  sites: Site[]
  olts: ResourceRow[]
  constructions: ConstructionLite[]
}

/** L4 聚焦态:主区过滤联动由页面实现,树只上抛 (group,key)。 */
export type LeafFocus = { group: GroupKey; key: string } | null

export interface TreeTexts {
  title: string
  search: string
  groupOlts: string
  groupGrids: string
  groupSites: string
  groupConstructions: string
}

interface TreeProps {
  regions: Region[]
  cities: City[]
  prv: string
  city: string
  data: TreeData
  focus: LeafFocus
  g: TreeTexts
  onPrv: (v: string) => void
  onCity: (v: string) => void
  onLeaf: (group: GroupKey, key: string) => void
}

type Dot = 'ok' | 'warn' | 'off'

// 状态点(spec §2/§6):绿=ACTIVE/IN_USE/IN_SERVICE/ONLINE/ACCEPTED;琥珀(金色令牌)=RESERVED/容量预警/BUILDING;灰=RETIRED/PENDING。
function dotOf(status: string, warn?: boolean): Dot {
  if (warn || status === 'RESERVED' || status === 'BUILDING') return 'warn'
  if (status === 'ACTIVE' || status === 'IN_USE' || status === 'IN_SERVICE' || status === 'ONLINE' || status === 'ACCEPTED') return 'ok'
  return 'off'
}

const DOT_BG: Record<Dot, string> = {
  ok: 'bg-[var(--color-success)]',
  warn: 'bg-[var(--color-brand-gold-500)]',
  off: 'bg-[var(--color-text-tertiary)]',
}

type Leaf = { group: GroupKey; key: string; icon: 'olt' | 'grid' | 'facility' | 'site' | 'construction'; code: string; name: string; dot: Dot }

const ROW = 'flex h-9 cursor-pointer select-none items-center gap-1.5 rounded-sm pr-2 text-[13px] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]'
const ROW_ACTIVE = ' bg-[var(--shell-menu-active-bg)] font-medium text-[var(--shell-menu-active-text)]'
const ICON = 'h-3.5 w-3.5 shrink-0 text-[var(--color-text-tertiary)]'

function LeafIcon({ kind }: { kind: Leaf['icon'] }) {
  if (kind === 'olt') return <Server className={ICON} />
  if (kind === 'grid') return <LayoutGrid className={ICON} />
  if (kind === 'site') return <Building2 className={ICON} />
  if (kind === 'construction') return <HardHat className={ICON} />
  return <MapPin className={ICON} />
}

// 行组件:L1-L3 带折叠开关(可传外部受控 open/onToggle),L4 传 open=null 渲染占位对齐。
function Row({ depth, open, onToggle, icon, label, count, dot, selected, onClick }: {
  depth: number
  open: boolean | null
  onToggle: () => void
  icon: ReactNode
  label: string
  count?: number
  dot?: Dot
  selected: boolean
  onClick: () => void
}) {
  const leaf = open === null
  return (
    <div role="treeitem" aria-selected={selected} aria-label={label} onClick={onClick}
      className={ROW + (selected ? ROW_ACTIVE : '')} style={{ paddingLeft: 8 + depth * 16 }}>
      {leaf ? <span className="w-5 shrink-0" /> : (
        <button type="button" aria-label="toggle" className="flex h-5 w-5 shrink-0 cursor-pointer items-center justify-center border-none bg-transparent p-0"
          onClick={(e) => { e.stopPropagation(); onToggle() }}>
          <ChevronRight className={'h-3 w-3 text-[var(--color-text-tertiary)] transition-transform ' + (open ? 'rotate-90' : '')} />
        </button>
      )}
      {icon}
      <span className="min-w-0 truncate">{label}</span>
      {count !== undefined && <span className="ml-auto shrink-0 rounded-full bg-[var(--shell-menu-hover-bg)] px-1.5 text-[11px] leading-4 text-[var(--color-text-secondary)]">{count}</span>}
      {dot && <span className={'ml-auto h-2 w-2 shrink-0 rounded-full ' + DOT_BG[dot]} aria-hidden="true" />}
    </div>
  )
}

export function ResourceTree({ regions, cities, prv, city, data, focus, g, onPrv, onCity, onLeaf }: TreeProps) {
  const [kw, setKw] = useState('')
  // 展开状态 useState(spec §2);市/省行默认展开(未显式操作过时),分组默认收起。
  const [open, setOpen] = useState<Record<string, boolean>>({})
  const isOpen = (k: string) => (k in open ? open[k] : k.startsWith('c:'))
  const toggle = (k: string) => setOpen((m) => ({ ...m, [k]: !(k in m ? m[k] : k.startsWith('c:')) }))

  // L4 叶子按当前市投影(OLT/施工项目无城市维度,取已加载全量);搜索按 code+name 过滤。
  const leaves = useMemo<Leaf[]>(() => {
    const needle = kw.trim().toLowerCase()
    const hit = (s: string) => !needle || s.toLowerCase().includes(needle)
    const out: Leaf[] = []
    for (const r of data.grids.filter((x) => x.cityPrefix === city)) {
      if (hit(r.gridCode + ' ' + r.name)) out.push({ group: 'grids', key: 'g:' + r.gridCode, icon: 'grid', code: String(r.gridCode).padStart(2, '0'), name: r.name, dot: dotOf(r.status, r.warn) })
    }
    for (const f of data.facilities.filter((x) => x.cityPrefix === city)) {
      if (hit(f.code + ' ' + f.name)) out.push({ group: 'grids', key: 'f:' + f.code, icon: 'facility', code: f.code, name: f.name, dot: dotOf(f.status) })
    }
    for (const s of data.sites.filter((x) => x.cityPrefix === city)) {
      if (hit(s.cityPrefix + String(s.siteNo).padStart(3, '0') + ' ' + s.name)) out.push({ group: 'sites', key: 's:' + s.siteNo, icon: 'site', code: s.cityPrefix + String(s.siteNo).padStart(3, '0'), name: s.name, dot: dotOf(s.status) })
    }
    for (const o of data.olts) {
      if (hit(o.code + ' ' + o.name)) out.push({ group: 'olts', key: 'o:' + o.id, icon: 'olt', code: o.code, name: o.name, dot: dotOf(o.status) })
    }
    for (const c of data.constructions) {
      if (hit(c.projNo + ' ' + c.name)) out.push({ group: 'constructions', key: 'c:' + c.id, icon: 'construction', code: c.projNo, name: c.name, dot: dotOf(c.status) })
    }
    return out
  }, [city, data, kw])

  const byGroup = useMemo(() => {
    const m: Record<GroupKey, Leaf[]> = { olts: [], grids: [], sites: [], constructions: [] }
    for (const l of leaves) m[l.group].push(l)
    return m
  }, [leaves])

  const GROUPS: { key: GroupKey; label: string }[] = [
    { key: 'olts', label: g.groupOlts },
    { key: 'grids', label: g.groupGrids },
    { key: 'sites', label: g.groupSites },
    { key: 'constructions', label: g.groupConstructions },
  ]

  const region = regions.find((r) => r.prvCode === prv)

  return (
    <Card>
      <h3 className="m-0 px-3 pt-3 text-[13px] font-semibold text-[var(--shell-heading)]">{g.title}</h3>
      <div className="relative p-3">
        <Search className="pointer-events-none absolute left-5 top-1/2 h-3.5 w-3.5 -translate-y-1/2 text-[var(--color-text-tertiary)]" />
        <Input value={kw} onChange={(e) => setKw(e.target.value)} placeholder={g.search} aria-label={g.search}
          className="h-8 w-full pl-8 pr-2 text-[13px]" />
      </div>
      <div role="tree" className="max-h-[calc(100vh-230px)] overflow-y-auto pb-2">
        {region && (
          <Row depth={0} open={isOpen('r:' + prv)} onToggle={() => toggle('r:' + prv)}
            icon={<MapPin className={ICON} />} label={region.prvCode + ' ' + region.name}
            selected={false} onClick={() => { onPrv(region.prvCode); toggle('r:' + prv) }} />
        )}
        {cities.map((ct) => {
          const ck = 'c:' + ct.cityPrefix
          return (
            <div key={ct.cityPrefix}>
              <Row depth={1} open={isOpen(ck)} onToggle={() => toggle(ck)}
                icon={<Building2 className={ICON} />} label={ct.cityPrefix + ' ' + ct.name}
                selected={ct.cityPrefix === city} onClick={() => { onCity(ct.cityPrefix); toggle(ck) }} />
              {isOpen(ck) && GROUPS.map((grp) => {
                const gk = ck + ':' + grp.key
                const items = byGroup[grp.key]
                return (
                  <div key={grp.key}>
                    <Row depth={2} open={isOpen(gk)} onToggle={() => toggle(gk)}
                      icon={<Folder className={ICON} />} label={grp.label} count={items.length}
                      selected={false} onClick={() => toggle(gk)} />
                    {isOpen(gk) && items.map((l) => (
                      <Row key={l.group + l.key} depth={3} open={null} onToggle={() => undefined}
                        icon={<LeafIcon kind={l.icon} />} label={l.name ? l.code + ' ' + l.name : l.code}
                        dot={l.dot} selected={focus !== null && focus.group === l.group && focus.key === l.key}
                        onClick={() => onLeaf(l.group, l.key)} />
                    ))}
                  </div>
                )
              })}
            </div>
          )
        })}
      </div>
    </Card>
  )
}
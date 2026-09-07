// ODN 关联链拓扑(spec oss-odn-v2 §3 / oss-odn-relation-v1):
// OLT → 局点 → 分光器 → 接头盒/终端盒 → 覆盖,横向节点卡(tinted 图标+编码+两行说明)。
// 逻辑桥接:OLT 取资源域 /resources,同市按编码段(OLT-<CITY>-NN)关联;覆盖取
// /odn/coverage/list 按 facilityCode/deviceId 归 chain。节点可点击切换候选。
import { useState } from 'react'
import { ChevronRight, HardHat, Server, Building2, GitBranch, Box, Radio } from 'lucide-react'
import type { ResourceRow } from '../types'
import type { Facility, Site, Device } from './forms'

export interface ChainCoverage { facilityCode: string; deviceId: number; status: string }
export interface ChainTexts {
  title: string
  olt: string
  site: string
  splitter: string
  closure: string
  terminal: string
  coverage: string
  servedCount: string
  empty: string
}

interface ChainProps {
  olts: ResourceRow[]
  sites: Site[]
  devices: Device[]
  facilities: Facility[]
  coverages: ChainCoverage[]
  city: string
  g: ChainTexts
}

const CARD = 'rounded-md border border-[var(--shell-card-border)] bg-[var(--shell-card-bg)] p-4 shadow-[var(--shell-card-shadow)]'
const NODE = 'flex w-[128px] shrink-0 cursor-pointer flex-col items-center gap-1.5 rounded-md border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-2 py-3 text-center hover:border-[var(--shell-input-border-hover)]'
const NODE_ACTIVE = ' border-[var(--color-info)] ring-1 ring-[color-mix(in_srgb,var(--color-info)_35%,transparent)]'

type Cand = { key: string; code: string; line2: string }

// 节点卡:tinted 图标 + 编码(mono)+ 阶段名/状态两行说明;多候选点击轮换。
function Node({ cand, stage, icon, tone, active, onClick }: { cand: Cand; stage: string; icon: typeof Server; tone: string; active: boolean; onClick: () => void }) {
  const Icon = icon
  return <div className={NODE + (active ? NODE_ACTIVE : '')} onClick={onClick} role="button" aria-label={stage + ' ' + cand.code}>
    <span className="flex h-9 w-9 items-center justify-center rounded-md" style={{ background: tone }}>
      <Icon className="h-5 w-5 text-[var(--shell-heading)]" />
    </span>
    <span className="w-full truncate font-mono text-[12px] font-medium text-[var(--shell-heading)]">{cand.code}</span>
    <span className="text-[11px] leading-4 text-[var(--color-text-secondary)]">{stage}</span>
    <span className="text-[11px] leading-4 text-[var(--color-text-tertiary)]">{cand.line2}</span>
  </div>
}

const Arrow = () => <ChevronRight className="h-4 w-4 shrink-0 text-[var(--color-text-tertiary)]" />

export function RelationChain({ olts, sites, devices, facilities, coverages, city, g }: ChainProps) {
  // 各阶段候选(市域过滤);OLT 无城市维度,按编码段 OLT-<CITY>-NN 逻辑关联(spec §3 桥接)。
  const cityOlts = olts.filter((o) => o.code.split('-').includes(city))
  const splitters = devices.filter((d) => d.kind === 'ODB' || d.kind === 'OBD')
  const closures = facilities.filter((f) => f.kind === 'CLS')
  const terminals = facilities.filter((f) => f.kind === 'TBX')
  const [oltIdx, setOltIdx] = useState(0)
  const [siteIdx, setSiteIdx] = useState(0)
  const [spIdx, setSpIdx] = useState(0)
  const [clIdx, setClIdx] = useState(0)
  const [tbIdx, setTbIdx] = useState(0)
  const cycle = (n: number, i: number, set: (v: number) => void) => set(n ? (i + 1) % n : 0)
  const at = (xs: unknown[], i: number) => (xs.length ? xs[i % xs.length] : null)

  const olt = at(cityOlts, oltIdx) as ResourceRow | null
  const site = at(sites, siteIdx) as Site | null
  const sp = at(splitters, spIdx) as Device | null
  const cl = at(closures, clIdx) as Facility | null
  const tb = at(terminals, tbIdx) as Facility | null

  // 覆盖:归当前终端盒/接头盒/分光器引用的 SERVED 覆盖行数。
  const chainFacCodes = new Set([cl?.code, tb?.code].filter(Boolean) as string[])
  const chainDevIds = new Set([sp?.id].filter(Boolean) as number[])
  const served = coverages.filter((c) => (c.facilityCode && chainFacCodes.has(c.facilityCode)) || (c.deviceId && chainDevIds.has(c.deviceId)))
  const servedOn = served.filter((c) => c.status === 'SERVED').length

  const isEmpty = !olt && !site && !sp && !cl && !tb
  if (isEmpty) return <div className={CARD}><div className="text-[13px] font-semibold text-[var(--shell-heading)]">{g.title}</div><div className="mt-2 text-xs text-[var(--color-text-tertiary)]">{g.empty}</div></div>

  const covLine = g.servedCount.replace('{served}', String(servedOn)).replace('{total}', String(served.length))
  const tones = {
    olt: 'color-mix(in srgb, var(--color-info) 14%, transparent)',
    site: 'color-mix(in srgb, var(--color-brand-blue-700) 14%, transparent)',
    sp: 'color-mix(in srgb, var(--color-brand-blue-300) 24%, transparent)',
    cl: 'color-mix(in srgb, var(--color-warning) 16%, transparent)',
    tb: 'color-mix(in srgb, var(--color-success) 14%, transparent)',
    cov: 'color-mix(in srgb, var(--color-success) 14%, transparent)',
  }
  return <div className={CARD}>
    <div className="mb-3 text-[13px] font-semibold text-[var(--shell-heading)]">{g.title}</div>
    <div className="flex items-stretch gap-2 overflow-x-auto pb-1">
      {olt && <Node cand={{ key: 'o' + olt.id, code: olt.code, line2: olt.status }} stage={g.olt} icon={Server} tone={tones.olt} active={cityOlts.length > 1} onClick={() => cycle(cityOlts.length, oltIdx, setOltIdx)} />}
      {(olt && site) && <Arrow />}
      {site && <Node cand={{ key: 's' + site.siteNo, code: site.cityPrefix + String(site.siteNo).padStart(3, '0'), line2: site.status }} stage={g.site} icon={Building2} tone={tones.site} active={sites.length > 1} onClick={() => cycle(sites.length, siteIdx, setSiteIdx)} />}
      {(site && sp) && <Arrow />}
      {sp && <Node cand={{ key: 'd' + sp.id, code: sp.code, line2: sp.status }} stage={g.splitter} icon={GitBranch} tone={tones.sp} active={splitters.length > 1} onClick={() => cycle(splitters.length, spIdx, setSpIdx)} />}
      {(sp && cl) && <Arrow />}
      {cl && <Node cand={{ key: 'f' + cl.code, code: cl.code, line2: cl.status }} stage={g.closure} icon={Box} tone={tones.cl} active={closures.length > 1} onClick={() => cycle(closures.length, clIdx, setClIdx)} />}
      {(cl && tb) && <Arrow />}
      {tb && <Node cand={{ key: 't' + tb.code, code: tb.code, line2: tb.status }} stage={g.terminal} icon={Radio} tone={tones.tb} active={terminals.length > 1} onClick={() => cycle(terminals.length, tbIdx, setTbIdx)} />}
      {tb && <Arrow />}
      <Node cand={{ key: 'cov', code: g.coverage, line2: covLine }} stage={g.coverage} icon={HardHat} tone={tones.cov} active={false} onClick={() => undefined} />
    </div>
  </div>
}
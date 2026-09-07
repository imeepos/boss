// ODN 详情抽屉(spec oss-odn-v2 §5 / oss-odn-drawer-detail-v1):480px 四分区——
// 基本信息(两列 k-v)→ 关联关系(纵向链行卡,点击换详情对象)→ 资产信息
// (assetReg 凭证/登记号 badge,无则未登记灰字)→ 操作记录(/audit-logs 按
// targetType 拉取后客户端按 targetId 过滤;无 menu:audit 权限时优雅空态)。
import { useEffect, useState } from 'react'
import { ChevronRight } from 'lucide-react'
import { apiFetch } from '../../../api/client'
import { Drawer } from '../../../components/Drawer'
import { Badge } from '../../../components/ui/badge'
import { useConfirm } from '../../../components/ConfirmDialog'
import type { Tab, Grid, Facility, Site, Device, Region } from './forms'
import type { ResourceRow } from '../types'

export type DetailTarget = { tab: Tab; row: Grid | Facility | Site | Device }
export type AuditLite = { id: number; action: string; targetId: string; detail: string; operator: string; createdAt: string }

interface DetailProps {
  target: DetailTarget
  regions: Region[]
  grids: Grid[]
  facilities: Facility[]
  sites: Site[]
  devices: Device[]
  olts: ResourceRow[]
  city: string
  g: Record<string, any>
  onClose: () => void
  onEdit: (t: DetailTarget) => void
  onRetire: (t: DetailTarget) => void
  onOlt: (o: ResourceRow) => void
  /** 点击链行卡换详情对象(tab 变化由父级裁决)。 */
  onSwitch: (t: DetailTarget) => void
}

const SEC = 'py-4'
const SEC_TITLE = 'mb-3 text-[13px] font-semibold text-[var(--shell-heading)]'
const KV = 'flex flex-col gap-0.5'
const K = 'text-[11px] text-[var(--color-text-tertiary)]'
const V = 'text-[13px] text-[var(--shell-content-text)]'
const CHAIN = 'flex w-full cursor-pointer items-center gap-3 rounded-md border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-3 py-2.5 text-left hover:border-[var(--shell-input-border-hover)]'

function statusBadge(status: string) {
  const variant = status === 'ACTIVE' || status === 'IN_USE' || status === 'IN_SERVICE' || status === 'ONLINE' ? 'success' : status === 'RESERVED' || status === 'BUILDING' || status === 'PENDING' ? 'warning' : 'default'
  return <Badge variant={variant}>{status}</Badge>
}

function Kv({ k, v }: { k: string; v: string }) {
  return <div className={KV}><span className={K}>{k}</span><span className={V}>{v || '-'}</span></div>
}

export function DetailDrawer({ target, regions, grids, facilities, sites, devices, olts, city, g, onClose, onEdit, onRetire, onOlt, onSwitch }: DetailProps) {
  const { tab, row } = target
  const confirmDialog = useConfirm()
  const [audit, setAudit] = useState<AuditLite[]>([])
  const code = tab === 'grids' ? String((row as Grid).gridCode).padStart(2, '0') : tab === 'sites' ? (row as Site).cityPrefix + String((row as Site).siteNo).padStart(3, '0') : (row as Facility).code
  const name = row.name
  const status = row.status

  // 操作记录:targetType 按 odn_facility/odn_site/odn_device 拉取,客户端按 targetId 筛;
  // 无 menu:audit 权限时 403 → 空态(不阻塞详情)。
  useEffect(() => {
    const tt = tab === 'grids' ? '' : tab === 'facilities' ? 'odn_facility' : tab === 'sites' ? 'odn_site' : 'odn_device'
    if (!tt) { setAudit([]); return }
    apiFetch<{ items: AuditLite[] }>('/audit-logs', { query: { targetType: tt, limit: 500 } })
      .then((x) => {
        const key = tab === 'facilities' ? (row as Facility).code : tab === 'sites' ? (row as Site).prvCode + '/' + (row as Site).cityPrefix + '/' + (row as Site).siteNo : String((row as Device).id)
        setAudit((x?.items ?? []).filter((a) => a.targetId === key).slice(0, 8))
      })
      .catch((e) => { console.error('[odn] AUDIT LOAD FAILED', e); setAudit([]) })
  }, [tab, row])

  const regionName = regions.find((r) => r.prvCode === row.prvCode)?.name ?? ''
  const cityOlt = olts.find((o) => o.code.split('-').includes(city))
  const grid = 'gridCode' in row && row.gridCode ? grids.find((x) => x.gridCode === row.gridCode && x.cityPrefix === row.cityPrefix) : undefined

  // 关联链行卡:按实体给可跳转的关联对象(OLT 为资源域,点击交由父级跳设备页)。
  const chain: { key: string; icon: string; tone: string; code: string; name: string; kindLabel: string; go: () => void }[] = []
  if (tab !== 'grids' && cityOlt) chain.push({ key: 'olt', icon: 'OLT', tone: 'var(--color-info)', code: cityOlt.code, name: cityOlt.name, kindLabel: g.kpi.linkedOlt, go: () => onOlt(cityOlt) })
  if (tab === 'devices') {
    const dev = row as Device
    if (dev.siteNo) {
      const site = sites.find((s) => s.siteNo === dev.siteNo && s.cityPrefix === dev.cityPrefix)
      if (site) chain.push({ key: 'site', icon: '局', tone: 'var(--color-brand-blue-700)', code: site.cityPrefix + String(site.siteNo).padStart(3, '0'), name: site.name, kindLabel: g.relSite, go: () => onSwitch({ tab: 'sites', row: site }) })
    }
    if (dev.parentId) {
      const parent = devices.find((x) => x.id === dev.parentId)
      if (parent) chain.push({ key: 'parent', icon: '设备', tone: 'var(--color-brand-blue-300)', code: parent.code, name: parent.name, kindLabel: g.parentDevice, go: () => onSwitch({ tab: 'devices', row: parent }) })
    }
  }
  if (tab === 'facilities' && grid) chain.push({ key: 'grid', icon: '网格', tone: 'var(--color-brand-gold-300)', code: String(grid.gridCode).padStart(2, '0'), name: grid.name, kindLabel: g.relGrid, go: () => onSwitch({ tab: 'grids', row: grid }) })
  if (tab === 'sites') {
    const devChildren = devices.filter((x) => x.siteNo === (row as Site).siteNo).slice(0, 3)
    for (const c of devChildren) chain.push({ key: 'dev' + c.id, icon: '设备', tone: 'var(--color-brand-blue-300)', code: c.code, name: c.name, kindLabel: g.tabs.devices, go: () => onSwitch({ tab: 'devices', row: c }) })
  }
  if (tab === 'grids') {
    const gr = row as Grid
    const facs = facilities.filter((x) => x.cityPrefix === gr.cityPrefix && x.gridCode === gr.gridCode).slice(0, 3)
    for (const fc of facs) chain.push({ key: 'f' + fc.code, icon: '设施', tone: 'var(--color-success)', code: fc.code, name: fc.name, kindLabel: g.tabs.facilities, go: () => onSwitch({ tab: 'facilities', row: fc }) })
  }

  const assetReg = 'assetReg' in row ? row.assetReg : null

  const retire = async () => {
    if (await confirmDialog(g.retireConfirm, { danger: true })) onRetire(target)
  }

  return <Drawer title={g.detail.title} onClose={onClose} width={480}
    footer={<><button type="button" className="mr-auto cursor-pointer border-none bg-transparent p-0 text-[13px] text-[var(--color-danger)] hover:underline" onClick={() => void retire()}>{g.retire}</button>
      <button type="button" className="cursor-pointer rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-4 py-1.5 text-[13px] text-[var(--shell-content-text)] hover:border-[var(--color-border-hover)]" onClick={() => onEdit(target)}>{g.drawer.edit}</button></>}>
    <div className="mb-1 flex items-center gap-2">
      <span className="rounded-sm bg-[var(--shell-menu-hover-bg)] px-2 py-0.5 font-mono text-[12px] text-[var(--shell-heading)]">{code}</span>
      <span className="min-w-0 truncate text-[15px] font-semibold text-[var(--shell-heading)]">{name || code}</span>
      {statusBadge(status)}
    </div>
    <div className="divide-y divide-[var(--shell-side-border)]">
      <section className={SEC + ' pt-3'}>
        <h4 className={SEC_TITLE}>{g.detail.basic}</h4>
        <div className="grid grid-cols-2 gap-x-4 gap-y-3">
          <Kv k={g.prvLabel} v={row.prvCode + ' ' + regionName} />
          <Kv k={g.cityLabel} v={row.cityPrefix} />
          {grid && <Kv k={g.relGrid} v={String(grid.gridCode).padStart(2, '0') + ' ' + grid.name} />}
          <Kv k={g.name} v={name} />
          {'lat' in row && <Kv k={g.lat + '/' + g.lng} v={(row.lat == null ? '-' : String(row.lat)) + ', ' + (row.lng == null ? '-' : String(row.lng))} />}
          <Kv k={g.drawer.lifecycle} v={('lifecycleStatus' in row ? row.lifecycleStatus : '') || status} />
          {tab === 'grids' && <Kv k={g.coverage} v={(row as Grid).coverage} />}
          {tab === 'grids' && <Kv k={g.usage} v={(row as Grid).facilities + '/999'} />}
        </div>
      </section>
      <section className={SEC}>
        <h4 className={SEC_TITLE}>{g.detail.relations}</h4>
        <div className="flex flex-col gap-2">
          {chain.length === 0 && <span className="text-xs text-[var(--color-text-tertiary)]">{g.empty}</span>}
          {chain.map((c) => <button key={c.key} type="button" className={CHAIN} onClick={c.go}>
            <span className="flex h-8 w-8 shrink-0 items-center justify-center rounded-md font-mono text-[10px] font-semibold" style={{ background: 'color-mix(in srgb, ' + c.tone + ' 14%, transparent)', color: c.tone }}>{c.icon}</span>
            <span className="min-w-0 flex-1"><span className="block truncate font-mono text-[13px] text-[var(--shell-heading)]">{c.code}</span><span className="block truncate text-[11px] text-[var(--color-text-secondary)]">{c.kindLabel}{c.name ? ' ' + c.name : ''}</span></span>
            <ChevronRight className="h-4 w-4 shrink-0 text-[var(--color-text-tertiary)]" />
          </button>)}
        </div>
      </section>
      <section className={SEC}>
        <h4 className={SEC_TITLE}>{g.detail.asset}</h4>
        {assetReg ? <div className="flex flex-col gap-1"><span className={K}>{g.detail.regNo}</span><span><Badge variant="warning">{assetReg.registrationNo}</Badge> <span className="ml-2 font-mono text-[12px] text-[var(--shell-content-text)]">{assetReg.assetCode}</span></span></div>
          : <span className="text-[13px] text-[var(--color-text-tertiary)]">{g.detail.noAsset}</span>}
      </section>
      <section className={SEC}>
        <h4 className={SEC_TITLE}>{g.detail.records}</h4>
        {audit.length === 0 ? <span className="text-[13px] text-[var(--color-text-tertiary)]">{g.detail.noRecords}</span> : <ol className="ml-1 flex flex-col gap-3 border-l border-[var(--shell-side-border)] pl-4">
          {audit.map((a) => <li key={a.id} className="relative">
            <span className="absolute -left-[21px] top-1.5 h-2 w-2 rounded-full bg-[var(--color-text-tertiary)]" />
            <div className="text-[11px] text-[var(--color-text-tertiary)]">{a.createdAt}{a.operator ? ' · ' + a.operator : ''}</div>
            <div className="text-[13px] text-[var(--shell-content-text)]">{a.action} {a.detail && a.detail !== '{}' ? a.detail : ''}</div>
          </li>)}
        </ol>}
      </section>
    </div>
  </Drawer>
}
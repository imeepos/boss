// ODN 点位反查取数编排(T14-1):仅编排既有只读接口,零后端改动。
// 点位口径(internal/domain/gis/pg_odn_points):9=设施(name=编码) 10=局点(name=城市前缀+3位序号)
// 11=设备(name=编码,真实 id)。绑定反查:设备→端口→/odn/bindings?portId(仅非 IDLE 端口)。
import { apiFetch } from '../../../api/client'
import type {
  OdnBindingRow, OdnCoverageResolved, OdnDeviceRow, OdnFacilityRow, OdnGridRow, OdnPortRow, OdnSiteRow,
} from '../types'

export type OdnEntity = 'facility' | 'site' | 'device'

// 点位 level → ODN 实体;非 ODN 点位返回 null(与 drill 点位共点位层并存)。
export function odnEntityOf(level: number): OdnEntity | null {
  if (level === 9) return 'facility'
  if (level === 10) return 'site'
  if (level === 11) return 'device'
  return null
}

export interface OdnBindingView {
  portLabel: string
  portStatus: string
  orderId: number
  boundAt: string
  note: string
}

export interface OdnReverseData {
  entity: OdnEntity
  code: string
  name: string
  kind: string      // 原始类型码:设施 P/MH/TW/CLS/TBX;设备 SNW/OLT/...;局点空
  status: string    // IN_USE/ACTIVE(点位层已过滤,兜底保留)
  lifecycle: string // PLANNED/IN_BUILD/IN_SERVICE/RETIRED
  lat: number
  lng: number
  gridName: string  // 所属网格名(设施);空=无
  siteName: string  // 局点名(局点自身/设备所属);空=无
  parentName: string // 上级设备名(设备);空=顶层
  bindings: OdnBindingView[]
  coverage: OdnCoverageResolved | null
}

async function loadFacility(code: string): Promise<{ f: OdnFacilityRow; gridName: string }> {
  const f = await apiFetch<OdnFacilityRow>('/odn/facilities/' + code)
  if (!f) throw new Error('facility not found: ' + code)
  let gridName = ''
  if (f.gridCode > 0 && f.prvCode && f.cityPrefix) {
    const grids = (await apiFetch<OdnGridRow[]>('/odn/grids', { query: { prvCode: f.prvCode, cityPrefix: f.cityPrefix } })) ?? []
    gridName = grids.find((x) => x.gridCode === f.gridCode)?.name ?? ''
  }
  return { f, gridName }
}

// 局点反查:点位 name=cityPrefix+siteNo;prvCode 须经设备列表(只读)旁证后按复合键取局点。
async function loadSite(nodeCode: string): Promise<OdnSiteRow | null> {
  const cityPrefix = nodeCode.slice(0, -3)
  const siteNo = Number(nodeCode.slice(-3))
  if (!cityPrefix || !siteNo) return null
  const devs = (await apiFetch<OdnDeviceRow[]>('/odn/devices')) ?? []
  const prvCode = devs.find((d) => d.cityPrefix === cityPrefix)?.prvCode ?? ''
  if (!prvCode) return null
  const sites = (await apiFetch<OdnSiteRow[]>('/odn/sites', { query: { prvCode, cityPrefix } })) ?? []
  return sites.find((s) => s.siteNo === siteNo) ?? null
}

// 设备绑定视图:非 IDLE 端口逐口反查绑定(一口一绑定,数量有界)。
async function loadBindings(deviceId: number): Promise<OdnBindingView[]> {
  const ports = (await apiFetch<OdnPortRow[]>('/odn/devices/' + deviceId + '/ports')) ?? []
  const active = ports.filter((p) => p.status !== 'IDLE')
  const out: OdnBindingView[] = []
  for (const p of active) {
    const rows = (await apiFetch<OdnBindingRow[]>('/odn/bindings', { query: { portId: String(p.id) } })) ?? []
    for (const b of rows) {
      out.push({ portLabel: 'P' + p.portNo, portStatus: p.status, orderId: b.orderId, boundAt: b.boundAt, note: b.note })
    }
  }
  return out
}

// 设备反查:无过滤列表一次取回(设备父子异 kind,不能按子 kind 过滤);id 优先,编码兜底。
function findDevice(devs: OdnDeviceRow[], point: { id: number; name: string }): OdnDeviceRow {
  const d = devs.find((x) => x.id === point.id) ?? devs.find((x) => x.code === point.name)
  if (!d) throw new Error('device not found: ' + point.name)
  return d
}

// 反查主入口:资源信息 + 关联资源 + 业务绑定 + 就近可装性,一次拉齐由抽屉分区渲染。
export async function loadOdnReverse(point: { id: number; name: string; level: number; lng: number; lat: number }): Promise<OdnReverseData> {
  const entity = odnEntityOf(point.level)
  if (!entity) throw new Error('not an odn point: level ' + point.level)
  let code = point.name
  let name = point.name
  let kind = ''
  let status = ''
  let lifecycle = ''
  let gridName = ''
  let siteName = ''
  let parentName = ''
  let lat = point.lat
  let lng = point.lng
  let bindings: OdnBindingView[] = []
  if (entity === 'facility') {
    const { f, gridName: gn } = await loadFacility(point.name)
    kind = f.kind; status = f.status; lifecycle = f.lifecycleStatus; gridName = gn
    name = f.name || point.name; lat = f.lat || point.lat; lng = f.lng || point.lng
  } else if (entity === 'site') {
    const s = await loadSite(point.name)
    if (s) { status = s.status; lifecycle = s.lifecycleStatus; name = s.name || point.name; siteName = s.name || point.name }
    kind = ''
  } else {
    const devs = (await apiFetch<OdnDeviceRow[]>('/odn/devices')) ?? []
    const d = findDevice(devs, point)
    code = d.code; name = d.name || d.code; kind = d.kind; status = d.status; lifecycle = d.lifecycleStatus
    lat = d.lat ?? point.lat; lng = d.lng ?? point.lng
    if (d.siteNo > 0 && d.prvCode && d.cityPrefix) {
      const sites = (await apiFetch<OdnSiteRow[]>('/odn/sites', { query: { prvCode: d.prvCode, cityPrefix: d.cityPrefix } })) ?? []
      siteName = sites.find((s) => s.siteNo === d.siteNo)?.name ?? ''
    }
    if (d.parentId > 0) parentName = devs.find((x) => x.id === d.parentId)?.name ?? ''
    bindings = await loadBindings(d.id)
  }
  const coverage = await apiFetch<OdnCoverageResolved>('/odn/coverage/resolve', { query: { lat: String(lat), lng: String(lng) } })
  return { entity, code, name, kind, status, lifecycle, lat, lng, gridName, siteName, parentName, bindings, coverage: coverage ?? null }
}
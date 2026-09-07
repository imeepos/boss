// 代客下单可装性判定(T14-2):地址→登记覆盖→坐标就近判定,全链既有只读接口。
// 判定顺序:address_coverage 登记值权威(SERVED/PENDING/UNSERVED 三态);
// 未登记再走 /gis/points 取坐标 + /odn/coverage/resolve 就近判定(两态);无坐标=无法评估。
import { apiFetch } from '../../../api/client'

export interface CoverageRow {
  id: number
  addressId: number
  facilityCode: string
  deviceId: number
  status: string // SERVED/PENDING/UNSERVED
  note: string
  addressName?: string
  updatedAt: string
}

export interface ResolveRow {
  status: string // SERVED/UNSERVED(distanceM omitempty:自指 0 被省略)
  facilityCode?: string
  facilityName?: string
  distanceM?: number
}

export type Servability =
  | { kind: 'registered'; status: string }
  | { kind: 'proximity'; status: string; distanceM: number; facilityCode: string; facilityName: string }
  | { kind: 'no-coords' }

interface GisPointLite { id: number; lng: number; lat: number }

// gis 点位(level 1-5,point.id=addresses.id)按层缓存;TTL 5 分钟避免长期 stale。
const coordCache = new Map<number, { at: number; map: Map<number, [number, number]> }>()
const CACHE_TTL_MS = 5 * 60 * 1000

async function levelCoords(level: number): Promise<Map<number, [number, number]>> {
  const hit = coordCache.get(level)
  if (hit && Date.now() - hit.at < CACHE_TTL_MS) return hit.map
  const rows = (await apiFetch<{ items: GisPointLite[] }>('/gis/points', { query: { level: String(level), parentId: '0' } }))?.items ?? []
  const map = new Map<number, [number, number]>()
  for (const r of rows) map.set(r.id, [r.lat, r.lng])
  coordCache.set(level, { at: Date.now(), map })
  return map
}

async function addressCoords(addressId: number): Promise<[number, number] | null> {
  for (let level = 1; level <= 5; level++) {
    const map = await levelCoords(level)
    const hit = map.get(addressId)
    if (hit) return hit
  }
  return null
}

export async function resolveServability(addressId: number): Promise<Servability> {
  const row = await apiFetch<CoverageRow | null>('/odn/coverage', { query: { addressId: String(addressId) } })
  if (row && row.status) return { kind: 'registered', status: row.status }
  const coords = await addressCoords(addressId)
  if (!coords) return { kind: 'no-coords' }
  const res = await apiFetch<ResolveRow>('/odn/coverage/resolve', { query: { lat: String(coords[0]), lng: String(coords[1]) } })
  if (!res || !res.status) return { kind: 'no-coords' }
  return { kind: 'proximity', status: res.status, distanceM: res.distanceM ?? 0, facilityCode: res.facilityCode ?? '', facilityName: res.facilityName ?? '' }
}
// 区划级联数据源与纯逻辑（node 环境可单测）。
// 契约（U1 已合入 main 的 1d9eab53 实测口径）：GET /geo/subdivisions 查询参数为
//   country（旧参数名兼容，非 countryCode）/ parentCode（三态：缺省不过滤、空串=顶层、
//   非空=直接子节点）/ keyword（须配 country，编码+全译名 ILIKE）/ limit（默认50最大200）；
// 行字段 code/displayName（译名）/level/hasChildren/parentCode/countryCode。
// GET /geo/default-country 返回 {countryCode,configured}，未配置为空值对象，前端兜底 PH。
// 字段口径 docs/contract/fields.md 1.5.1；懒加载调用序列与回显展开在此收敛，UI 层只消费。
import { apiFetch } from '../../api/client'

export interface SubdivRow {
  code: string
  name: string
  level: number
  hasChildren: boolean
  parentCode?: string | null
  countryCode?: string
}

export interface RegionSelection {
  countryCode: string
  code: string
  /** 完整展示链：[国家名, 一级译名, ...选中级译名]，便于表单回显。 */
  names: string[]
}

export interface SubdivQuery {
  /** 国家 alpha2；缺省=不过滤（keyword 搜索必须配 country）。 */
  countryCode?: string
  /** 三态：undefined=不过滤；''=顶层节点；非空=该父码直接子节点。 */
  parentCode?: string
  keyword?: string
  limit?: number
}

export interface CountryLite {
  alpha2: string
  displayName: string
}

export const SUBDIV_LIMIT = 200
export const DEFAULT_COUNTRY_FALLBACK = 'PH'
export const DIRECT_SEARCH_DEBOUNCE_MS = 300
const MAX_PATH_DEPTH = 8

export type FetchLike = <T = unknown>(
  path: string,
  opts?: { method?: string; body?: unknown; query?: Record<string, string | number | undefined> },
) => Promise<T | null>

// 手动拼查询串：apiFetch 的 toQuery 会丢弃空字符串值，而 parentCode='' 正是 U1 的"顶层节点"
// 语义，必须原样发出，故查询串并入 path 而非 opts.query。
function buildQuery(q: Record<string, string | number | undefined>): string {
  const parts: string[] = []
  for (const [k, v] of Object.entries(q)) {
    if (v === undefined) continue
    parts.push(`${k}=${encodeURIComponent(String(v))}`)
  }
  return parts.length ? `?${parts.join('&')}` : ''
}

/** 区划列表：按 U1 契约拼查询参数（limit 默认 200），行字段归一化（name 缺省回退 displayName）。 */
export async function fetchSubdivisions(query: SubdivQuery, fetchImpl: FetchLike = apiFetch): Promise<SubdivRow[]> {
  const q: Record<string, string | number | undefined> = { limit: query.limit ?? SUBDIV_LIMIT }
  if (query.countryCode) q.country = query.countryCode
  if (query.parentCode !== undefined) q.parentCode = query.parentCode
  if (query.keyword) q.keyword = query.keyword
  const rows = (await fetchImpl<Array<Record<string, unknown>>>('/geo/subdivisions' + buildQuery(q))) ?? []
  return rows.map(normalizeRow)
}

function normalizeRow(raw: Record<string, unknown>): SubdivRow {
  const r = raw as Partial<SubdivRow> & { displayName?: unknown }
  const code = String(r.code ?? '')
  return {
    code,
    name: String(r.name ?? r.displayName ?? code),
    level: Number(r.level ?? 0),
    hasChildren: Boolean(r.hasChildren ?? false),
    parentCode: r.parentCode == null ? null : String(r.parentCode),
    countryCode: r.countryCode == null ? undefined : String(r.countryCode),
  }
}

/** 默认国家：兼容 {countryCode}/{alpha2} 对象与裸字符串；空值/解析失败/请求失败一律兜底 PH（契约 N3）。 */
export async function fetchDefaultCountry(fetchImpl: FetchLike = apiFetch): Promise<string> {
  try {
    const alpha2 = pickAlpha2(await fetchImpl<unknown>('/geo/default-country'))
    if (alpha2) return alpha2
  } catch (err) {
    console.warn('[region-picker] default-country fetch failed, fallback PH:', (err as Error)?.message ?? err)
  }
  return DEFAULT_COUNTRY_FALLBACK
}

function pickAlpha2(body: unknown): string {
  if (typeof body === 'string') return body.trim().toUpperCase()
  if (body && typeof body === 'object') {
    const obj = body as Record<string, unknown>
    const v = obj.alpha2 ?? obj.countryCode
    if (typeof v === 'string' && v.trim()) return v.trim().toUpperCase()
  }
  return ''
}

/** 尾缘防抖（直搜用）；返回带 cancel 的可取消函数，便于卸载清理与单测。 */
export function debounce<A extends unknown[]>(fn: (...args: A) => void, ms: number): ((...args: A) => void) & { cancel: () => void } {
  let timer: ReturnType<typeof setTimeout> | null = null
  const wrapped = (...args: A): void => {
    if (timer !== null) clearTimeout(timer)
    timer = setTimeout(() => { timer = null; fn(...args) }, ms)
  }
  wrapped.cancel = () => {
    if (timer !== null) { clearTimeout(timer); timer = null }
  }
  return wrapped
}

/** 回显展开（契约 N4）：按已存 code 反查节点，沿 parentCode 逐级上溯成 [一级..目标] 链。
 *  行无 parentCode 时退化为单节点链（不拉全量），由 UI 展示并告警。 */
export async function resolvePath(countryCode: string, code: string, fetchImpl: FetchLike = apiFetch): Promise<SubdivRow[]> {
  const chain: SubdivRow[] = []
  let current: SubdivRow | null = await findNode(countryCode, code, fetchImpl)
  if (!current) return chain
  chain.push(current)
  let guard = 0
  // 沿 parentCode 上溯到一级区划为止（level===1 时其父即国家，不再上溯）
  while (current.level > 1 && current.parentCode && guard < MAX_PATH_DEPTH) {
    guard += 1
    const parent = await findNode(countryCode, current.parentCode, fetchImpl)
    if (!parent) break
    chain.push(parent)
    current = parent
  }
  return chain.reverse()
}

async function findNode(countryCode: string, code: string, fetchImpl: FetchLike): Promise<SubdivRow | null> {
  const rows = await fetchSubdivisions({ countryCode, keyword: code, limit: 50 }, fetchImpl)
  return rows.find((r) => r.code === code) ?? null
}

/** 级联数据源：懒加载下钻（parentCode）、直搜、默认国家、回显展开的统一入口；
 *  fetchImpl 可注入，单测直接驱动调用序列断言。 */
export class RegionCascadeSource {
  constructor(private readonly fetchImpl: FetchLike = apiFetch) {}

  countries(): Promise<CountryLite[]> {
    return this.fetchImpl<CountryLite[]>('/geo/countries').then((rows) => rows ?? [])
  }

  defaultCountry(): Promise<string> {
    return fetchDefaultCountry(this.fetchImpl)
  }

  /** parentCode 为空串=取该国顶层区划（U1 三态语义）。 */
  children(parentCode: string, countryCode: string): Promise<SubdivRow[]> {
    return fetchSubdivisions({ countryCode, parentCode, limit: SUBDIV_LIMIT }, this.fetchImpl)
  }

  search(countryCode: string, keyword: string): Promise<SubdivRow[]> {
    return fetchSubdivisions({ countryCode, keyword, limit: SUBDIV_LIMIT }, this.fetchImpl)
  }

  resolvePath(countryCode: string, code: string): Promise<SubdivRow[]> {
    return resolvePath(countryCode, code, this.fetchImpl)
  }
}

// 服务端配置:多套接口地址存 localStorage(仅本机,不上后端),后端 CORS 直连。
// 无内置默认环境:未配置/未启用时由 UI 门禁(ServerManagerDialog)强制先完成配置。
export interface ServerConfig {
  id: string
  name: string
  baseUrl: string
}

export interface ServerDraft {
  id?: string
  name: string
  baseUrl: string
}

const LIST_KEY = 'boss.servers'
const ACTIVE_KEY = 'boss.server.active'
export const API_PREFIX = '/api/v1'

const storage = (): Storage | null =>
  typeof localStorage === 'undefined' ? null : localStorage

function readStored(): ServerConfig[] {
  try {
    const raw = storage()?.getItem(LIST_KEY)
    const parsed = raw ? (JSON.parse(raw) as unknown) : []
    if (!Array.isArray(parsed)) return []
    return parsed.filter(
      (it): it is ServerConfig =>
        !!it && typeof (it as ServerConfig).id === 'string'
          && typeof (it as ServerConfig).name === 'string'
          && typeof (it as ServerConfig).baseUrl === 'string',
    )
  } catch {
    return []
  }
}

export function listServers(): ServerConfig[] {
  return readStored()
}

export function saveServers(list: ServerConfig[]): void {
  storage()?.setItem(LIST_KEY, JSON.stringify(list))
}

export function activeServerId(): string | null {
  return storage()?.getItem(ACTIVE_KEY) ?? null
}

export function setActiveServerId(id: string | null): void {
  const s = storage()
  if (!id) s?.removeItem(ACTIVE_KEY)
  else s?.setItem(ACTIVE_KEY, id)
}

/** 当前生效配置:active 未设置或悬空(指向已删项)返回 null。 */
export function activeServer(): ServerConfig | null {
  const id = activeServerId()
  return id ? (readStored().find((it) => it.id === id) ?? null) : null
}

/** 请求基址:生效配置 baseUrl(去尾斜杠)+ /api/v1;未配置时相对前缀仅为兜死占位(UI 门禁拦在前)。 */
export function apiBaseUrl(): string {
  const base = activeServer()?.baseUrl
  return base ? `${base.replace(/\/+$/, '')}${API_PREFIX}` : API_PREFIX
}

export function newServerId(): string {
  return typeof crypto !== 'undefined' && 'randomUUID' in crypto
    ? crypto.randomUUID()
    : `srv-${Date.now()}-${Math.random().toString(36).slice(2, 8)}`
}

/** 规整地址:去首尾空白与尾部斜杠。 */
export function normalizeBaseUrl(input: string): string {
  return input.trim().replace(/\/+$/, '')
}

/** 校验草稿(名称必填;地址必填且 http(s) 合法),空对象 = 通过。 */
export function validateDraft(draft: ServerDraft): { name?: string; baseUrl?: string } {
  const errors: { name?: string; baseUrl?: string } = {}
  if (!draft.name.trim()) errors.name = 'required'
  const url = normalizeBaseUrl(draft.baseUrl)
  if (!url) {
    errors.baseUrl = 'required'
    return errors
  }
  let parsed: URL
  try {
    parsed = new URL(url)
  } catch {
    errors.baseUrl = 'invalid'
    return errors
  }
  if (!/^https?:$/.test(parsed.protocol)) errors.baseUrl = 'invalid'
  return errors
}

/** 重名检测(排除自身 id)。 */
export function isDuplicateName(list: { id?: string; name: string }[], draft: ServerDraft): boolean {
  return list.some((it) => it.id !== draft.id && it.name.trim() === draft.name.trim())
}

export type UpsertResult =
  | { ok: true; servers: ServerConfig[] }
  | { ok: false; error: 'name' | 'url' }

/** 新增/编辑一条配置并落库(含校验与重名检测)。 */
export function upsertServer(draft: ServerDraft): UpsertResult {
  const errors = validateDraft(draft)
  const servers = readStored()
  if (isDuplicateName(servers, draft)) errors.name = 'duplicate'
  if (errors.name || errors.baseUrl) return { ok: false, error: errors.name ? 'name' : 'url' }
  const item: ServerConfig = {
    id: draft.id ?? newServerId(),
    name: draft.name.trim(),
    baseUrl: normalizeBaseUrl(draft.baseUrl),
  }
  const next = draft.id ? servers.map((it) => (it.id === draft.id ? item : it)) : [...servers, item]
  saveServers(next)
  return { ok: true, servers: next }
}

/** 删除一条配置;若删的是当前生效项则清空 active(由门禁重新引导选择)。 */
export function removeServer(id: string): { servers: ServerConfig[]; activeCleared: boolean } {
  const servers = readStored().filter((it) => it.id !== id)
  saveServers(servers)
  const activeCleared = activeServerId() === id
  if (activeCleared) setActiveServerId(null)
  return { servers, activeCleared }
}

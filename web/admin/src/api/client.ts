// 请求层单通道:dev 经 Vite 代理,prod 同源反代;baseUrl 固定 /api/v1(真实实现前缀)。
import { ApiError, unwrap, type Envelope } from './envelope'

const BASE = '/api/v1'

let authToken: string | null = null

const storage = (): Pick<Storage, 'getItem' | 'setItem' | 'removeItem'> | null =>
  typeof localStorage === 'undefined' ? null : localStorage

/** 登录后注入 token(持久化 localStorage,与后端 JWT TTL 对齐)。 */
export function setAuthToken(token: string | null): void {
  authToken = token
  const s = storage()
  if (token === null) s?.removeItem('boss.token')
  else s?.setItem('boss.token', token)
}

/** 读取持久化 token(应用启动时恢复)。 */
export function getAuthToken(): string | null {
  if (authToken !== null) return authToken
  authToken = storage()?.getItem('boss.token') ?? null
  return authToken
}

export interface RequestOptions {
  method?: string
  body?: unknown
  query?: Record<string, string | number | undefined>
}

/** 发起请求并解 envelope,返回 data 负载。 */
export async function apiFetch<T = unknown>(path: string, opts: RequestOptions = {}): Promise<T | null> {
  const url = BASE + path + toQuery(opts.query)
  const headers: Record<string, string> = {}
  const token = getAuthToken()
  if (token) headers.Authorization = `Bearer ${token}`
  let body: string | undefined
  if (opts.body !== undefined) {
    headers['Content-Type'] = 'application/json'
    body = JSON.stringify(opts.body)
  }
  const res = await fetch(url, { method: opts.method ?? 'GET', headers, body })
  if (!res.ok) throw new ApiError(res.status, `网关错误(HTTP ${res.status})`)
  return unwrap<T>((await res.json()) as Envelope<T>)
}

function toQuery(q?: Record<string, string | number | undefined>): string {
  if (!q) return ''
  const parts = Object.entries(q)
    .filter(([, v]) => v !== undefined && v !== '')
    .map(([k, v]) => `${k}=${encodeURIComponent(String(v))}`)
  return parts.length ? `?${parts.join('&')}` : ''
}

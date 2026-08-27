// 请求层单通道:baseUrl 由服务端配置决定(默认环境=102 直连,后端已配置 CORS;
// 登录页/基础配置-服务端配置页可切换,localStorage 记忆)。禁止再引入 vite 代理通道。
import { ApiError, unwrap, type Envelope } from './envelope'
import { apiBaseUrl } from '../lib/serverConfig'

// 转导出:二进制下载等不走 apiFetch 的调用方需要同一基址。
export { apiBaseUrl }

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
  const url = apiBaseUrl() + path + toQuery(opts.query)
  const headers: Record<string, string> = {}
  const token = getAuthToken()
  if (token) headers.Authorization = `Bearer ${token}`
  let body: BodyInit | undefined
  if (opts.body !== undefined) {
    if (opts.body instanceof FormData) {
      body = opts.body // multipart 由浏览器自动补 boundary,勿手写 Content-Type
    } else if (typeof opts.body === 'string') {
      // 调用方已经 JSON.stringify 过:直接发,不要再 stringify 一遍(否则服务端收的是被双引号包裹的字符串)。
      headers['Content-Type'] = 'application/json'
      body = opts.body
    } else {
      headers['Content-Type'] = 'application/json'
      body = JSON.stringify(opts.body)
    }
  }
  const res = await fetch(url, { method: opts.method ?? 'GET', headers, body })
  if (!res.ok) {
    // 授权门禁拦截(无证书/证书失效):触发全局跳转到系统授权页,引导激活。
    // 后端返回 {"code":"LICENSE_REQUIRED",...} HTTP 403(见 internal/pkg/middleware/license.go)。
    if (res.status === 403) {
      try {
        const body = (await res.json()) as { code?: string }
        if (body.code === 'LICENSE_REQUIRED') {
          dispatchLicenseRequired()
          throw new LicenseRequiredError()
        }
      } catch (err) {
        if (err instanceof LicenseRequiredError) throw err
        // 403 但 body 非 LICENSE_REQUIRED(如 RBAC 无权限):走通用错误。
      }
    }
    throw new ApiError(res.status, `网关错误(HTTP ${res.status})`)
  }
  return unwrap<T>((await res.json()) as Envelope<T>)
}

/** LICENSE_REQUIRED 专用错误:业务层可据此展示"系统未授权"而非普通报错。 */
export class LicenseRequiredError extends Error {
  constructor() {
    super('license required')
    this.name = 'LicenseRequiredError'
  }
}

// 授权跳转事件:模块级订阅(不依赖 window,node 环境可测;App 层订阅后 Navigate)。
type LicenseHandler = () => void
const licenseHandlers = new Set<LicenseHandler>()

/** 触发全局授权跳转事件(App 层监听后 Navigate 到 /base/license)。 */
export function dispatchLicenseRequired(): void {
  for (const h of [...licenseHandlers]) h()
}

/** 订阅授权跳转事件;返回取消函数。 */
export function onLicenseRequired(handler: LicenseHandler): () => void {
  licenseHandlers.add(handler)
  return () => {
    licenseHandlers.delete(handler)
  }
}

function toQuery(q?: Record<string, string | number | undefined>): string {
  if (!q) return ''
  const parts = Object.entries(q)
    .filter(([, v]) => v !== undefined && v !== '')
    .map(([k, v]) => `${k}=${encodeURIComponent(String(v))}`)
  return parts.length ? `?${parts.join('&')}` : ''
}

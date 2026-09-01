// 响应 envelope 解包(契约:docs/plan/admin-a0-plan.md T2)。
// Go 实现 HTTP 恒 200;code===0 取 data,code===401 未授权,其余业务错误。

export const CODE_OK = 0
export const CODE_UNAUTHORIZED = 401

export interface Envelope<T = unknown> {
  code: number
  msg: string
  data?: T | null
}

/** 业务错误:携带 code 与 msg,unauthorized 标记触发登出;reason 为 data.reason 透传(如风控拦截指引)。 */
export class ApiError extends Error {
  readonly code: number
  readonly reason: string
  constructor(code: number, msg: string, reason = '') {
    super(reason ? msg + '：' + reason : msg)
    this.name = 'ApiError'
    this.code = code
    this.reason = reason
  }
  get unauthorized(): boolean {
    return this.code === CODE_UNAUTHORIZED
  }
}

/** 解包:code=0 → data;否则抛 ApiError(data.reason 拼进 message,页面零改动可读)。 */
export function unwrap<T>(env: Envelope<T>): T | null {
  if (env.code === CODE_OK) return env.data ?? null
  const data = env.data as { reason?: unknown } | null | undefined
  const reason = typeof data?.reason === 'string' ? data.reason : ''
  throw new ApiError(env.code, env.msg || `业务错误(${env.code})`, reason)
}

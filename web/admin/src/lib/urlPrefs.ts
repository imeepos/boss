// URL 一次性偏好覆盖:?theme=dark&lang=zh-CN&token=<jwt> → 写 localStorage 后 replaceState 抹除参数。
// 权威存储仍是 localStorage;URL 只承担"可分享的一次性覆盖",不随路由跳转传播。
interface PrefSpec {
  storage: string
  valid?: readonly string[]
  /** 仅 dev 构建应用(prod 仍抹除参数防泄漏,但不写入)。 */
  devOnly?: boolean
  allow?: (v: string) => boolean
}

const URL_PREFS: Record<string, PrefSpec> = {
  theme: { storage: 'boss.theme', valid: ['light', 'dark'] },
  lang: { storage: 'boss.locale', valid: ['zh-CN', 'en-US', 'ms-MY'] },
  // dev 免登录:真实 /auth/login 换来的 JWT(见 scripts/dev-token.mjs),禁止假数据
  token: { storage: 'boss.token', devOnly: true, allow: (v) => v.length > 10 },
}

/** 应用启动时调用一次(须在 Provider 读 localStorage 之前)。 */
export function applyUrlPrefs(): void {
  if (typeof window === 'undefined') return
  const url = new URL(window.location.href)
  let dirty = false
  for (const [param, spec] of Object.entries(URL_PREFS)) {
    const v = url.searchParams.get(param)
    if (v === null) continue
    const applicable = (!spec.devOnly || import.meta.env.DEV) && isAcceptable(spec, v)
    if (applicable) {
      try { localStorage.setItem(spec.storage, v) } catch { /* quota */ }
    }
    url.searchParams.delete(param)
    dirty = true
  }
  if (dirty) window.history.replaceState(null, '', url)
}

function isAcceptable(spec: PrefSpec, v: string): boolean {
  if (spec.valid) return spec.valid.includes(v)
  if (spec.allow) return spec.allow(v)
  return true
}

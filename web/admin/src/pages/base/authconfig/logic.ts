// 认证配置页纯逻辑:字段 key 常量、草稿初始化、提交 payload 组装(与后端 authconfig_fields.go 对齐)。

export interface AuthFieldState {
  value: string
  hasValue: boolean
}

export type AuthFields = Record<string, AuthFieldState>

export const CN_KEYS = [
  'auth.cn.enabled', 'auth.cn.appKey', 'auth.cn.appSecret',
  'auth.cn.packageName', 'auth.cn.preloadTimeoutMs',
] as const

export const MY_KEYS = [
  'auth.my.enabled', 'auth.my.provider', 'auth.my.smsProvider',
  'auth.my.apiKey', 'auth.my.countryCode', 'auth.my.smsSign',
] as const

export const FB_KEYS = [
  'auth.fallback.smsOnFail', 'auth.fallback.billingAlert', 'auth.fallback.autoRegister',
  'auth.compliance.privacyVersion', 'auth.compliance.agreementUrl',
] as const

export const SECRET_KEYS = new Set(['auth.cn.appSecret', 'auth.my.apiKey'])

export const FIELD_DEFAULTS: Record<string, string> = {
  'auth.cn.enabled': 'false',
  'auth.cn.preloadTimeoutMs': '5000',
  'auth.my.enabled': 'false',
  'auth.my.provider': 'none',
  'auth.my.smsProvider': 'engagelab',
  'auth.my.countryCode': '+60',
  'auth.fallback.smsOnFail': 'true',
  'auth.fallback.billingAlert': 'true',
  'auth.fallback.autoRegister': 'false',
}

/** 草稿初始化:非 secret 字段取 value 或默认;secret 恒空串(掩码,未改不提交)。 */
export function initDraft(fields: AuthFields): Record<string, string> {
  const draft: Record<string, string> = {}
  for (const key of [...CN_KEYS, ...MY_KEYS, ...FB_KEYS]) {
    const f = fields[key]
    if (SECRET_KEYS.has(key)) {
      draft[key] = ''
      continue
    }
    draft[key] = f?.value ?? FIELD_DEFAULTS[key] ?? ''
  }
  return draft
}

/** 提交 payload:只带相对已加载状态变化的 key;secret 空串不提交(=不修改)。 */
export function payloadFor(
  keys: readonly string[],
  draft: Record<string, string>,
  loaded: Record<string, string>,
): Record<string, string> {
  const values: Record<string, string> = {}
  for (const key of keys) {
    const cur = draft[key] ?? ''
    if (SECRET_KEYS.has(key)) {
      if (cur !== '') values[key] = cur
      continue
    }
    if (cur !== (loaded[key] ?? '')) values[key] = cur
  }
  return values
}

/** 预取号超时校验:2000–10000 整数,合法返回空串。 */
export function timeoutError(v: string): boolean {
  const n = Number(v)
  return v === '' || !Number.isInteger(n) || n < 2000 || n > 10000
}

// Stripe 支付配置页纯逻辑:字段 key 常量、草稿初始化、提交 payload 组装(与后端 stripeconfig_fields.go 对齐)。

export interface StripeFieldState {
  value: string
  hasValue: boolean
}

export type StripeFields = Record<string, StripeFieldState>

export const CH_KEYS = [
  'stripe.enabled', 'stripe.apiKey', 'stripe.publishableKey', 'stripe.currency', 'stripe.apiBaseUrl',
] as const

export const WH_KEYS = ['stripe.webhookSecret'] as const

export const STRIPE_SECRET_KEYS = new Set(['stripe.apiKey', 'stripe.webhookSecret'])

const FIELD_DEFAULTS: Record<string, string> = { 'stripe.enabled': 'true', 'stripe.currency': 'php' }

/** 草稿初始化:非 secret 取 value 或默认;secret 恒空串(掩码,未改不提交)。 */
export function draftFrom(fields: StripeFields): Record<string, string> {
  const draft: Record<string, string> = {}
  for (const key of [...CH_KEYS, ...WH_KEYS]) {
    if (STRIPE_SECRET_KEYS.has(key)) {
      draft[key] = ''
      continue
    }
    draft[key] = fields[key]?.value ?? FIELD_DEFAULTS[key] ?? ''
  }
  return draft
}

/** 提交 payload:只带相对已加载状态变化的 key;secret 空串不提交(=不修改)。 */
export function stripePayloadFor(
  keys: readonly string[],
  draft: Record<string, string>,
  loaded: Record<string, string>,
): Record<string, string> {
  const values: Record<string, string> = {}
  for (const key of keys) {
    const cur = draft[key] ?? ''
    if (STRIPE_SECRET_KEYS.has(key)) {
      if (cur !== '') values[key] = cur
      continue
    }
    if (cur !== (loaded[key] ?? '')) values[key] = cur
  }
  return values
}
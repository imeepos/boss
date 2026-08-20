// 短信配置页纯逻辑:字段 key 常量、草稿初始化、提交 payload 组装(与后端 smsconfig_fields.go 对齐)。

export interface SmsFieldState {
  value: string
  hasValue: boolean
}

export type SmsFields = Record<string, SmsFieldState>

export const CH_KEYS = [
  'sms.enabled', 'sms.provider', 'sms.accessKeyId', 'sms.accessKeySecret', 'sms.from',
] as const

export const TP_KEYS = ['sms.template.cn', 'sms.template.my'] as const

export const SMS_SECRET_KEYS = new Set(['sms.accessKeySecret'])

const FIELD_DEFAULTS: Record<string, string> = { 'sms.enabled': 'true', 'sms.provider': 'aliyun_intl' }

/** 草稿初始化:非 secret 取 value 或默认;secret 恒空串(掩码,未改不提交)。 */
export function draftFrom(fields: SmsFields): Record<string, string> {
  const draft: Record<string, string> = {}
  for (const key of [...CH_KEYS, ...TP_KEYS]) {
    if (SMS_SECRET_KEYS.has(key)) {
      draft[key] = ''
      continue
    }
    draft[key] = fields[key]?.value ?? FIELD_DEFAULTS[key] ?? ''
  }
  return draft
}

/** 提交 payload:只带相对已加载状态变化的 key;secret 空串不提交(=不修改)。 */
export function smsPayloadFor(
  keys: readonly string[],
  draft: Record<string, string>,
  loaded: Record<string, string>,
): Record<string, string> {
  const values: Record<string, string> = {}
  for (const key of keys) {
    const cur = draft[key] ?? ''
    if (SMS_SECRET_KEYS.has(key)) {
      if (cur !== '') values[key] = cur
      continue
    }
    if (cur !== (loaded[key] ?? '')) values[key] = cur
  }
  return values
}

/** 测试手机号校验:+8613..(中国)/+6012..(马来西亚)或裸 11 位中国号;合法返回空串。 */
export function phoneError(v: string): string {
  const d = v.replace(/[^0-9]/g, '')
  if (v.startsWith('+')) {
    if (/^\+861\d{10}$/.test(v) || /^\+601\d{8,9}$/.test(v)) return ''
    return 'bad-e164'
  }
  if (/^1\d{10}$/.test(d)) return ''
  return 'bad-phone'
}

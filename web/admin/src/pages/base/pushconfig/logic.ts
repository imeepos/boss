// 推送配置页纯逻辑:字段 key 常量、草稿初始化、提交 payload 组装(与后端 pushconfig_fields.go 对齐)。

export interface PushFieldState {
  value: string
  hasValue: boolean
}

export type PushFields = Record<string, PushFieldState>

export const CH_KEYS = [
  'push.enabled', 'push.provider', 'push.jpush.appKey', 'push.jpush.masterSecret',
  'push.jpush.apiUrl', 'push.jpush.apnsProduction', 'push.jpush.liveTime',
] as const

export const PUSH_SECRET_KEYS = new Set(['push.jpush.masterSecret'])

const FIELD_DEFAULTS: Record<string, string> = {
  'push.enabled': 'true',
  'push.provider': 'jpush',
  'push.jpush.apiUrl': 'https://bjapi.push.jiguang.cn/v3',
  'push.jpush.apnsProduction': 'true',
  'push.jpush.liveTime': '86400',
}

/** 草稿初始化:非 secret 取 value 或默认;secret 恒空串(掩码,未改不提交)。 */
export function draftFrom(fields: PushFields): Record<string, string> {
  const draft: Record<string, string> = {}
  for (const key of CH_KEYS) {
    if (PUSH_SECRET_KEYS.has(key)) {
      draft[key] = ''
      continue
    }
    draft[key] = fields[key]?.value ?? FIELD_DEFAULTS[key] ?? ''
  }
  return draft
}

/** 提交 payload:只带相对已加载状态变化的 key;secret 空串不提交(=不修改)。 */
export function pushPayloadFor(
  draft: Record<string, string>,
  loaded: Record<string, string>,
): Record<string, string> {
  const values: Record<string, string> = {}
  for (const key of CH_KEYS) {
    const cur = draft[key] ?? ''
    if (PUSH_SECRET_KEYS.has(key)) {
      if (cur !== '') values[key] = cur
      continue
    }
    if (cur !== (loaded[key] ?? '')) values[key] = cur
  }
  return values
}

/** 测试目标校验:registration_id(数字字母,>=10 位)或 alias 前缀 user:/worker:;合法返回空串。 */
export function targetError(v: string, kind: 'registration_id' | 'alias'): string {
  if (kind === 'alias') return /^(user|worker):\d+$/.test(v) ? '' : 'bad-alias'
  return /^[0-9a-zA-Z]{10,}$/.test(v) ? '' : 'bad-regid'
}

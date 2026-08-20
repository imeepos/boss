// 实名核验配置页纯逻辑:字段 key 常量、草稿初始化、提交 payload 组装(与后端 realidconfig_fields.go 对齐)。

export interface RealIDFieldState {
  value: string
  hasValue: boolean
}

export type RealIDFields = Record<string, RealIDFieldState>

export const CH_KEYS = [
  'realid.enabled', 'realid.provider', 'realid.accessKeyId', 'realid.accessKeySecret', 'realid.endpoint',
] as const

export const SECRET_KEYS = new Set(['realid.accessKeySecret'])

const FIELD_DEFAULTS: Record<string, string> = {
  'realid.enabled': 'true',
  'realid.provider': 'aliyun_cloudauth',
}

/** 草稿初始化:非 secret 取 value 或默认;secret 恒空串(掩码,未改不提交)。 */
export function draftFrom(fields: RealIDFields): Record<string, string> {
  const draft: Record<string, string> = {}
  for (const key of CH_KEYS) {
    if (SECRET_KEYS.has(key)) {
      draft[key] = ''
      continue
    }
    draft[key] = fields[key]?.value ?? FIELD_DEFAULTS[key] ?? ''
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

/** 试核二要素校验:中文姓名 2-30 字,18 位身份证(尾号允许 X);合法返回空串。 */
export function idPairError(name: string, idNo: string): string {
  if (!name && !idNo) return ''
  if (!/^[\u4e00-\u9fa5·]{2,30}$/.test(name)) return 'bad-name'
  if (!/^\d{17}[\dXx]$/.test(idNo)) return 'bad-idno'
  return ''
}

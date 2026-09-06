// 危险操作确认与事件变更展示的纯逻辑(P2-T4):抽出便于单测,组件只做渲染。

export interface ChangedPair {
  key: string
  from: string
  to: string
}

// 解绑确认放行条件:原因必填(纯空格不算)。
export function canConfirmUnbind(reason: string): boolean {
  return reason.trim().length > 0
}

// 报废确认放行条件:原因必填 + 资产编码精确匹配(防误操作,trim 后比较)。
export function canConfirmScrap(reason: string, codeInput: string, assetCode: string): boolean {
  return reason.trim().length > 0 && codeInput.trim() === assetCode
}

// changedPairs 把写侧 changed JSONB(键 -> [旧值, 新值])转为可渲染行:
// 只渲染实际变化的键,不 dump 全量 JSON;非数组值视为仅新值。
export function changedPairs(changed: Record<string, unknown> | null | undefined): ChangedPair[] {
  if (!changed) return []
  return Object.entries(changed).map(([key, v]) => {
    if (Array.isArray(v) && v.length === 2) {
      return { key, from: stringifyVal(v[0]), to: stringifyVal(v[1]) }
    }
    return { key, from: '', to: stringifyVal(v) }
  })
}

function stringifyVal(v: unknown): string {
  if (v === null || v === undefined || v === '') return '-'
  return typeof v === 'string' ? v : JSON.stringify(v)
}

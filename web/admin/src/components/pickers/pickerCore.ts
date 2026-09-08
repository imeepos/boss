// 选择器公共基座核心逻辑:双数据源合并、单/多选集合、已选回显、分页查询组装、键盘导航、钉选回显与清空口径。
// 纯函数零依赖(仅类型),供 SimplePicker / DialogPicker / ResourcePicker 复用并配 vitest 单测。
import type { DropdownOption } from '../Dropdown'

/**
 * 合并选项组:按出现顺序拼接,按 value 去重(先出现者优先)。
 * 惯例传参顺序:空值选项 → 钉选选项 → 静态/服务端检索结果。
 */
export function mergeOptions(...groups: Array<DropdownOption[] | undefined>): DropdownOption[] {
  const seen = new Set<string>()
  const out: DropdownOption[] = []
  for (const group of groups) {
    for (const o of group ?? []) {
      if (seen.has(o.value)) continue
      seen.add(o.value)
      out.push(o)
    }
  }
  return out
}

/** 多选集合:勾选追加(已存在则原样返回),取消移除,保持选择顺序。 */
export function togglePickKey(selected: string[], key: string, checked: boolean): string[] {
  if (checked) return selected.includes(key) ? selected : [...selected, key]
  return selected.filter((it) => it !== key)
}

/** 单选集合:重置为当前项;空串视为清空。 */
export function pickSingleKey(key: string): string[] {
  return key === '' ? [] : [key]
}

/** 已选回显项:key + 展示文案。 */
export interface SelectionChip {
  key: string
  label: string
}

/**
 * 把选中 key 序列映射为回显 chips;keyToLabel 索引缺失的项剔除
 * (跨页分页场景下,已选行未必都在当前页数据里,索引由组件增量维护)。
 */
export function toSelectionChips(selected: string[], keyToLabel: Map<string, string>): SelectionChip[] {
  const out: SelectionChip[] = []
  for (const key of selected) {
    const label = keyToLabel.get(key)
    if (label !== undefined) out.push({ key, label })
  }
  return out
}

export interface PickerQueryInput {
  /** 关键字(原样传入,组装时 trim,空串剔除)。 */
  keyword: string
  /** 附加筛选项:value 为空串表示"全部",剔除不出参。 */
  filters: Record<string, string>
  /** 页码,1 起;非法值钳位为 1。 */
  page: number
  pageSize: number
}

/** 组装服务端分页查询参数:limit/offset + 非空 keyword + 非空筛选项。 */
export function buildPickerQuery(q: PickerQueryInput): Record<string, string | number> {
  const page = Math.max(1, Math.floor(q.page))
  const out: Record<string, string | number> = {
    limit: q.pageSize,
    offset: (page - 1) * q.pageSize,
  }
  const kw = q.keyword.trim()
  if (kw) out.keyword = kw
  for (const [k, v] of Object.entries(q.filters)) {
    const t = v.trim()
    if (t) out[k] = t
  }
  return out
}

/** 统一防抖毫秒:SimplePicker 服务端检索与 DialogPicker 关键字检索共用,全体系同节奏。 */
export const PICKER_DEBOUNCE_MS = 300

/**
 * 键盘上下移动活动项:跳过禁用项并循环回绕;未初始化(-1)向下落首项、向上落末项;
 * 空列表返回 -1;全部禁用返回原值(保持不动)。
 */
export function moveActive(count: number, current: number, delta: 1 | -1, isDisabled?: (index: number) => boolean): number {
  if (count <= 0) return -1
  let idx = current
  for (let n = 0; n < count; n += 1) {
    idx = idx < 0 ? (delta === 1 ? 0 : count - 1) : (idx + delta + count) % count
    if (!isDisabled?.(idx)) return idx
  }
  return current < 0 ? -1 : current
}

/** 已选值不在选项集时追加合成选项(value 兼作 label),保证触发器回显不丢失;空值不钉。 */
export function withPinnedValue(options: DropdownOption[], value: string): DropdownOption[] {
  if (value === '' || options.some((o) => o.value === value)) return options
  return [...options, { value, label: value }]
}

/** 清空按钮可见性口径:显式开启 clearable 且未禁用且有值。 */
export function canClearValue(clearable: boolean | undefined, disabled: boolean | undefined, value: string): boolean {
  return !!clearable && !disabled && value !== ''
}

/** 服务端检索状态机状态:结果集 + loading/error + 已发出的最新请求序号(防竞态)。 */
export interface PickerSearchState<T> {
  items: T[]
  loading: boolean
  error: boolean
  reqSeq: number
}

export function initialPickerSearchState<T>(): PickerSearchState<T> {
  return { items: [], loading: false, error: false, reqSeq: 0 }
}

export type PickerSearchAction<T> =
  | { type: 'request'; seq: number }
  | { type: 'ok'; seq: number; items: T[] }
  | { type: 'fail'; seq: number }

/** 纯状态机:仅接受最新 seq 的响应,过期响应原样忽略;fail 清空结果并置错误态(可重试)。 */
export function pickerSearchReducer<T>(state: PickerSearchState<T>, action: PickerSearchAction<T>): PickerSearchState<T> {
  switch (action.type) {
    case 'request':
      return { ...state, loading: true, error: false, reqSeq: action.seq }
    case 'ok':
      if (action.seq !== state.reqSeq) return state
      return { ...state, items: action.items, loading: false, error: false }
    case 'fail':
      if (action.seq !== state.reqSeq) return state
      return { ...state, items: [], loading: false, error: true }
    default:
      return state
  }
}

/**
 * 有效选中值解析(W1 裁定防御):精确 value 命中优先;存量调用把显示文案当 value 传时,
 * 按 label 同值兜底(effectiveValue 回到真实 value);双 miss 原样返回,交由 withPinnedValue
 * 合成钉选回显。hit.value !== value 即 label 兜底命中,调用方应 console.warn 留痕。
 */
export function resolveOptionMatch(options: DropdownOption[], value: string): { effectiveValue: string; hit?: DropdownOption } {
  if (value === '') return { effectiveValue: '' }
  const exact = options.find((o) => o.value === value)
  if (exact) return { effectiveValue: value, hit: exact }
  const byLabel = options.find((o) => o.label === value)
  if (byLabel) return { effectiveValue: byLabel.value, hit: byLabel }
  return { effectiveValue: value }
}

/**
 * 已选人类可读回显(W0-R3):检索结果 label 增量缓存按 value 命中;
 * 空值或未命中返回 undefined(组件再走详情接口兜底),避免触发器跌回裸内部编号。
 */
export function echoPinFromCache(labels: Map<string, string>, value: string): DropdownOption | undefined {
  if (value === '') return undefined
  const label = labels.get(value)
  return label === undefined ? undefined : { value, label }
}

/** 增量记录检索结果 label(按 value 覆盖,空值不入缓存),供 echoPinFromCache 与详情兜底复用。 */
export function rememberOptionLabels(labels: Map<string, string>, options: DropdownOption[]): void {
  for (const o of options) {
    if (o.value === '' || !o.label) continue
    labels.set(o.value, o.label)
  }
}

/** 实体数组转选中 key 序列:保持传入顺序,去重并剔除空 key(DialogPicker 重开预选用)。 */
export function pickKeysFromItems<T>(items: T[], rowKey: (item: T) => string): string[] {
  const out: string[] = []
  for (const item of items) {
    const key = rowKey(item)
    if (key === '' || out.includes(key)) continue
    out.push(key)
  }
  return out
}

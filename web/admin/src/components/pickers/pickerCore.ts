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

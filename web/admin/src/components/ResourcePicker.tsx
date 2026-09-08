// 兼容层(W0 定位):既有约 20 处直用页面的 props 契约原样保留(load/search + toOption),
// 内部统一转发 SimplePicker —— 服务端检索的防抖/loading/空态/失败重试/键盘可达/回显钉选
// 由基座统一供给,直用页面零改动受益。新页面勿再直用本组件:小数据量 SimplePicker,
// 大数据量 DialogPicker(见 docs/admin/picker-guide.md)。
// W0-R4 加法:clearable 默认开启(存量页面自动获得一键清空),clearLabel 缺省取
// pages.pickers.common.clear 三语词条;传 clearable={false} 可退出。
import { useEffect, useRef, useState } from 'react'
import { useT } from '../i18n'
import { SimplePicker } from './pickers/SimplePicker'
import type { DropdownOption } from './Dropdown'

export interface ResourcePickerProps<T> {
  value: string
  onChange: (value: string) => void
  /** 一次性全量加载(与 search 二选一,兼容既有用法)。 */
  load?: () => Promise<T[] | null>
  /** 服务端关键字检索(load 优先级低于 search);空串 = 首屏全量。 */
  search?: (keyword: string) => Promise<T[] | null>
  /** 服务端检索防抖毫秒数,默认 300。 */
  debounceMs?: number
  toOption: (item: T) => DropdownOption
  ariaLabel: string
  /** 提供时追加 value='' 的空选项(可选空值)。 */
  emptyLabel?: string
  /** 钉选选项:已选值不在检索结果内(如默认带出的档案地址)时保证回显;按 value 去重。 */
  pinnedOptions?: DropdownOption[]
  searchPlaceholder?: string
  /** 加载失败文案;缺失时兜底为 ariaLabel,仅就地展示不阻止继续填表。 */
  errorText?: string
  disabled?: boolean
  minWidth?: number
  /** 值为空时触发器占位文案;缺省回退 ariaLabel。 */
  placeholder?: string
  /** 清空按钮开关,默认开启(值非空且未禁用时展示);传 false 退出。 */
  clearable?: boolean
  /** 清空按钮 aria 文案;缺省取 pages.pickers.common.clear。 */
  clearLabel?: string
}

/** load 模式一次性全量拉取:loading/error/重试由本层持有,拉完交静态源本地过滤。 */
function useLoadOnce<T>(load: (() => Promise<T[] | null>) | undefined, toOption: (item: T) => DropdownOption) {
  const [items, setItems] = useState<DropdownOption[]>([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState(false)
  const [tick, setTick] = useState(0)
  const loadRef = useRef(load)
  loadRef.current = load
  const mapRef = useRef(toOption)
  mapRef.current = toOption

  useEffect(() => {
    const req = loadRef.current
    if (!req) return undefined
    let alive = true
    setLoading(true)
    setError(false)
    req()
      .then((data) => { if (alive) { setItems((data ?? []).map(mapRef.current)); setLoading(false) } })
      .catch(() => { if (alive) { setItems([]); setError(true); setLoading(false) } })
    return () => { alive = false }
  }, [tick])

  return { items, loading, error, retry: () => setTick((n) => n + 1) }
}

export function ResourcePicker<T>({ value, onChange, load, search, debounceMs, toOption, ariaLabel, emptyLabel, pinnedOptions, searchPlaceholder, errorText, disabled, minWidth, placeholder, clearable = true, clearLabel }: ResourcePickerProps<T>) {
  const commonClear = useT().pages.pickers.common.clear
  const loaded = useLoadOnce(load, toOption)
  if (load) {
    return (
      <SimplePicker
        value={value}
        onChange={onChange}
        options={loaded.items}
        loading={loaded.loading}
        error={loaded.error}
        onRetry={loaded.retry}
        ariaLabel={ariaLabel}
        placeholder={placeholder}
        emptyLabel={emptyLabel}
        pinnedOptions={pinnedOptions}
        searchPlaceholder={searchPlaceholder}
        errorText={errorText}
        clearable={clearable}
        clearLabel={clearLabel ?? commonClear}
        disabled={disabled}
        minWidth={minWidth}
      />
    )
  }
  return (
    <SimplePicker
      value={value}
      onChange={onChange}
      search={search ? (kw) => search(kw).then((items) => (items ?? []).map(toOption)) : undefined}
      debounceMs={debounceMs}
      ariaLabel={ariaLabel}
      placeholder={placeholder}
      emptyLabel={emptyLabel}
      pinnedOptions={pinnedOptions}
      searchPlaceholder={searchPlaceholder}
      errorText={errorText}
      clearable={clearable}
      clearLabel={clearLabel ?? commonClear}
      disabled={disabled}
      minWidth={minWidth}
    />
  )
}

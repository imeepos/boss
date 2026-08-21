// 通用资源选择器:异步加载列表项并复用 Dropdown 视觉;支持可选空值、
// label/value 映射与关键字搜索;加载失败仅就地提示,不阻塞表单提交。
// search 模式:关键字经 debounce 后请求服务端(keyword 检索),关闭本地过滤。
import { useEffect, useRef, useState } from 'react'
import { Dropdown, type DropdownOption } from './Dropdown'

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
  searchPlaceholder?: string
  /** 加载失败文案;缺失时兜底为 ariaLabel,仅就地展示不阻止继续填表。 */
  errorText?: string
  disabled?: boolean
  minWidth?: number
}

export function ResourcePicker<T>({ value, onChange, load, search, debounceMs = 300, toOption, ariaLabel, emptyLabel, searchPlaceholder, errorText, disabled, minWidth = 220 }: ResourcePickerProps<T>) {
  const [items, setItems] = useState<T[]>([])
  const [loadError, setLoadError] = useState('')
  const [keyword, setKeyword] = useState('')
  const seqRef = useRef(0)

  const run = (kw: string) => {
    if (!search) return
    const seq = ++seqRef.current
    search(kw)
      .then((data) => { if (seq === seqRef.current) setItems(data ?? []) })
      .catch(() => { if (seq === seqRef.current) setLoadError(errorText ?? ariaLabel) })
  }

  // 首屏:search 模式取全量(空关键字),load 模式一次性加载。
  useEffect(() => {
    let alive = true
    const seq = ++seqRef.current
    const req = search ? search('') : load?.()
    req
      ?.then((data) => { if (alive && seq === seqRef.current) setItems(data ?? []) })
      ?.catch(() => { if (alive && seq === seqRef.current) setLoadError(errorText ?? ariaLabel) })
    return () => { alive = false }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  // search 模式:关键字防抖后请求服务端,seq 保证只采最新响应。
  useEffect(() => {
    if (!search) return undefined
    const timer = setTimeout(() => run(keyword), debounceMs)
    return () => clearTimeout(timer)
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [keyword])

  const options: DropdownOption[] = [
    ...(emptyLabel ? [{ value: '', label: emptyLabel }] : []),
    ...items.map(toOption),
  ]

  return (
    <div>
      <Dropdown
        value={value}
        options={options}
        onChange={onChange}
        ariaLabel={ariaLabel}
        disabled={disabled}
        searchable
        remote={!!search}
        onKeywordChange={search ? setKeyword : undefined}
        searchPlaceholder={searchPlaceholder}
        triggerStyle={{ minWidth }}
      />
      {loadError && <div className="mt-1 text-[11px] text-[var(--color-danger)]">{loadError}</div>}
    </div>
  )
}

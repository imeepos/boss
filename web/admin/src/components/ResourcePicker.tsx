// 通用资源选择器:异步加载列表项并复用 Dropdown 视觉;支持可选空值、
// label/value 映射与关键字搜索;加载失败仅就地提示,不阻塞表单提交。
import { useEffect, useState } from 'react'
import { Dropdown, type DropdownOption } from './Dropdown'

export interface ResourcePickerProps<T> {
  value: string
  onChange: (value: string) => void
  load: () => Promise<T[] | null>
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

export function ResourcePicker<T>({ value, onChange, load, toOption, ariaLabel, emptyLabel, searchPlaceholder, errorText, disabled, minWidth = 220 }: ResourcePickerProps<T>) {
  const [items, setItems] = useState<T[]>([])
  const [loadError, setLoadError] = useState('')

  useEffect(() => {
    let alive = true
    load()
      .then((data) => { if (alive) setItems(data ?? []) })
      .catch(() => { if (alive) setLoadError(errorText ?? ariaLabel) })
    return () => { alive = false }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

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
        searchPlaceholder={searchPlaceholder}
        triggerStyle={{ minWidth }}
      />
      {loadError && <div className="mt-1 text-[11px] text-[var(--color-danger)]">{loadError}</div>}
    </div>
  )
}
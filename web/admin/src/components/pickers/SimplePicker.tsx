// 小数据量选择器基座:双数据源(静态 options 数组 / 服务端关键字检索函数),全站表单字段下拉统一入口。
// 静态源走 Dropdown 本地过滤;服务端源经 ResourcePicker 防抖检索;支持清空/禁用/占位/i18n aria/键盘可达。
// 视觉复用 Dropdown 与 shell-* 令牌;文案(含 aria)一律由调用方 i18n 传入,组件内不硬编码。
import { useMemo } from 'react'
import { Dropdown, type DropdownOption } from '../Dropdown'
import { ResourcePicker } from '../ResourcePicker'
import { mergeOptions } from './pickerCore'

export interface SimplePickerProps {
  value: string
  onChange: (value: string) => void
  /** 静态数据源:全量选项数组(本地关键字过滤);与 search 二选一,search 优先。 */
  options?: DropdownOption[]
  /** 服务端数据源:关键字检索返回选项列表。 */
  search?: (keyword: string) => Promise<DropdownOption[] | null>
  /** 服务端检索防抖毫秒数,默认 300。 */
  debounceMs?: number
  /** 触发器 aria-label(调用方 i18n)。 */
  ariaLabel: string
  /** 值为空时触发器占位文案;缺省回退 ariaLabel。 */
  placeholder?: string
  /** 展示清空按钮(值非空且未禁用时)。 */
  clearable?: boolean
  /** 清空按钮 aria 文案(clearable 时由调用方 i18n 传入)。 */
  clearLabel?: string
  disabled?: boolean
  /** 浮层过滤输入占位(服务端模式亦复用为检索占位)。 */
  searchPlaceholder?: string
  /** 服务端首屏/检索失败的就地提示文案。 */
  errorText?: string
  /** 追加 value='' 的空选项(可选空值场景)。 */
  emptyLabel?: string
  /** 钉选选项:已选值不在检索结果内时保证回显;按 value 去重。 */
  pinnedOptions?: DropdownOption[]
  /** 触发器最小宽度,默认 220。 */
  minWidth?: number
}

const CLEAR_BTN =
  'inline-flex h-6 w-6 shrink-0 cursor-pointer items-center justify-center rounded-sm border-none bg-none text-[var(--shell-group-title)] hover:bg-[var(--shell-menu-hover-bg)] hover:text-[var(--shell-content-text)]'

export function SimplePicker(props: SimplePickerProps) {
  const {
    value, onChange, options, search, debounceMs, ariaLabel, placeholder,
    clearable, clearLabel, disabled, searchPlaceholder, errorText,
    emptyLabel, pinnedOptions, minWidth,
  } = props
  // 静态源选项编排:空值选项 + 钉选项 + 静态全量(关键字过滤交给 Dropdown 本地过滤)。
  const staticOptions = useMemo(
    () => mergeOptions(
      emptyLabel ? [{ value: '', label: emptyLabel }] : undefined,
      pinnedOptions,
      options,
    ),
    [emptyLabel, pinnedOptions, options],
  )
  const canClear = !!clearable && !disabled && value !== ''
  return (
    <div className="inline-flex items-center gap-1" data-testid="simple-picker">
      {search ? (
        <ResourcePicker
          value={value}
          onChange={onChange}
          search={search}
          debounceMs={debounceMs}
          toOption={(o) => o}
          ariaLabel={ariaLabel}
          placeholder={placeholder}
          emptyLabel={emptyLabel}
          pinnedOptions={pinnedOptions}
          searchPlaceholder={searchPlaceholder}
          errorText={errorText}
          disabled={disabled}
          minWidth={minWidth}
        />
      ) : (
        <Dropdown
          value={value}
          options={staticOptions}
          onChange={onChange}
          ariaLabel={ariaLabel}
          placeholder={placeholder}
          disabled={disabled}
          searchable
          searchPlaceholder={searchPlaceholder}
          triggerStyle={{ minWidth }}
        />
      )}
      {canClear && (
        <button
          type="button"
          className={CLEAR_BTN}
          aria-label={clearLabel ?? ariaLabel}
          title={clearLabel ?? ariaLabel}
          onClick={() => onChange('')}
        >
          <svg viewBox="0 0 24 24" width="14" height="14" fill="none" stroke="currentColor"
            strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round" aria-hidden>
            <path d="M18 6L6 18M6 6l12 12" />
          </svg>
        </button>
      )}
    </div>
  )
}

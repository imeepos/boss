// 小数据量选择器基座:双数据源(静态 options 数组 / 服务端 search 函数),全站表单字段下拉统一入口。
// 静态源走 Dropdown 本地过滤;服务端源经 useServerPickerSearch 防抖检索(seq 防竞态)。
// loading/空态/失败重试/清空/键盘可达由基座统一供给(W0 契约);结构性文案缺省取
// pages.pickers.common 三语词条,调用方可用 loadingText/emptyText/retryText 覆盖,组件内不硬编码。
import { useMemo } from 'react'
import { useT } from '../../i18n'
import { Dropdown, type DropdownOption } from '../Dropdown'
import { canClearValue, mergeOptions } from './pickerCore'
import { useServerPickerSearch } from './useServerPickerSearch'

export interface SimplePickerProps {
  value: string
  onChange: (value: string) => void
  /** 静态数据源:全量选项数组(本地关键字过滤);与 search 二选一,search 优先。 */
  options?: DropdownOption[]
  /** 服务端数据源:关键字检索返回选项列表(内部防抖,默认 300ms)。 */
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
  /** 浮层过滤/检索输入占位。 */
  searchPlaceholder?: string
  /** 检索输入 aria 文案;缺省回退 searchPlaceholder。 */
  searchAriaLabel?: string
  /** 服务端首屏/检索失败的就地提示文案;缺省回退 ariaLabel。 */
  errorText?: string
  /** 追加 value='' 的空选项(可选空值场景)。 */
  emptyLabel?: string
  /** 调用方显式钉选选项(按 value 去重);已选值漏项另由基座 withPinnedValue 自动兜底。 */
  pinnedOptions?: DropdownOption[]
  /** 触发器最小宽度,默认 220。 */
  minWidth?: number
  /** 外部加载中(无 search 的一次性 load 场景由兼容层透传)。 */
  loading?: boolean
  /** 外部错误态(无 search 场景透传);配合 onRetry 展示重试按钮。 */
  error?: boolean
  /** 外部错误重试回调。 */
  onRetry?: () => void
  /** 覆盖结构性文案(缺省取 pages.pickers.common)。 */
  loadingText?: string
  emptyText?: string
  retryText?: string
}

const CLEAR_BTN =
  'inline-flex h-6 w-6 shrink-0 cursor-pointer items-center justify-center rounded-sm border-none bg-none text-[var(--shell-group-title)] hover:bg-[var(--shell-menu-hover-bg)] hover:text-[var(--shell-content-text)]'

const RETRY_BTN = 'shrink-0 cursor-pointer border-none bg-none p-0 text-[11px] font-medium text-[var(--color-text-link)] hover:underline'

export function SimplePicker(props: SimplePickerProps) {
  const {
    value, onChange, options, search, debounceMs, ariaLabel, placeholder,
    clearable, clearLabel, disabled, searchPlaceholder, searchAriaLabel, errorText,
    emptyLabel, pinnedOptions, minWidth, loading, error, onRetry,
  } = props
  const common = useT().pages.pickers.common
  const loadingText = props.loadingText ?? common.loading
  const emptyText = props.emptyText ?? common.empty
  const retryText = props.retryText ?? common.retry
  // hook 无条件调用:fetcher 为空时内部空转,满足条件渲染下的 hooks 顺序稳定。
  const srv = useServerPickerSearch({ fetcher: search, debounceMs })
  const staticOptions = useMemo(
    () => mergeOptions(
      emptyLabel ? [{ value: '', label: emptyLabel }] : undefined,
      pinnedOptions,
      options,
    ),
    [emptyLabel, pinnedOptions, options],
  )
  const clearBtn = canClearValue(clearable, disabled, value) && (
    <button
      type='button'
      className={CLEAR_BTN}
      aria-label={clearLabel ?? ariaLabel}
      title={clearLabel ?? ariaLabel}
      onClick={() => onChange('')}
    >
      <svg viewBox='0 0 24 24' width='14' height='14' fill='none' stroke='currentColor'
        strokeWidth='1.8' strokeLinecap='round' strokeLinejoin='round' aria-hidden>
        <path d='M18 6L6 18M6 6l12 12' />
      </svg>
    </button>
  )
  // 错误行渲染在触发器上方:浮层向下展开不会遮挡,失败时原因与重试按钮始终可点(W0-R2)。
  const errRow = (showError: boolean, retry?: () => void) => showError && (
    <div role='alert' className='flex items-center gap-2 text-[11px] text-[var(--color-danger)]'>
      <span>{errorText ?? ariaLabel}</span>
      {retry && <button type='button' className={RETRY_BTN} onClick={retry}>{retryText}</button>}
    </div>
  )
  if (search) {
    return (
      <div className='inline-flex flex-col gap-1' data-testid='simple-picker'>
        {errRow(srv.error, srv.retry)}
        <div className='inline-flex items-center gap-1'>
          <Dropdown
            value={value}
            options={mergeOptions(
              emptyLabel ? [{ value: '', label: emptyLabel }] : undefined,
              pinnedOptions,
              srv.items,
            )}
            onChange={onChange}
            ariaLabel={ariaLabel}
            placeholder={placeholder}
            disabled={disabled}
            searchable
            remote
            onKeywordChange={srv.setKeyword}
            searchPlaceholder={searchPlaceholder}
            searchAriaLabel={searchAriaLabel}
            loading={srv.loading}
            loadingText={loadingText}
            emptyText={srv.error ? (errorText ?? ariaLabel) : emptyText}
            triggerStyle={{ minWidth }}
          />
          {clearBtn}
        </div>
      </div>
    )
  }
  return (
    <div className='inline-flex flex-col gap-1' data-testid='simple-picker'>
      {errRow(!!error, onRetry)}
      <div className='inline-flex items-center gap-1'>
        <Dropdown
          value={value}
          options={staticOptions}
          onChange={onChange}
          ariaLabel={ariaLabel}
          placeholder={placeholder}
          disabled={disabled}
          searchable
          searchPlaceholder={searchPlaceholder}
          searchAriaLabel={searchAriaLabel}
          loading={loading}
          loadingText={loadingText}
          emptyText={error ? (errorText ?? ariaLabel) : emptyText}
          triggerStyle={{ minWidth }}
        />
        {clearBtn}
      </div>
    </div>
  )
}

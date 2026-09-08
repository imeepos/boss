// 通用自定义下拉:触发器按钮 + 浮层 listbox + 当前项打勾(antd Select 模式)。
// 替代原生 select 元素:系统渲染的 option 弹层无法定制,暗色主题下观感割裂。
// 键盘可达(W0 契约):箭头/Enter/Space 开合,上下移动跳过禁用项并循环回绕,
// Enter/Space 选定,Esc/Tab 关闭;aria:触发器 label、listbox/option 角色、
// aria-activedescendant。服务端检索 loading 行与空态文案由调用方透传;
// 已选值不在当前选项集时钉选回显(withPinnedValue),触发器不跌回占位文案。
import { useEffect, useId, useRef, useState, type CSSProperties, type KeyboardEvent as ReactKeyboardEvent } from 'react'
import { moveActive, resolveOptionMatch, withPinnedValue } from './pickers/pickerCore'

export interface DropdownOption {
  value: string
  label: string
  /** 置灰不可选(如权限不足);仍展示以保持选项可见性。 */
  disabled?: boolean
}

interface DropdownProps {
  value: string
  options: DropdownOption[]
  onChange: (value: string) => void
  ariaLabel: string
  disabled?: boolean
  triggerStyle?: CSSProperties
  /** 覆盖触发器按钮的默认 class(深色表面等场景)。 */
  buttonClassName?: string
  /** 浮层顶部渲染关键字过滤输入(按 label 大小写不敏感匹配)。 */
  searchable?: boolean
  searchPlaceholder?: string
  /** 过滤输入 aria 文案;缺省回退 searchPlaceholder,再回退 ariaLabel。 */
  searchAriaLabel?: string
  /** 远程检索模式:关闭本地过滤,关键字经 onKeywordChange 上抛由调用方请求服务端。 */
  remote?: boolean
  onKeywordChange?: (keyword: string) => void
  /** 服务端检索中:浮层顶部渲染 loading 行(role=status)。 */
  loading?: boolean
  /** loading 行文案(调用方 i18n)。 */
  loadingText?: string
  /** 无匹配选项时的空态文案;提供才渲染空态行(裸 Dropdown 调用方行为不变)。 */
  emptyText?: string
  /** 深色表面上使用(顶栏/页脚等常青藏青底):透明触发器 + 深色浮层,不随亮主题翻白。 */
  onDark?: boolean
  /** 值为空或无匹配项时触发器的占位文案;缺省回退 ariaLabel。 */
  placeholder?: string
  /** 浮层开合回调:仅在开合过渡时触发(mount 不触发)。服务端源调用方在 open=true 时复位检索状态。 */
  onOpenChange?: (open: boolean) => void
}

export function Dropdown({ value, options, onChange, ariaLabel, disabled, triggerStyle, searchable, searchPlaceholder, searchAriaLabel, remote, onKeywordChange, loading, loadingText, emptyText, onDark, placeholder, onOpenChange }: DropdownProps) {
  const [open, setOpen] = useState(false)
  const [keyword, setKeyword] = useState('')
  const [active, setActive] = useState(-1)
  const rootRef = useRef<HTMLDivElement>(null)
  const triggerRef = useRef<HTMLButtonElement>(null)
  const inputRef = useRef<HTMLInputElement>(null)
  const lbId = useId()
  // onOpenChange 经 ref 转发:回调身份变化不重挂监听,开合通知始终取最新闭包。
  const onOpenChangeRef = useRef(onOpenChange)
  onOpenChangeRef.current = onOpenChange

  useEffect(() => {
    if (!open || disabled) return
    const onDocClick = (e: MouseEvent) => {
      if (!rootRef.current?.contains(e.target as Node)) { setOpen(false); onOpenChangeRef.current?.(false) }
    }
    document.addEventListener('mousedown', onDocClick)
    return () => document.removeEventListener('mousedown', onDocClick)
  }, [open, disabled])

  // W1 裁定防御:存量调用把显示文案当 value 传时,label 同值兜底为有效选中项(勾选/高亮/
  // 键盘落点全按 effectiveValue,选定回写真实 value);双 miss 才合成钉选回显原值。
  // 两条兜底路径均 console.warn 留痕一次,便于各域 Phase B 逐处改传真实 value。
  const match = resolveOptionMatch(options, value)
  const effectiveValue = match.effectiveValue
  const fullOptions = match.hit ? options : withPinnedValue(options, value)
  const current = match.hit ?? fullOptions.find((o) => o.value === value)
  const warnedRef = useRef<Set<string>>(new Set())
  useEffect(() => {
    if (value === '' || !match.hit || match.hit.value === value) return
    const key = (match.hit.value === value ? 'v' : 'l') + ':' + value
    if (warnedRef.current.has(key)) return
    warnedRef.current.add(key)
    console.warn('[Dropdown] value 不在 options 值域:' + value + (match.hit.value === value ? '' : '(已按 label 同值兜底回显)') + ';请改传真实 value(见 picker-guide W1 裁定)')
    // match 每渲染重建,去重交给 warnedRef。
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [value])
  const kw = keyword.trim().toLowerCase()
  const visible = searchable && !remote && kw !== '' ? fullOptions.filter((o) => o.label.toLowerCase().includes(kw)) : fullOptions

  // 打开时:活动项落在已选项(无则首项);searchable 时焦点直达过滤输入。关闭即复位。
  useEffect(() => {
    if (!open) { setActive(-1); return }
    const idx = visible.findIndex((o) => o.value === effectiveValue)
    setActive(idx >= 0 ? idx : 0)
    if (searchable) inputRef.current?.focus()
    // visible/value 仅用于开合瞬间的落点计算,不作为重放依赖。
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [open])

  // 活动项滚动入可视区(键盘导航跟随)。
  useEffect(() => {
    if (active < 0) return
    document.getElementById(lbId + '-opt-' + active)?.scrollIntoView({ block: 'nearest' })
  }, [active, lbId])

  const close = (focusBack: boolean) => {
    setOpen(false)
    onOpenChangeRef.current?.(false)
    if (focusBack) triggerRef.current?.focus()
  }
  const openPanel = () => {
    setKeyword('')
    setOpen(true)
    onOpenChangeRef.current?.(true)
  }
  const toggle = () => {
    if (disabled) return
    if (open) { close(false); return }
    openPanel()
  }
  const step = (delta: 1 | -1) => setActive((prev) => moveActive(visible.length, prev, delta, (i) => !!visible[i]?.disabled))
  const commitActive = () => {
    const o = visible[active]
    if (!o || o.disabled) return
    onChange(o.value)
    close(true)
  }
  // 面板内(触发器或过滤输入)按键:移动/选定/关闭。
  const onPanelKey = (e: ReactKeyboardEvent<HTMLElement>) => {
    if (e.key === 'ArrowDown') { e.preventDefault(); step(1) }
    else if (e.key === 'ArrowUp') { e.preventDefault(); step(-1) }
    else if (e.key === 'Enter' || e.key === ' ') { e.preventDefault(); commitActive() }
    else if (e.key === 'Escape') { e.preventDefault(); e.stopPropagation(); close(true) }
    else if (e.key === 'Tab') close(false)
  }
  const onTriggerKey = (e: ReactKeyboardEvent<HTMLButtonElement>) => {
    if (disabled) return
    if (!open) {
      if (e.key === 'ArrowDown' || e.key === 'ArrowUp' || e.key === 'Enter' || e.key === ' ') {
        e.preventDefault()
        openPanel()
      }
      return
    }
    onPanelKey(e)
  }

  const activeId = active >= 0 && active < visible.length ? lbId + '-opt-' + active : undefined
  return (
    <div className='relative inline-flex' ref={rootRef} style={triggerStyle}>
      <button
        ref={triggerRef}
        type='button'
        className={'flex h-8 w-full items-center justify-between gap-2 rounded-sm border py-0 pr-1 pl-2.5 text-[13px]' + (disabled
          ? ' cursor-not-allowed border-[var(--shell-input-border)] bg-[var(--shell-input-disabled-bg)] text-[var(--shell-input-placeholder)]'
          : onDark
            ? ' cursor-pointer border-white/20 bg-white/5 text-white hover:border-white/40 focus-visible:border-white/60'
            : ' cursor-pointer border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] text-[var(--shell-content-text)] hover:border-[var(--color-border-focus)] focus-visible:border-[var(--color-border-focus)]')}
        onClick={toggle}
        onKeyDown={onTriggerKey}
        aria-haspopup='listbox'
        aria-expanded={open}
        aria-label={ariaLabel}
        disabled={disabled}
      >
        <span className='truncate whitespace-nowrap'>{current?.label ?? placeholder ?? ariaLabel}</span>
        <span className={'inline-flex text-[var(--shell-group-title)] transition-transform duration-150' + (open ? ' rotate-180' : '')} aria-hidden>
          <svg viewBox='0 0 24 24' width='14' height='14' fill='none' stroke='currentColor'
            strokeWidth='1.8' strokeLinecap='round' strokeLinejoin='round'>
            <path d='M6 9l6 6 6-6' />
          </svg>
        </span>
      </button>
      {open && (
        <div
          className={'absolute left-0 top-[calc(100%+6px)] z-popover max-h-[264px] min-w-full overflow-y-auto rounded-md border p-1 ' + (onDark
            ? 'border-white/10 bg-[var(--color-brand-navy-900)] shadow-[0_12px_32px_rgba(0,0,0,0.4)]'
            : 'border-[var(--shell-side-border)] bg-[var(--shell-card-bg)] shadow-[0_6px_16px_rgba(0,0,0,0.08)]')}
          role='listbox'
          id={lbId}
          aria-label={ariaLabel}
          aria-activedescendant={activeId}
          onMouseDown={(e) => e.stopPropagation()}
        >
          {searchable && (
            <div className='border-b border-[var(--shell-side-border)] px-1 pb-1'>
              <input
                ref={inputRef}
                type='text'
                value={keyword}
                onChange={(e) => { setKeyword(e.target.value); onKeywordChange?.(e.target.value) }}
                onKeyDown={onPanelKey}
                placeholder={searchPlaceholder}
                aria-label={searchAriaLabel ?? searchPlaceholder ?? ariaLabel}
                className='h-7 w-full rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-2 text-[12px] text-[var(--shell-content-text)] outline-none placeholder:text-[var(--shell-input-placeholder)] focus:border-[var(--color-border-focus)]'
              />
            </div>
          )}
          {loading && (
            <div role='status' aria-live='polite' className='flex items-center gap-2 px-2.5 py-1.5 text-[12px] text-[var(--shell-group-title)]'>
              <span className='h-3 w-3 shrink-0 animate-spin rounded-full border-[1.5px] border-[var(--shell-input-border)] border-t-transparent' aria-hidden />
              {loadingText}
            </div>
          )}
          {!loading && emptyText !== undefined && visible.length === 0 && (
            <div className='px-2.5 py-2 text-[12px] text-[var(--shell-input-placeholder)]'>{emptyText}</div>
          )}
          {visible.map((o, i) => {
            const activeCls = i === active && !o.disabled ? ' ' + (onDark ? 'bg-white/10' : 'bg-[var(--shell-menu-hover-bg)]') : ''
            const optCls = 'flex w-full items-center justify-between gap-4 rounded-sm border-none bg-none px-2.5 py-1.5 text-left text-[13px] whitespace-nowrap '
              + (o.disabled
                ? 'cursor-not-allowed text-[var(--shell-input-placeholder)]'
                : 'cursor-pointer ' + (onDark
                  ? 'text-white hover:bg-white/10' + (o.value === effectiveValue ? ' font-semibold text-[var(--color-brand-gold-500)]' : '')
                  : 'text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]' + (o.value === effectiveValue ? ' font-semibold text-[var(--shell-fab-bg)]' : '')))
              + activeCls
            return (
              <button
                key={o.value}
                id={lbId + '-opt-' + i}
                type='button'
                role='option'
                aria-selected={o.value === effectiveValue}
                aria-disabled={o.disabled || undefined}
                className={optCls}
                onMouseEnter={() => setActive(i)}
                onMouseDown={(e) => {
                  e.stopPropagation()
                  e.preventDefault()
                  if (o.disabled) return
                  onChange(o.value)
                  close(false)
                }}
                onClick={(e) => e.preventDefault()}
              >
                <span className='truncate'>{o.label}</span>
                <span className='min-w-[14px] text-right' aria-hidden>{o.value === effectiveValue && (
                  <svg viewBox='0 0 24 24' width='14' height='14' fill='none' stroke='currentColor'
                    strokeWidth='2' strokeLinecap='round' strokeLinejoin='round'>
                    <path d='M5 12.5l4.5 4.5L19 7.5' />
                  </svg>
                )}</span>
              </button>
            )
          })}
        </div>
      )}
    </div>
  )
}

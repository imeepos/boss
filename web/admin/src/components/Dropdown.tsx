// 通用自定义下拉:触发器按钮 + 浮层 listbox + 当前项打勾(antd Select 模式)。
// 替代原生 <select>:系统渲染的 option 弹层无法定制,暗色主题下观感割裂。
// 样式:tailwind 原子类(原 Dropdown.css 已删除),令牌走 shell-* 体系。
import { useEffect, useRef, useState, type CSSProperties } from 'react'

export interface DropdownOption {
  value: string
  label: string
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
  /** 远程检索模式:关闭本地过滤,关键字经 onKeywordChange 上抛由调用方请求服务端。 */
  remote?: boolean
  onKeywordChange?: (keyword: string) => void
  /** 深色表面上使用(顶栏/页脚等常青藏青底):透明触发器 + 深色浮层,不随亮主题翻白。 */
  onDark?: boolean
}

export function Dropdown({ value, options, onChange, ariaLabel, disabled, triggerStyle, searchable, searchPlaceholder, remote, onKeywordChange, onDark }: DropdownProps) {
  const [open, setOpen] = useState(false)
  const [keyword, setKeyword] = useState('')
  const rootRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    if (!open || disabled) return
    const onDocClick = (e: MouseEvent) => {
      if (!rootRef.current?.contains(e.target as Node)) setOpen(false)
    }
    document.addEventListener('mousedown', onDocClick)
    return () => document.removeEventListener('mousedown', onDocClick)
  }, [open, disabled])

  const current = options.find((o) => o.value === value)
  const kw = keyword.trim().toLowerCase()
  const visible = searchable && !remote && kw !== '' ? options.filter((o) => o.label.toLowerCase().includes(kw)) : options
  return (
    <div className="relative inline-flex" ref={rootRef} style={triggerStyle}>
      <button
        type="button"
        className={'flex h-8 w-full items-center justify-between gap-2 rounded-sm border py-0 pr-1 pl-2.5 text-[13px]' + (disabled
          ? ' cursor-not-allowed border-[var(--shell-input-border)] bg-[var(--shell-input-disabled-bg)] text-[var(--shell-input-placeholder)]'
          : onDark
            ? ' cursor-pointer border-white/20 bg-white/5 text-white hover:border-white/40 focus-visible:border-white/60'
            : ' cursor-pointer border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] text-[var(--shell-content-text)] hover:border-[var(--color-border-focus)] focus-visible:border-[var(--color-border-focus)]')}
        onClick={() => { if (!disabled) { setOpen((v) => !v); if (!open) setKeyword('') } }}
        aria-haspopup="listbox"
        aria-expanded={open}
        aria-label={ariaLabel}
        disabled={disabled}
      >
        <span className="truncate whitespace-nowrap">{current?.label ?? ariaLabel}</span>
        <span className={`inline-flex text-[var(--shell-group-title)] transition-transform duration-150${open ? ' rotate-180' : ''}`} aria-hidden>
          <svg viewBox="0 0 24 24" width="14" height="14" fill="none" stroke="currentColor"
            strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round">
            <path d="M6 9l6 6 6-6" />
          </svg>
        </span>
      </button>
      {open && (
        <div
          className={'absolute left-0 top-[calc(100%+6px)] z-[1000] max-h-[264px] min-w-full overflow-y-auto rounded-md border p-1 ' + (onDark
            ? 'border-white/10 bg-[var(--color-brand-navy-900)] shadow-[0_12px_32px_rgba(0,0,0,0.4)]'
            : 'border-[var(--shell-side-border)] bg-[var(--shell-card-bg)] shadow-[0_6px_16px_rgba(0,0,0,0.08)]')}
          role="listbox"
          aria-label={ariaLabel}
          onMouseDown={(e) => e.stopPropagation()}
        >
          {searchable && (
            <div className="border-b border-[var(--shell-side-border)] px-1 pb-1">
              <input
                type="text"
                value={keyword}
                onChange={(e) => { setKeyword(e.target.value); onKeywordChange?.(e.target.value) }}
                placeholder={searchPlaceholder}
                className="h-7 w-full rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-2 text-[12px] text-[var(--shell-content-text)] outline-none placeholder:text-[var(--shell-input-placeholder)] focus:border-[var(--color-border-focus)]"
              />
            </div>
          )}
          {visible.map((o) => (
            <button
              key={o.value}
              type="button"
              role="option"
              aria-selected={o.value === value}
              className={'flex w-full cursor-pointer items-center justify-between gap-4 rounded-sm border-none bg-none px-2.5 py-1.5 text-left text-[13px] whitespace-nowrap ' + (onDark
                ? 'text-white hover:bg-white/10' + (o.value === value ? ' font-semibold text-[var(--color-brand-gold-500)]' : '')
                : 'text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]' + (o.value === value ? ' font-semibold text-[var(--shell-fab-bg)]' : ''))}
              onMouseDown={(e) => {
                e.stopPropagation()
                e.preventDefault()
                onChange(o.value)
                setOpen(false)
              }}
              onClick={(e) => e.preventDefault()}
            >
              <span className="truncate">{o.label}</span>
              <span className="min-w-[14px] text-right" aria-hidden>{o.value === value && (
                <svg viewBox="0 0 24 24" width="14" height="14" fill="none" stroke="currentColor"
                  strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                  <path d="M5 12.5l4.5 4.5L19 7.5" />
                </svg>
              )}</span>
            </button>
          ))}
        </div>
      )}
    </div>
  )
}

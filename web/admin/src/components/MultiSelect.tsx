// 通用多选下拉:触发器 chips + 浮层 listbox 勾选(点选不收起),视觉沿用 Dropdown 令牌。
// 场景:开放平台订阅事件选择器等"一主体多选项"表单;选项由调用方给,本地关键字过滤。
import { useEffect, useRef, useState } from 'react'
import type { DropdownOption } from './Dropdown'

/** MultiOption 多选项:可附一行说明文字(浮层内弱化展示)。 */
export interface MultiOption extends DropdownOption {
  description?: string
}

/** toggleValue 勾选/取消勾选一个值(保序)。 */
export function toggleValue(values: string[], v: string): string[] {
  return values.includes(v) ? values.filter((x) => x !== v) : [...values, v]
}

/** filterOptions 按 label 大小写不敏感过滤。 */
export function filterOptions(options: MultiOption[], keyword: string): MultiOption[] {
  const kw = keyword.trim().toLowerCase()
  if (kw === '') return options
  return options.filter((o) => o.label.toLowerCase().includes(kw))
}

interface MultiSelectProps {
  values: string[]
  options: MultiOption[]
  onChange: (values: string[]) => void
  ariaLabel: string
  /** 未选中时的占位文案,缺省用 ariaLabel。 */
  placeholder?: string
  searchPlaceholder?: string
  /** 浮层内无匹配选项时的提示文案。 */
  emptyText?: string
  disabled?: boolean
}

export function MultiSelect({ values, options, onChange, ariaLabel, placeholder, searchPlaceholder, emptyText, disabled }: MultiSelectProps) {
  const [open, setOpen] = useState(false)
  const [keyword, setKeyword] = useState('')
  const rootRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    if (!open || disabled) return
    const onDocClick = (e: MouseEvent) => {
      if (!rootRef.current?.contains(e.target as Node)) setOpen(false)
    }
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') setOpen(false)
    }
    document.addEventListener('mousedown', onDocClick)
    document.addEventListener('keydown', onKey)
    return () => {
      document.removeEventListener('mousedown', onDocClick)
      document.removeEventListener('keydown', onKey)
    }
  }, [open, disabled])

  const labelOf = (v: string) => options.find((o) => o.value === v)?.label ?? v
  const visible = filterOptions(options, keyword)
  const triggerCls = 'flex min-h-8 w-full cursor-pointer flex-wrap items-center gap-1 rounded-sm border py-1 pr-1 pl-2.5 text-left text-[13px] ' + (disabled
    ? 'cursor-not-allowed border-[var(--shell-input-border)] bg-[var(--shell-input-disabled-bg)]'
    : 'border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] text-[var(--shell-content-text)] hover:border-[var(--color-border-focus)] focus-visible:border-[var(--color-border-focus)]')

  return (
    <div className="relative inline-flex" ref={rootRef}>
      <button type="button" className={triggerCls} onClick={() => { if (!disabled) { setOpen((v) => !v); if (!open) setKeyword('') } }}
        aria-haspopup="listbox" aria-expanded={open} aria-label={ariaLabel} disabled={disabled}>
        {values.length === 0 ? (
          <span className="py-0.5 text-[var(--shell-input-placeholder)]">{placeholder ?? ariaLabel}</span>
        ) : values.map((v) => (
          <span key={v} className="inline-flex items-center gap-1 rounded-sm bg-[var(--shell-menu-hover-bg)] py-0.5 pr-1 pl-1.5 font-mono text-xs text-[var(--shell-content-text)]">
            {labelOf(v)}
            {!disabled && (
              <span role="button" aria-label={`remove ${v}`}
                className="cursor-pointer text-[var(--shell-group-title)] hover:text-[var(--color-danger)]"
                onMouseDown={(e) => { e.stopPropagation(); e.preventDefault() }}
                onClick={(e) => { e.stopPropagation(); onChange(values.filter((x) => x !== v)) }}>
                <svg viewBox="0 0 24 24" width="11" height="11" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round">
                  <path d="M6 6l12 12M18 6L6 18" />
                </svg>
              </span>
            )}
          </span>
        ))}
        <span className={`ml-auto inline-flex shrink-0 text-[var(--shell-group-title)] transition-transform duration-150${open ? ' rotate-180' : ''}`} aria-hidden>
          <svg viewBox="0 0 24 24" width="14" height="14" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round">
            <path d="M6 9l6 6 6-6" />
          </svg>
        </span>
      </button>
      {open && (
        <div className="absolute left-0 top-[calc(100%+6px)] z-[1000] max-h-[264px] min-w-full overflow-y-auto rounded-md border border-[var(--shell-side-border)] bg-[var(--shell-card-bg)] p-1 shadow-[0_6px_16px_rgba(0,0,0,0.08)]"
          role="listbox" aria-label={ariaLabel} aria-multiselectable="true" onMouseDown={(e) => e.stopPropagation()}>
          <div className="sticky top-0 border-b border-[var(--shell-side-border)] bg-[var(--shell-card-bg)] px-1 pb-1">
            <input type="text" value={keyword} onChange={(e) => setKeyword(e.target.value)} placeholder={searchPlaceholder}
              className="h-7 w-full rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-2 text-[12px] text-[var(--shell-content-text)] outline-none placeholder:text-[var(--shell-input-placeholder)] focus:border-[var(--color-border-focus)]" />
          </div>
          {visible.length === 0 && <div className="px-2.5 py-2 text-[12px] text-[var(--shell-input-placeholder)]">{emptyText ?? '·'}</div>}
          {visible.map((o) => {
            const on = values.includes(o.value)
            return (
              <button key={o.value} type="button" role="option" aria-selected={on} aria-disabled={o.disabled || undefined}
                className={'flex w-full items-start justify-between gap-3 rounded-sm border-none bg-none px-2.5 py-1.5 text-left text-[13px] whitespace-nowrap ' + (o.disabled
                  ? 'cursor-not-allowed text-[var(--shell-input-placeholder)]'
                  : 'cursor-pointer text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)]')}
                onMouseDown={(e) => { e.stopPropagation(); e.preventDefault(); if (!o.disabled) onChange(toggleValue(values, o.value)) }}
                onClick={(e) => e.preventDefault()}>
                <span className="min-w-0">
                  <span className="block font-mono text-xs">{o.label}</span>
                  {o.description && <span className="mt-0.5 block text-[11px] whitespace-normal text-[var(--shell-crumb-text)]">{o.description}</span>}
                </span>
                <span className={'mt-0.5 inline-flex h-3.5 w-3.5 shrink-0 items-center justify-center rounded-[3px] border ' + (on
                  ? 'border-[var(--shell-fab-bg)] bg-[var(--shell-fab-bg)] text-[var(--shell-fab-icon)]'
                  : 'border-[var(--shell-input-border)]')} aria-hidden>
                  {on && (
                    <svg viewBox="0 0 24 24" width="10" height="10" fill="none" stroke="currentColor" strokeWidth="3" strokeLinecap="round" strokeLinejoin="round">
                      <path d="M5 12.5l4.5 4.5L19 7.5" />
                    </svg>
                  )}
                </span>
              </button>
            )
          })}
        </div>
      )}
    </div>
  )
}

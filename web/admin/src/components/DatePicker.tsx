// 通用自定义日期选择器:触发器按钮 + 浮层日历(antd DatePicker 模式)。
// 替代原生 <input type="date">:系统渲染的日历弹层无法定制,主题下观感割裂。
// 值为 ISO yyyy-MM-dd 字符串,空串表示未选;样式令牌与 Dropdown 一致(shell-* 体系)。
import { useEffect, useMemo, useRef, useState } from 'react'

interface DatePickerProps {
  value: string
  onChange: (value: string) => void
  ariaLabel: string
  placeholder?: string
  disabled?: boolean
  /** 浮层文案:月份/上一月/下一月/今天/清除/周标题(周日起始)。 */
  labels: DatePickerLabels
}

export interface DatePickerLabels {
  prevMonth: string
  nextMonth: string
  today: string
  clear: string
  weekdays: string[]
}

const WEEKDAYS_FALLBACK = ['日', '一', '二', '三', '四', '五', '六']

function pad(n: number) {
  return String(n).padStart(2, '0')
}

export function toISO(d: Date) {
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}`
}

function parseISO(v: string): Date | null {
  const m = /^(\d{4})-(\d{2})-(\d{2})$/.exec(v)
  if (!m) return null
  return new Date(Number(m[1]), Number(m[2]) - 1, Number(m[3]))
}

function monthKey(d: Date) {
  return d.getFullYear() * 12 + d.getMonth()
}

export function DatePicker({ value, onChange, ariaLabel, placeholder, disabled, labels }: DatePickerProps) {
  const weekdays = labels.weekdays.length === 7 ? labels.weekdays : WEEKDAYS_FALLBACK
  const [open, setOpen] = useState(false)
  // 视口月份:默认跟随已选日期,否则今天。
  const [view, setView] = useState<Date>(() => parseISO(value) ?? new Date())
  const rootRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    if (!open || disabled) return
    const onDocClick = (e: MouseEvent) => {
      if (!rootRef.current?.contains(e.target as Node)) setOpen(false)
    }
    document.addEventListener('mousedown', onDocClick)
    return () => document.removeEventListener('mousedown', onDocClick)
  }, [open, disabled])

  useEffect(() => {
    if (open) setView(parseISO(value) ?? new Date())
  }, [open]) // eslint-disable-line react-hooks/exhaustive-deps

  const grid = useMemo(() => {
    // 周日起始,前置补位到 6 周,保证月份切换时高度稳定。
    const first = new Date(view.getFullYear(), view.getMonth(), 1)
    const start = new Date(first)
    start.setDate(1 - first.getDay())
    return Array.from({ length: 42 }, (_, i) => {
      const d = new Date(start)
      d.setDate(start.getDate() + i)
      return d
    })
  }, [view])

  const shiftMonth = (delta: number) => {
    setView((v) => new Date(v.getFullYear(), v.getMonth() + delta, 1))
  }

  const todayISO = toISO(new Date())
  const title = `${view.getFullYear()}年 ${view.getMonth() + 1}月`

  return (
    <div className="relative inline-flex" ref={rootRef}>
      <button
        type="button"
        className={'flex h-8 w-full min-w-[124px] items-center justify-between gap-2 rounded-sm border py-0 pr-1 pl-2.5 text-[13px]' + (disabled ? ' cursor-not-allowed border-[var(--shell-input-border)] bg-[var(--shell-input-disabled-bg)] text-[var(--shell-input-placeholder)]' : ' cursor-pointer border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] text-[var(--shell-content-text)] hover:border-[var(--color-border-focus)] focus-visible:border-[var(--color-border-focus)]')}
        onClick={() => { if (!disabled) setOpen((v) => !v) }}
        aria-haspopup="dialog"
        aria-expanded={open}
        aria-label={ariaLabel}
        disabled={disabled}
      >
        <span className="truncate whitespace-nowrap">{value || placeholder || ariaLabel}</span>
        <span className="inline-flex shrink-0 text-[var(--shell-group-title)]" aria-hidden>
          <svg viewBox="0 0 24 24" width="14" height="14" fill="none" stroke="currentColor"
            strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round">
            <rect x="3" y="5" width="18" height="16" rx="2" />
            <path d="M16 3v4M8 3v4M3 11h18" />
          </svg>
        </span>
      </button>
      {open && (
        <div
          className="absolute left-0 top-[calc(100%+6px)] z-[1000] w-[252px] rounded-md border border-[var(--shell-side-border)] bg-[var(--shell-card-bg)] p-2 shadow-[0_6px_16px_rgba(0,0,0,0.08)]"
          role="dialog"
          aria-label={ariaLabel}
          onMouseDown={(e) => e.stopPropagation()}
        >
          <div className="mb-1 flex items-center justify-between px-1">
            <NavBtn label={labels.prevMonth} onClick={() => shiftMonth(-1)} dir="prev" />
            <span className="text-[13px] font-semibold text-[var(--shell-content-text)]">{title}</span>
            <NavBtn label={labels.nextMonth} onClick={() => shiftMonth(1)} dir="next" />
          </div>
          <div className="grid grid-cols-7">
            {weekdays.map((w, i) => (
              <div key={i} className="py-1 text-center text-[12px] text-[var(--shell-group-title)]" aria-hidden>{w}</div>
            ))}
            {grid.map((d) => {
              const iso = toISO(d)
              const inMonth = monthKey(d) === monthKey(view)
              const selected = iso === value
              const isToday = iso === todayISO
              return (
                <button
                  key={iso}
                  type="button"
                  aria-pressed={selected}
                  aria-label={iso}
                  className={'h-8 cursor-pointer rounded-sm border-none bg-none p-0 text-[13px] transition-colors hover:bg-[var(--shell-menu-hover-bg)]'
                    + (inMonth ? ' text-[var(--shell-content-text)]' : ' text-[var(--shell-input-placeholder)] opacity-60')
                    + (isToday && !selected ? ' font-semibold text-[var(--shell-fab-bg)]' : '')
                    + (selected ? ' bg-[var(--shell-fab-bg)] font-semibold text-white hover:bg-[var(--shell-fab-bg)]' : '')}
                  onMouseDown={(e) => { e.stopPropagation(); e.preventDefault() }}
                  onClick={() => { onChange(iso); setOpen(false) }}
                >
                  {d.getDate()}
                </button>
              )
            })}
          </div>
          <div className="mt-1 flex items-center justify-between border-t border-[var(--shell-side-border)] pt-1">
            <button
              type="button"
              className="cursor-pointer border-none bg-none px-2 py-1 text-[12px] text-[var(--shell-fab-bg)] hover:underline"
              onMouseDown={(e) => e.stopPropagation()}
              onClick={() => { onChange(todayISO); setOpen(false) }}
            >{labels.today}</button>
            <button
              type="button"
              className="cursor-pointer border-none bg-none px-2 py-1 text-[12px] text-[var(--shell-group-title)] hover:underline"
              onMouseDown={(e) => e.stopPropagation()}
              onClick={() => { onChange(''); setOpen(false) }}
            >{labels.clear}</button>
          </div>
        </div>
      )}
    </div>
  )
}

function NavBtn({ label, onClick, dir }: { label: string; onClick: () => void; dir: 'prev' | 'next' }) {
  return (
    <button
      type="button"
      aria-label={label}
      className="flex h-6 w-6 cursor-pointer items-center justify-center rounded-sm border-none bg-none p-0 text-[var(--shell-group-title)] hover:bg-[var(--shell-menu-hover-bg)] hover:text-[var(--shell-content-text)]"
      onMouseDown={(e) => e.stopPropagation()}
      onClick={onClick}
    >
      <svg viewBox="0 0 24 24" width="14" height="14" fill="none" stroke="currentColor"
        strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round">
        {dir === 'prev' ? <path d="M15 6l-6 6 6 6" /> : <path d="M9 6l6 6-6 6" />}
      </svg>
    </button>
  )
}

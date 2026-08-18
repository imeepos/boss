// 通用自定义下拉:触发器按钮 + 浮层 listbox + 当前项打勾(antd Select 模式)。
// 替代原生 <select>:系统渲染的 option 弹层无法定制,暗色主题下观感割裂。
import { useEffect, useRef, useState, type CSSProperties } from 'react'
import './Dropdown.css'

export interface DropdownOption {
  value: string
  label: string
}

interface DropdownProps {
  value: string
  options: DropdownOption[]
  onChange: (value: string) => void
  ariaLabel: string
  triggerStyle?: CSSProperties
}

export function Dropdown({ value, options, onChange, ariaLabel, triggerStyle }: DropdownProps) {
  const [open, setOpen] = useState(false)
  const rootRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    if (!open) return
    const onDocClick = (e: MouseEvent) => {
      if (!rootRef.current?.contains(e.target as Node)) setOpen(false)
    }
    document.addEventListener('mousedown', onDocClick)
    return () => document.removeEventListener('mousedown', onDocClick)
  }, [open])

  const current = options.find((o) => o.value === value)
  return (
    <div className="dd" ref={rootRef} style={triggerStyle}>
      <button
        type="button"
        className="dd-trigger"
        onClick={() => setOpen((v) => !v)}
        aria-haspopup="listbox"
        aria-expanded={open}
        aria-label={ariaLabel}
      >
        <span className="dd-trigger-label">{current?.label ?? ariaLabel}</span>
        <span className={`dd-caret${open ? ' open' : ''}`} aria-hidden>
          <svg viewBox="0 0 24 24" width="14" height="14" fill="none" stroke="currentColor"
            strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round">
            <path d="M6 9l6 6 6-6" />
          </svg>
        </span>
      </button>
      {open && (
        <div className="dd-menu" role="listbox" aria-label={ariaLabel}>
          {options.map((o) => (
            <button
              key={o.value}
              type="button"
              role="option"
              aria-selected={o.value === value}
              className={o.value === value ? 'active' : ''}
              onClick={() => {
                onChange(o.value)
                setOpen(false)
              }}
            >
              <span className="dd-option-label">{o.label}</span>
              <span className="dd-check" aria-hidden>{o.value === value && (
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

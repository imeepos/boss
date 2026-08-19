// 自研 Button：替代 antd Button，支持 primary/default/danger 变体 + loading 态
import { type ButtonHTMLAttributes, type ReactNode } from 'react'

type Variant = 'primary' | 'default' | 'danger' | 'ghost'
type Size = 'sm' | 'md' | 'lg'

interface ButtonProps extends ButtonHTMLAttributes<HTMLButtonElement> {
  variant?: Variant
  size?: Size
  loading?: boolean
  icon?: ReactNode
  children?: ReactNode
}

const variantClass: Record<Variant, string> = {
  primary:
    'bg-[var(--color-brand-blue-700)] text-white hover:bg-[var(--color-brand-blue-800)] active:bg-[var(--color-brand-blue-900)] border-transparent',
  default:
    'bg-[var(--shell-card-bg)] text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)] border-[var(--shell-input-border)] hover:border-[var(--shell-input-border-hover)]',
  danger:
    'bg-[var(--color-danger)] text-white hover:opacity-90 active:opacity-80 border-transparent',
  ghost:
    'bg-transparent text-[var(--shell-content-text)] hover:bg-[var(--shell-menu-hover-bg)] border-transparent',
}

const sizeClass: Record<Size, string> = {
  sm: 'px-2.5 py-1 text-xs gap-1',
  md: 'px-4 py-1.5 text-sm gap-1.5',
  lg: 'px-6 py-2 text-base gap-2',
}

export function Button({
  variant = 'default',
  size = 'md',
  loading = false,
  icon,
  disabled,
  children,
  className = '',
  ...rest
}: ButtonProps) {
  const base =
    'inline-flex items-center justify-center border rounded-md font-medium transition-colors duration-150 cursor-pointer disabled:opacity-50 disabled:cursor-not-allowed focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[var(--shell-input-border-focus)]'
  return (
    <button
      className={`${base} ${variantClass[variant]} ${sizeClass[size]} ${className}`}
      disabled={disabled || loading}
      {...rest}
    >
      {loading ? (
        <svg
          className="animate-spin -ml-1 h-4 w-4"
          xmlns="http://www.w3.org/2000/svg"
          fill="none"
          viewBox="0 0 24 24"
        >
          <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" />
          <path
            className="opacity-75"
            fill="currentColor"
            d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"
          />
        </svg>
      ) : icon ? (
        <span className="inline-flex">{icon}</span>
      ) : null}
      {children}
    </button>
  )
}
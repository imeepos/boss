// Select: custom select dropdown, replaces .h-8 rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-2.5 pr-6 text-[13px] text-[var(--shell-content-text)] outline-none focus:border-[var(--color-border-focus)]
import { type SelectHTMLAttributes, forwardRef } from 'react'
import { cn } from '../../lib/cn'

export const Select = forwardRef<HTMLSelectElement, SelectHTMLAttributes<HTMLSelectElement>>(
  ({ className, children, ...props }, ref) => (
    <select
      ref={ref}
      className={cn(
        'flex h-8 rounded-sm border border-[var(--shell-input-border)] bg-[var(--shell-input-bg)] px-2.5 py-1 text-xs text-[var(--shell-input-text)] outline-none focus:border-[var(--shell-input-border-focus)] disabled:cursor-not-allowed disabled:bg-[var(--shell-input-disabled-bg)]',
        className,
      )}
      {...props}
    >
      {children}
    </select>
  ),
)
Select.displayName = 'Select'